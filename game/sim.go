package game

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/engine/rpg2d"
	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/rpg2d/quad"
	"github.com/ghthor/engine/sim/stime"
)

type actorIndex map[int64]*actor

type inputPhase struct {
	index actorIndex
}
type narrowPhase struct {
	index actorIndex
}

func (phase inputPhase) ApplyInputsIn(c quad.Chunk, now stime.Time) quad.Chunk {
	for _, e := range c.Entities {
		switch a := e.(type) {
		case *actorEntity:
			actor := phase.index[a.Id()]

			// Remove any movement actions that have completed
			if actor.pathAction != nil && actor.pathAction.End() <= now {
				actor.lastMoveAction = actor.pathAction
				actor.cell = actor.pathAction.Dest
				actor.pathAction = nil
			}

			cmdReq := actor.ReadCmdRequest()

			if cmdReq.moveRequest == nil {
				// The client has canceled all move requests
				actor.actorCmdRequest.moveRequest = nil
				continue
			} else {
				// The client has a standing move request
				actor.actorCmdRequest.moveRequest = cmdReq.moveRequest
			}

			// Actor is already moving so we can't accept a new movement request
			if actor.pathAction != nil {
				continue
			}

			dest := actor.Cell().Neighbor(cmdReq.Direction)
			direction := actor.Cell().DirectionTo(dest)

			// If the last MoveAction was a PathAction that ended on this step
			// And the direction was the same as the requested movement
			if pathAction, ok := actor.lastMoveAction.(*coord.PathAction); (ok && pathAction.End() == now) || actor.facing == direction {
				pathAction = &coord.PathAction{
					// now+speed
					stime.NewSpan(now, now+stime.Time(20)),
					actor.Cell(),
					dest,
				}

				if pathAction.CanHappenAfter(actor.lastMoveAction) {
					// actor.ApplyAction(pathAction)
					actor.pathAction = pathAction
					actor.facing = pathAction.Direction()
					actor.actorCmdRequest.moveRequest = nil
				}

				// If the requested direction is different from
				// the actor's current facing.
			} else if actor.facing != direction {
				turnAction := coord.TurnAction{
					actor.facing,
					direction,
					now,
				}

				if turnAction.CanHappenAfter(actor.lastMoveAction) {
					// actor.ApplyAction(turnAction
					actor.facing = direction
					actor.lastMoveAction = turnAction
					actor.actorCmdRequest.moveRequest = nil
				}
			}

		default:
			panic(fmt.Sprint("unexpected entity type:", a))
		}
	}
	return c
}

func (narrowPhase) ResolveCollisions(c quad.CollisionGroup, now stime.Time) quad.CollisionGroup {
	return c
}

type indexHandler struct {
	tmpl     *template.Template
	settings ClientSettings
}

func (index indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := index.tmpl.Execute(w, index.settings)

	if err != nil {
		http.Error(w, fmt.Sprint("template error:", err), http.StatusInternalServerError)
	}
}

// A set of data that the index page template will
// have access to when it is executed.
type ClientSettings struct {
	JsMain       string
	WebsocketURL string
	Simulation   SimulationSettings
}

// Part of the ClientSettings that are rpg2d.Simulation specific
type SimulationSettings struct {
	Width, Height int
}

type ShardConfig struct {
	// Is the resultant http server going to be
	// run using TLS or just standard HTTP.
	IsHTTPS bool

	// LAddr for the http server
	LAddr,

	// Path to javascript directory
	JsDir,

	// The javascript module that require.js should
	// call as the javascript main.
	JsMain,

	// Path to the graphic asset directory
	AssetDir,

	// Path to the css directory
	CssDir string

	// A template for the index page. The template
	// will be executed with a game.ClientSettings{} struct.
	IndexTmpl *template.Template

	// A mux that http server will use. Is provided so the
	// user can extend the server with additional routes.
	Mux *http.ServeMux
}

// Type used to wrap a running simulation interface
// and start and stop the actor's IO muxer.
type simulation struct {
	actorIndex
	rpg2d.RunningSimulation
}

func (s simulation) ConnectActor(a rpg2d.Actor) {
	switch a := a.(type) {
	case *actor:
		a.startIO()
		s.actorIndex[a.Id()] = a

	default:
		panic(fmt.Sprint("unexpected sim.Actor:", a))
	}

	s.RunningSimulation.ConnectActor(a)
}

func (s simulation) RemoveActor(a rpg2d.Actor) {
	s.RunningSimulation.RemoveActor(a)

	switch a := a.(type) {
	case *actor:
		a.stopIO()
		delete(s.actorIndex, a.Id())

	default:
		panic(fmt.Sprint("unexpected sim.Actor:", a))
	}
}

func NewSimShard(c ShardConfig) (*http.Server, error) {
	// TODO pull this information from a datastore
	bounds := coord.Bounds{
		coord.Cell{-1024, 1024},
		coord.Cell{1023, -1023},
	}

	quadTree, err := quad.New(bounds, 40, nil)
	if err != nil {
		return nil, err
	}

	terrainMap, err := rpg2d.NewTerrainMap(bounds, string(rpg2d.TT_GRASS))
	if err != nil {
		return nil, err
	}

	now := stime.Time(0)

	actorIndex := make(actorIndex)

	simDef := rpg2d.SimulationDef{
		FPS: 40,

		// Initial World State
		Now:        now,
		QuadTree:   quadTree,
		TerrainMap: terrainMap,

		InputPhaseHandler:  inputPhase{actorIndex},
		NarrowPhaseHandler: narrowPhase{actorIndex},
	}

	runningSim, err := simDef.Begin()
	if err != nil {
		return nil, err
	}

	datastore := datastore.NewMemDatastore()

	wsRoute := "/actor/socket"
	var wsUrl string
	if c.IsHTTPS {
		wsUrl = "wss://"
	} else {
		wsUrl = "ws://"
	}

	wsUrl += c.LAddr + wsRoute

	indexHandler := indexHandler{
		c.IndexTmpl,
		ClientSettings{
			c.JsMain,
			wsUrl,
			SimulationSettings{
				Width:  quadTree.Bounds().Width(),
				Height: quadTree.Bounds().Height(),
			},
		},
	}

	mux := c.Mux

	mux.Handle("/", indexHandler)
	mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir(c.JsDir))))
	mux.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir(c.AssetDir))))
	mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(c.CssDir))))
	mux.Handle(wsRoute, newWebsocketActorHandler(simulation{
		actorIndex,
		runningSim,
	}, datastore))

	return &http.Server{
		Addr:    c.LAddr,
		Handler: mux,
	}, nil
}
