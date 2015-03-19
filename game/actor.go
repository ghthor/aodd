package game

import (
	"fmt"

	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/engine/rpg2d"
	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/rpg2d/entity"
)

// Object stored in the quad tree
type actorEntity struct {
	id      entity.Id
	actorId rpg2d.ActorId

	name string

	// Movement and position
	cell   coord.Cell
	facing coord.Direction
	speed  int

	pathAction     *coord.PathAction
	lastMoveAction coord.MoveAction

	// Health and Mana
	hp, hpMax,
	mp, mpMax int
}

type ActorEntityState struct {
	Id entity.Id `json:"id"`

	Name string `json:"name"`

	// Movement and position
	Facing coord.Direction `json:"facing"`
	Cell   coord.Cell      `json:"cell"`
	bounds coord.Bounds

	PathAction *coord.PathActionState `json:"pathAction"`

	// Health and Mana
	Hp    int `json:"hp"`
	HpMax int `json:"hpMax"`
	Mp    int `json:"mp"`
	MpMax int `json:"mpMax"`
}

type actor struct {
	id rpg2d.ActorId

	actorEntity
	undoLastMoveAction func()

	// Store the last assail me made
	lastAssail assailEntity

	actorConn
}

func NewActor(id entity.Id, dsactor datastore.Actor, stateWriter InitialStateWriter) *actor {
	return &actor{
		id: dsactor.Id,

		actorEntity: actorEntity{
			id:      id,
			actorId: dsactor.Id,

			name: dsactor.Name,

			cell:   origin,
			facing: dsactor.Facing,
			speed:  15,

			pathAction: nil,
			lastMoveAction: coord.TurnAction{
				From: dsactor.Facing,
				To:   dsactor.Facing,
			},

			hp:    100,
			hpMax: 100,
		},

		actorConn: newActorConn(stateWriter),
	}
}

func (a actor) Id() rpg2d.ActorId      { return a.id }
func (a *actor) Entity() entity.Entity { return a.actorEntity }

func (e actorEntity) ActorId() rpg2d.ActorId { return e.actorId }

func (e actorEntity) Id() entity.Id    { return e.id }
func (e actorEntity) Cell() coord.Cell { return e.cell }
func (e actorEntity) Bounds() coord.Bounds {
	bounds := coord.Bounds{
		e.cell,
		e.cell,
	}

	if e.pathAction != nil {
		bounds = coord.JoinBounds(bounds, e.pathAction.Bounds())
	}

	return bounds
}

func (e actorEntity) ToState() entity.State {
	var pathAction *coord.PathActionState

	if e.pathAction != nil {
		pa := e.pathAction.ToState()
		pathAction = &pa
	}

	return ActorEntityState{
		Id: e.id,

		Name: e.name,

		Cell:   e.cell,
		Facing: e.facing,

		bounds: e.Bounds(),

		PathAction: pathAction,

		Hp:    e.hp,
		HpMax: e.hpMax,
		Mp:    e.mp,
		MpMax: e.mpMax,
	}
}

func (e actorEntity) String() string {
	return fmt.Sprintf("{name: %s, id %d, cell%v, %v, speed:%d, pathAction:%v}", e.name, e.id, e.cell, e.facing, e.speed, e.pathAction)
}

func (e ActorEntityState) EntityId() entity.Id  { return e.Id }
func (e ActorEntityState) Bounds() coord.Bounds { return e.bounds }
func (e ActorEntityState) IsDifferentFrom(other entity.State) (different bool) {
	o := other.(ActorEntityState)

	switch {
	case e.Name != o.Name:
		return true

	case e.Facing != o.Facing:
		return true
	case e.PathAction != nil && o.PathAction != nil:
		if *e.PathAction != *o.PathAction {
			return true
		}
	case e.Cell != o.Cell:
		return true
	case e.bounds != o.bounds:
		return true

	case e.Hp != o.Hp || e.HpMax != o.HpMax:
		return true
	case e.Mp != o.Mp || e.MpMax != o.MpMax:
		return true
	}

	return false
}
