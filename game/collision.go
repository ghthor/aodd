package game

import (
	"errors"
	"fmt"

	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/rpg2d/entity"
	"github.com/ghthor/engine/rpg2d/quad"
	"github.com/ghthor/engine/sim/stime"
)

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

// Returns if the collision exists in the
// slice of collisions that have been
// solved during this narrow phase tick.
func (phase narrowPhase) hasSolved(c quad.Collision) bool {
	for _, solved := range phase.solved {
		if c.IsSameAs(solved) {
			return true
		}
	}

	return false
}

// Implementation of the quad.NarrowPhaseHandler interface.
func (phase narrowPhase) ResolveCollisions(cg *quad.CollisionGroup, now stime.Time) ([]entity.Entity, []entity.Entity) {
	// Reset the resolved slice
	phase.solved = phase.solved[:0]

	// Generate a collision index for the collision group
	phase.collisionIndex = cg.CollisionIndex()

	// A map to store entities that still remain in the world
	remaining := make(map[entity.Id]entity.Entity, len(cg.Entities))
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
			entities = phase.resolveActorEntity(phase.actorIndex[e.ActorId()], c.B, c)
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
		b := phase.actorIndex[e.ActorId()]

		return phase.solveActorActor(&solverActorActor{}, a, b, collision)
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
		// It is more relevant to return nil here then a
		// coord.Collision with type CT_NONE
		if collision.Type() == coord.CT_NONE {
			return a, b, nil
		}

	case a.pathAction != nil && b.pathAction != nil:
		pathCollision := coord.NewPathCollision(*a.pathAction, *b.pathAction)

		// coord.NewPathCollision can flip the,
		// A and B paths to simplify the number
		// of collision types. This normalizes
		// actor A with pathCollision.A
		if *a.pathAction != pathCollision.A {
			a, b = b, a
		}

		collision = pathCollision
	case a.pathAction == nil && b.pathAction == nil:
		// This case handles actors being on the same square,
		// but not moving at all.
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

// Move forward in the directed graph. This movement is based on
// which entity is occupying the destination of the other's path action.
func followGraph(a, b *actor, collision quad.Collision) node {
	// normalize a, b to collision.[A, B]
	if a.actorEntity.Id() != collision.A.Id() {
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

// Used to figure out which actor is "A" if
// the collision was CT_A_INTO_B instead of CT_NONE
func currentNode(a, b *actor, collision quad.Collision) *actor {
	switch {
	case a.pathAction.Orig == b.pathAction.Dest:
		return b

	case b.pathAction.Orig == a.pathAction.Dest:
		return a

	default:
		panic(fmt.Sprintf("unexpected graph state %v between %v & %v", collision, a, b))
	}
}

// Compare entity Id's with the entities in
// a collision and return the one that isn't
// the actor.
func otherEntityIn(a *actor, collision quad.Collision) entity.Entity {
	var e entity.Entity

	// figure out is prioritized actor is A or B in the collision
	switch {
	case a.actorEntity.Id() != collision.A.Id():
		e = collision.A

	case a.actorEntity.Id() != collision.B.Id():
		e = collision.B

	default:
		panic(fmt.Sprintf("unexpected graph state %v actor %v", collision, a))
	}

	return e
}

// Store what actor's have been visited during
// a recursive solve. Used to avoid infinite
// recursion through a cycle in the graph.
type solverActorActor struct {
	visited []*actor
}

func (s solverActorActor) hasVisited(actor *actor) bool {
	for _, a := range s.visited {
		if actor == a {
			return true
		}
	}

	return false
}

func (phase *narrowPhase) solveActorActor(solver *solverActorActor, a, b *actor, collision quad.Collision) []entity.Entity {

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
		e, err := phase.solveDependencies(solver, a, b, collision)

		switch err {
		case nil:
			if len(e) > 0 {
				entities = append(entities, e...)
			}

			// Try solving again
			goto attemptSolve

		case errCycleDetected:
			// Detected a cycle, we can't move
			currentNode(a, b, collision).revertMoveAction()
			goto resolved

		case errNoDependencies:
			// All dependencies have been solved
			// We can move
			goto resolved
		}

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
		e, err := phase.solveDependencies(solver, a, b, collision)

		switch err {
		case nil:
			if len(e) > 0 {
				entities = append(entities, e...)
			}

			// Try solving again
			goto attemptSolve

		case errCycleDetected:
			a.revertMoveAction()
			goto resolved

		case errNoDependencies:
			if a.pathAction.End() >= b.pathAction.End() {
				goto resolved
			}

			a.revertMoveAction()
			goto resolved
		}

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

var errNoDependencies = errors.New("no dependencies")
var errCycleDetected = errors.New("cycle detected")

// Error can be errNoDependencies, errCycleDetected or nil
func (phase *narrowPhase) solveDependencies(solver *solverActorActor, a, b *actor, collision quad.Collision) ([]entity.Entity, error) {
	node := followGraph(a, b, collision)

	// Mark what actors have been visited
	if a != node.actor {
		solver.visited = append(solver.visited, a)
	} else {
		solver.visited = append(solver.visited, b)
	}

	// If the next node only has one collision
	// then there are no dependencies and the
	// collision can be solved
	if len(phase.collisionIndex[node.entity]) == 1 {
		return nil, errNoDependencies
	}

	// Walk through the directed graph of collisions and solve
	// all the collisions that the collision depends on.
	for _, c := range phase.collisionIndex[node.entity] {
		// Ignore the collision that caused us to recurse
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
			actor := phase.actorIndex[e.ActorId()]

			// Detect cycles
			if solver.hasVisited(actor) {
				return nil, errCycleDetected
			}

			// Recurse
			return phase.solveActorActor(solver, node.actor, actor, c), nil
		}
	}

	return nil, errNoDependencies
}
