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
	index actorIndex
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
