package game

import (
	"errors"
	"fmt"
	"net/http"
	"text/template"

	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/engine/rpg2d"
	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/rpg2d/entity"
	"github.com/ghthor/engine/rpg2d/quad"
	"github.com/ghthor/engine/sim/stime"
)

type actorIndex map[int64]*actor

type updatePhase struct {
	index actorIndex
}
type inputPhase struct {
	index actorIndex
}

type narrowPhase struct {
	actorIndex actorIndex

	// Reset at the beginning of every ResolveCollisions call
	solved []quad.Collision
	// Generated at the beginning of every ResolveCollisions call
	collisionIndex quad.CollisionIndex
}

func newNarrowPhase(actorIndex actorIndex) narrowPhase {
	return narrowPhase{actorIndex, make([]quad.Collision, 0, 10), nil}
}

func (phase updatePhase) Update(e entity.Entity, now stime.Time) entity.Entity {
	switch e := e.(type) {
	case actorEntity:
		actor := phase.index[e.Id()]

		// Remove any movement actions that have completed
		if actor.pathAction != nil && actor.pathAction.End() <= now {
			actor.lastMoveAction = actor.pathAction
			actor.cell = actor.pathAction.Dest
			actor.pathAction = nil
		}

		return actor.Entity()

	default:
		panic(fmt.Sprint("unexpected entity type:", e))
	}
}

func (phase inputPhase) ApplyInputsTo(e entity.Entity, now stime.Time) []entity.Entity {
	switch e := e.(type) {
	case actorEntity:
		actor := phase.index[e.Id()]
		cmdReq := actor.ReadCmdRequest()

		if cmdReq.moveRequest == nil {
			// The client has canceled all move requests
			actor.actorCmdRequest.moveRequest = nil
			return []entity.Entity{actor.Entity()}
		}

		// The client has a standing move request
		moveRequest := cmdReq.moveRequest
		actor.actorCmdRequest.moveRequest = moveRequest

		// Actor is already moving so the moveRequest won't be
		// consumed until the path action has been completed
		if actor.pathAction != nil {
			return []entity.Entity{actor.Entity()}
		}

		// Actor may be able to move
		pathAction := &coord.PathAction{
			Span: stime.NewSpan(now, now+stime.Time(actor.speed)),
			Orig: actor.Cell(),
			Dest: actor.Cell().Neighbor(moveRequest.Direction),
		}

		if pathAction.CanHappenAfter(actor.lastMoveAction) {
			actor.applyPathAction(pathAction)
			return []entity.Entity{actor.Entity()}
		}

		// Actor must change facing
		if actor.facing != moveRequest.Direction {
			turnAction := coord.TurnAction{
				From: actor.facing,
				To:   moveRequest.Direction,
				Time: now,
			}

			if turnAction.CanHappenAfter(actor.lastMoveAction) {
				actor.applyTurnAction(turnAction)
			}
		}

		return []entity.Entity{actor.Entity()}

	default:
		panic(fmt.Sprint("unexpected entity type:", e))
	}
}

func (phase narrowPhase) hasSolved(c quad.Collision) bool {
	for _, solved := range phase.solved {
		if c.IsSameAs(solved) {
			return true
		}
	}

	return false
}

func (phase narrowPhase) ResolveCollisions(cg *quad.CollisionGroup, now stime.Time) ([]entity.Entity, []entity.Entity) {
	// Reset the resolved slice
	phase.solved = phase.solved[:0]

	// Generate a collision index for the collision group
	phase.collisionIndex = cg.CollisionIndex()

	// A map to store entities that still remain in the world
	remaining := make(map[int64]entity.Entity, len(cg.Entities))
	remainingSlice := func() []entity.Entity {
		// Build a slice from the `remaining` map
		s := make([]entity.Entity, 0, len(remaining))
		for _, e := range remaining {
			s = append(s, e)
		}
		return s
	}

	for _, c := range cg.Collisions {
		if phase.hasSolved(c) {
			continue
		}

		var entities []entity.Entity

		// Resolve type of entity in collision.A
		switch e := c.A.(type) {
		case actorEntity:
			// Resolve the type of entity in collision.B
			entities = phase.resolveActorEntity(phase.actorIndex[e.Id()], c.B, c)
		}

		// As collisions are solved they return entities
		// that have been created or modified and we store
		// them in a map by their Id. Multiple collisions
		// may modify and entity, therefor we only will
		// one version of the entity back to engine when
		// we return.
		for _, e := range entities {
			remaining[e.Id()] = e
		}
	}

	return remainingSlice(), nil
}

func (phase *narrowPhase) resolveActorEntity(a *actor, with entity.Entity, collision quad.Collision) []entity.Entity {
	switch e := with.(type) {
	case actorEntity:
		b := phase.actorIndex[e.Id()]

		s := solverActorActor{
			startedFrom: collision,
			a:           a, b: b,
			collision: collision,
		}

		return phase.solveActorActor(s)
	}

	return nil
}

func newActorActorCollision(a, b *actor) (*actor, *actor, coord.Collision) {
	var collision coord.Collision

	switch {
	case a.pathAction == nil && b.pathAction != nil:
		a, b = b, a
		fallthrough
	case a.pathAction != nil && b.pathAction == nil:
		collision = coord.NewCellCollision(*a.pathAction, b.Cell())

		// A or B may have had a previous collision resolved that
		// caused this collision to not be possible anymore.
		// Returning nil here will short circut the switch
		// in the resolveActorActorCollision method and
		// avoid a typecast.
		if collision.Type() == coord.CT_NONE {
			return a, b, nil
		}

	case a.pathAction != nil && b.pathAction != nil:
		pathCollision := coord.NewPathCollision(*a.pathAction, *b.pathAction)

		// coord.NewPathCollision can flip the,
		// A and B paths to simplify the number
		// of cases. This normalizes our A and B
		// with the path collision.
		if *a.pathAction != pathCollision.A {
			a, b = b, a
		}

		collision = pathCollision
	case a.pathAction == nil && b.pathAction == nil:
		// This case handles actors being on the same square.
		// There isn't a coord.CollisionType for this case.
		// Maybe there should be?
		return a, b, nil

	default:
		panic(fmt.Sprintf("impossible collision between {%v} {%v}", a, b))
	}
	return a, b, collision
}

type node struct {
	actor  *actor
	entity entity.Entity
}

func followGraph(a, b *actor, collision quad.Collision) node {
	// normalize a, b to collision.[A, B]
	if a.Id() != collision.A.Id() {
		a, b = b, a
	}

	var actor *actor
	var entity entity.Entity

	switch {
	case a.pathAction.Orig == b.pathAction.Dest:
		entity = collision.A
		actor = a

	case b.pathAction.Orig == a.pathAction.Dest:
		entity = collision.B
		actor = b

	default:
		panic(fmt.Sprintf("unexpected graph state %v between %v & %v", collision, a, b))
	}

	return node{actor, entity}
}

func otherEntityIn(a *actor, collision quad.Collision) entity.Entity {
	var e entity.Entity

	// figure out is prioritized actor is A or B in the collision
	switch {
	case a.Id() != collision.A.Id():
		e = collision.A

	case a.Id() != collision.B.Id():
		e = collision.B

	default:
		panic(fmt.Sprintf("unexpected graph state %v actor %v", collision, a))
	}

	return e
}

type solverActorActor struct {
	startedFrom quad.Collision
	a, b        *actor
	collision   quad.Collision
}

func (phase *narrowPhase) solveActorActor(solver solverActorActor) []entity.Entity {
	a, b := solver.a, solver.b
	collision := solver.collision

	// When this functions returns the
	// collision will have been solved
	defer func() {
		phase.solved = append(phase.solved, collision)
	}()

	var entities []entity.Entity

attemptSolve:
	a, b, coordCollision := newActorActorCollision(a, b)
	if coordCollision == nil {
		goto resolved
	}

	switch coordCollision.Type() {
	case coord.CT_NONE:
		// This may not be entirely accurate.
		// We should walk through the collision index
		// of our partner too see if they should resolve
		// some of there collisions first. They may
		// appear to be moving to us right now, but
		// have a collision that when solved will
		// render them motionless, thus we would become
		// motionless as well.
		e, err := phase.solveDependencies(solver)

		if err != nil && err == ErrNoDependencies {
			// If we ever hit this point it means we've
			// resolved all the collisions this one
			// depends on and therefor it can be resolved.
			goto resolved
		}

		if len(e) > 0 {
			entities = append(entities, e...)
		}

		// Try solving again
		goto attemptSolve

	case coord.CT_CELL_DEST:
		a.revertMoveAction()
		goto resolved

	case coord.CT_SWAP:
		a.revertMoveAction()
		b.revertMoveAction()
		goto resolved

	case coord.CT_A_INTO_B_FROM_SIDE:
		// This may not be entirely accurate.
		// We should walk through the collision index
		// of our partner too see if they should resolve
		// some of there collisions first. They may
		// appear to be moving to us right now, but
		// have a collision that when solved will
		// render them motionless, thus we would become
		// motionless as well.
		e, err := phase.solveDependencies(solver)

		switch err {
		case nil:
			if len(e) > 0 {
				entities = append(entities, e...)
			}

			// Try solving again
			goto attemptSolve

		case ErrCycle:
			a.revertMoveAction()
			goto resolved

		case ErrNoDependencies:
		}

		// If we ever hit this point it means we've
		// resolved all the collisions this one
		// depends on and therefor it can be resolved.
		if a.pathAction.End() >= b.pathAction.End() {
			goto resolved
		}

		fallthrough

	case coord.CT_A_INTO_B:
		a.revertMoveAction()
		goto resolved

	case coord.CT_HEAD_TO_HEAD:
		fallthrough

	case coord.CT_FROM_SIDE:
		if a.pathAction.Start() < b.pathAction.Start() {
			// A has already won the destination
			b.revertMoveAction()
			goto resolved

		} else if a.pathAction.Start() > b.pathAction.Start() {
			// B has already won the destination
			a.revertMoveAction()
			goto resolved
		}
		// Start values are equal

		if a.pathAction.End() < b.pathAction.End() {
			// A is moving faster and wins the destination
			b.revertMoveAction()
			goto resolved

		} else if a.pathAction.End() > b.pathAction.End() {
			// B is moving faster and wins the destination
			a.revertMoveAction()
			goto resolved
		}
		// End values are equal

		// Movement direction priority goes in this order
		// N -> E -> S -> W
		if a.facing < b.facing {
			// A's movement direction has a higher priority
			b.revertMoveAction()
			goto resolved

		} else {
			// B's movement direction has a higher priority
			a.revertMoveAction()
			goto resolved
		}
	}

resolved:
	return append(entities, a.Entity(), b.Entity())
}

var ErrNoDependencies = errors.New("no dependencies")
var ErrCycle = errors.New("cycle detected")

func (phase *narrowPhase) solveDependencies(solver solverActorActor) ([]entity.Entity, error) {
	a, b := solver.a, solver.b
	collision := solver.collision

	node := followGraph(a, b, collision)

	// If the next node only has one collision
	// then there are no dependencies and the
	// collision can be solved
	if len(phase.collisionIndex[node.entity]) == 1 {
		return nil, ErrNoDependencies
	}

	// Walk through the directed graph of collisions and solve
	// all the collisions that the collision depends on.
	for _, c := range phase.collisionIndex[node.entity] {
		// Detect cycles
		if c.IsSameAs(solver.startedFrom) && !c.IsSameAs(collision) {
			return nil, ErrCycle
		}

		// Ignore the collision that caused us to
		// recursively solve dependencies
		if c.IsSameAs(collision) {
			continue
		}

		// Avoid solving a collision that's already been solving.
		if phase.hasSolved(c) {
			continue
		}

		e := otherEntityIn(node.actor, c)

		switch e := e.(type) {
		case actorEntity:
			actor := phase.actorIndex[e.Id()]

			solver.a = node.actor
			solver.b = actor
			solver.collision = c

			return phase.solveActorActor(solver), nil
		}

	}

	return nil, ErrNoDependencies
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
// Also adds and removes actors from the
// simulation's actor index.
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

		UpdatePhaseHandler: updatePhase{actorIndex},
		InputPhaseHandler:  inputPhase{actorIndex},
		NarrowPhaseHandler: newNarrowPhase(actorIndex),
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
	mux.Handle(wsRoute, newWebsocketHandler(simulation{
		actorIndex,
		runningSim,
	}, datastore))

	return &http.Server{
		Addr:    c.LAddr,
		Handler: mux,
	}, nil
}
