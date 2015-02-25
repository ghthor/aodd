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

	case assailEntity:
		// Destroy all assail entities
		return nil

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

// In frames
const assailCooldown = 40

type assailEntity struct {
	id entity.Id

	spawnedBy entity.Id
	spawnedAt stime.Time

	cell coord.Cell

	damage int
}

type assailEntityState struct {
	Type string `json:"type"`

	EntityId entity.Id `json:"id"`

	SpawnedBy entity.Id  `json:"spawnedBy"`
	SpawnedAt stime.Time `json:"spawnedAt"`

	Cell coord.Cell `json:"cell"`
}

func (e assailEntity) Id() entity.Id    { return e.id }
func (e assailEntity) Cell() coord.Cell { return e.cell }
func (e assailEntity) Bounds() coord.Bounds {
	return coord.Bounds{e.cell, e.cell}
}

func (e assailEntity) ToState() entity.State {
	return assailEntityState{
		Type: "assail",

		EntityId: e.id,

		SpawnedBy: e.spawnedBy,
		SpawnedAt: e.spawnedAt,

		Cell: e.cell,
	}
}

func (e assailEntityState) Id() entity.Id { return e.EntityId }
func (e assailEntityState) Bounds() coord.Bounds {
	return coord.Bounds{e.Cell, e.Cell}
}

func (e assailEntityState) IsDifferentFrom(entity.State) bool {
	return true
}

func (phase inputPhase) processUseCmd(a *actor, now stime.Time) []entity.Entity {
	cmd := a.ReadUseCmd()
	if cmd == nil {
		return nil
	}

	// TODO Only allow when stationary
	// TODO Trigger a cooldown
	switch cmd.skill {
	case "assail":
		// Implement a cooldown
		if a.lastAssail.spawnedAt+assailCooldown > now {
			return nil
		}

		e := assailEntity{
			id: phase.nextId(),

			spawnedBy: a.actorEntity.Id(),
			spawnedAt: now,

			cell: a.Cell().Neighbor(a.facing),

			damage: 25,
		}

		a.lastAssail = e

		return []entity.Entity{e}
	}
	return nil
}
