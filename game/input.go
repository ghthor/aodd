package game

import (
	"fmt"

	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/rpg2d/entity"
	"github.com/ghthor/engine/sim/stime"
)

type updatePhase struct {
	index actorIndex
}

type inputPhase struct {
	index  actorIndex
	nextId func() entity.Id
}

func (phase updatePhase) Update(e entity.Entity, now stime.Time) entity.Entity {
	switch e := e.(type) {
	case actorEntity:
		actor := phase.index[e.ActorId()]

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
		var entities []entity.Entity
		actor := phase.index[e.ActorId()]

		phase.processMoveCmd(actor, now)
		entities = append(entities,
			phase.processUseCmd(actor, now)...,
		)

		return append(entities, actor.Entity())

	default:
		panic(fmt.Sprint("unexpected entity type:", e))
	}
}

func (phase inputPhase) processMoveCmd(a *actor, now stime.Time) {
	cmd := a.ReadMoveCmd()
	if cmd == nil {
		// The client has canceled all move requests
		return
	}

	// Actor is already moving so the moveRequest won't be
	// consumed until the path action has been completed
	if a.pathAction != nil {
		return
	}

	// Actor may be able to move
	pathAction := &coord.PathAction{
		Span: stime.NewSpan(now, now+stime.Time(a.speed)),
		Orig: a.Cell(),
		Dest: a.Cell().Neighbor(cmd.Direction),
	}

	if pathAction.CanHappenAfter(a.lastMoveAction) {
		a.applyPathAction(pathAction)
		return
	}

	// Actor must change facing
	if a.facing != cmd.Direction {
		turnAction := coord.TurnAction{
			From: a.facing,
			To:   cmd.Direction,
			Time: now,
		}

		if turnAction.CanHappenAfter(a.lastMoveAction) {
			a.applyTurnAction(turnAction)
		}
	}
}

func (phase inputPhase) processUseCmd(a *actor, now stime.Time) []entity.Entity {
	cmd := a.ReadUseCmd()
	if cmd == nil {
		return nil
	}

	switch cmd.skill {
	default:
	}
	return nil
}
