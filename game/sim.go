package game

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/engine/rpg2d"
	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/rpg2d/entity"
	"github.com/ghthor/engine/rpg2d/quad"
	"github.com/ghthor/engine/sim"
	"github.com/ghthor/engine/sim/stime"
)

type entityResolver struct{}
type inputPhase struct{}
type narrowPhase struct{}

func (entityResolver) EntityForActor(a sim.Actor) entity.Entity {
	switch a := a.(type) {
	case actor:
		return a.actorEntity
	default:
		panic(fmt.Sprint("unexpected actor type:", a))
	}
	return nil
}

func (inputPhase) ApplyInputsIn(c quad.Chunk, now stime.Time) quad.Chunk {
	for _, e := range c.Entities {
		switch a := e.(type) {
		case *actorEntity:
			// TODO Resolve to an actor
			// TODO Read in a movement request
			// TODO Apply the movement request
			fmt.Print(a.Id())
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
	sim.RunningSimulation
}

func (s simulation) ConnectActor(a sim.Actor) error {
	err := s.RunningSimulation.ConnectActor(a)
	if err != nil {
		return err
	}

	switch a := a.(type) {
	case actor:
		a.startIO()

	default:
		panic(fmt.Sprint("unexpected sim.Actor:", a))
	}

	return nil

}

func (s simulation) RemoveActor(a sim.Actor) error {
	err := s.RunningSimulation.RemoveActor(a)

	switch a := a.(type) {
	case actor:
		a.stopIO()

	default:
		panic(fmt.Sprint("unexpected sim.Actor:", a))
	}
	return err
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

	simDef := rpg2d.SimulationDef{
		FPS: 40,

		// Initial World State
		QuadTree: quadTree,
		Now:      now,

		EntityResolver:     entityResolver{},
		InputPhaseHandler:  inputPhase{},
		NarrowPhaseHandler: narrowPhase{},
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
	mux.Handle(wsRoute, newWebsocketActorHandler(simulation{runningSim}, datastore))

	return &http.Server{
		Addr:    c.LAddr,
		Handler: mux,
	}, nil
}
