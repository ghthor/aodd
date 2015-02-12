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
		case actorEntity:
			actor := phase.index[a.Id()]
			_ = actor.ReadCmdRequest()
			// TODO Apply the movement request
		default:
			panic(fmt.Sprint("unexpected entity type:", a))
		}
	}
	return c
}

func (narrowPhase) ResolveCollisions(c quad.CollisionGroup, now stime.Time) quad.CollisionGroup {
	return c
}

type serveIndex struct {
	tmpl     *template.Template
	settings ClientSettings
}

func (html serveIndex) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := html.tmpl.Execute(w, html.settings)

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
	quadTree, err := quad.New(coord.Bounds{
		coord.Cell{-1024, 1024},
		coord.Cell{1023, -1023},
	}, 40, nil)

	if err != nil {
		return nil, err
	}

	now := stime.Time(0)

	actorIndex := make(actorIndex)

	simDef := rpg2d.SimulationDef{
		FPS: 40,

		// Initial World State
		QuadTree: quadTree,
		Now:      now,

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

	indexHandler := serveIndex{
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
