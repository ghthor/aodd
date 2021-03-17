package game

import (
	"fmt"
	"net/http"
	"sync"
	"text/template"

	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/filu/rpg2d"
	"github.com/ghthor/filu/rpg2d/coord"
	"github.com/ghthor/filu/rpg2d/entity"
	"github.com/ghthor/filu/rpg2d/quad"
	"github.com/ghthor/filu/sim/stime"
)

// Store actor's indexed by id
type ActorIndex map[rpg2d.ActorId]*actor

const quadMaxSize = 20

// A wrapper for actorIndex that provides
// safety for concurrent access.
type ActorIndexLocker struct {
	sync.RWMutex
	index ActorIndex
}

func (m *ActorIndexLocker) RLock() ActorIndex {
	m.RWMutex.RLock()
	return m.index
}

func (m *ActorIndexLocker) Lock() ActorIndex {
	m.RWMutex.Lock()
	return m.index
}

func (m *ActorIndexLocker) Unlock(index ActorIndex) {
	m.index = index
	m.RWMutex.Unlock()
}

func NewActorIndexLocker(around ActorIndex) *ActorIndexLocker {
	return &ActorIndexLocker{index: around}
}

// Type used to wrap a running simulation interface
// and start and stop the actor's IO muxer.
// Also adds and removes actors from the
// simulation's actor index.
type simulation struct {
	*ActorIndexLocker
	rpg2d.RunningSimulation
}

func (s simulation) ConnectActor(a rpg2d.Actor) {
	switch a := a.(type) {
	case *actor:
		a.startIO()
		actorIndex := s.ActorIndexLocker.Lock()
		actorIndex[a.Id()] = a
		s.ActorIndexLocker.Unlock(actorIndex)

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
		actorIndex := s.ActorIndexLocker.Lock()
		delete(actorIndex, a.Id())
		s.ActorIndexLocker.Unlock(actorIndex)

	default:
		panic(fmt.Sprint("unexpected sim.Actor:", a))
	}
}

func NewSimulation(actorIndex *ActorIndexLocker, sim rpg2d.RunningSimulation) rpg2d.RunningSimulation {
	return simulation{
		ActorIndexLocker:  actorIndex,
		RunningSimulation: sim,
	}
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
	// Flag whether the server is running from heroku
	OnHeroku bool

	// FQDN that the server is running at
	Domain,

	// Port that the server is running on
	Port,

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

	// A handler that will be used instead of the Mux
	// when setting the Handler field of the *http.Server.
	// If Handler is nil, the Mux will be used instead.
	Handler http.Handler
}

type inputReceiver struct {
	*actor
	disconnect func()
}

func (i inputReceiver) Close() {
	i.disconnect()
}

func NewSimShard(c ShardConfig) (*http.Server, error) {
	// TODO pull this information from a datastore
	bounds := coord.Bounds{
		coord.Cell{1, -1},
		coord.Cell{128, -128},
	}

	quadTree, err := quad.New(bounds, quadMaxSize, nil)
	if err != nil {
		return nil, err
	}

	terrainMap, err := rpg2d.NewTerrainMap(bounds, startingTerrain)
	if err != nil {
		return nil, err
	}

	now := stime.Time(0)

	actorIndex := NewActorIndexLocker(make(ActorIndex))

	entityIdGen := entity.NewIdGenerator()

	quadTree = addWalls(quadTree, entityIdGen)

	simDef := rpg2d.SimulationDef{
		FPS: 40,

		// Initial World State
		Now:        now,
		QuadTree:   quadTree,
		TerrainMap: terrainMap,

		UpdatePhaseHandler: updatePhaseLocker{actorIndex},
		InputPhaseHandler:  inputPhaseLocker{actorIndex, entityIdGen},
		NarrowPhaseHandler: newNarrowPhaseLocker(actorIndex),
	}

	runningSim, err := simDef.Begin()
	if err != nil {
		return nil, err
	}

	wsUrl := "ws://" + c.Domain
	wsRoute := "/actor/socket/gob"

	if !c.OnHeroku {
		wsUrl += ":" + c.Port
	}

	wsUrl += wsRoute

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

	ds := datastore.NewMemDatastore()
	sim := NewSimulation(actorIndex, runningSim)

	mux.Handle("/", indexHandler)
	mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir(c.JsDir))))
	mux.Handle("/asset/", http.StripPrefix("/asset/", http.FileServer(http.Dir(c.AssetDir))))
	mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(c.CssDir))))
	mux.Handle(wsRoute, newGobWebsocketHandler(
		ds,
		func(dsactor datastore.Actor, stateWriter InitialStateWriter) (InputReceiver, entity.State) {
			actor := NewActor(entityIdGen(), dsactor, stateWriter)
			sim.ConnectActor(actor)

			return inputReceiver{
				actor:      actor,
				disconnect: func() { sim.RemoveActor(actor) },
			}, actor.Entity().ToState()
		},
	))

	defaultHandler := c.Handler
	if defaultHandler == nil {
		defaultHandler = mux
	}

	var laddr string
	if c.OnHeroku {
		laddr = ":" + c.Port
	} else {
		laddr = fmt.Sprintf("%s:%s", c.Domain, c.Port)
	}

	return &http.Server{
		Addr:    laddr,
		Handler: defaultHandler,
	}, nil
}
