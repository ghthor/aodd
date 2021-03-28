package game

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ghthor/filu/rpg2d/coord"
	"github.com/ghthor/filu/rpg2d/entity"
	"github.com/ghthor/filu/sim/stime"
)

type updatePhaseLocker struct {
	*ActorIndexLocker
}

type updatePhase struct {
	index ActorIndex
}

type inputPhaseLocker struct {
	*ActorIndexLocker
	nextId func() entity.Id
}

type inputPhase struct {
	index  ActorIndex
	nextId func() entity.Id
}

func (phase updatePhaseLocker) Update(e entity.Entity, now stime.Time) entity.Entity {
	defer phase.ActorIndexLocker.RUnlock()
	return updatePhase{phase.ActorIndexLocker.RLock()}.Update(e, now)
}

func (phase updatePhase) Update(e entity.Entity, now stime.Time) entity.Entity {
	switch e := e.(type) {
	case actorEntity:
		actor := phase.index[e.ActorId()]
		if actor.createdAt == 0 {
			actor.flags = actor.flags | entity.FlagNew
			actor.createdAt = now
		} else {
			// Clear FlagNew & set the last know Actor State
			actor.flags = actor.flags &^ entity.FlagNew
			actor.lastState = actor.ToState()
		}

		// Remove any movement actions that have completed
		if actor.pathAction != nil && actor.pathAction.End() <= now {
			actor.lastMoveAction = actor.pathAction
			actor.cell = actor.pathAction.Dest
			actor.pathAction = nil
		}

		// Reset speed after a charge
		if actor.lastStartedCharge+chargeDuration <= now {
			actor.speed = baseSpeed
		}

		return actor.Entity()

	case assailEntity:
		// TODO Fix assail entities never being sent to the server
		//      I already checked the flags and they seem fine
		// Remove all assail entities
		return entity.Removed{e, now}

	case sayEntity:
		// TODO parametize server fps
		if e.saidAt+(sayEntityDuration*40) <= now {
			// Remove all say entities
			return entity.Removed{e, now}
		}

		e.flags = e.flags &^ entity.FlagNew
		return e

	case wallEntity:
		e.flags = e.flags &^ entity.FlagNew
		return e

	case entity.Removed:
		// 3 Secs * Sim FPS
		if e.RemovedAt+(3*40) <= now {
			return nil
		}
		return e

	default:
		panic(fmt.Sprint("unexpected entity type:", e))
	}
}

func (phase inputPhaseLocker) ApplyInputsTo(e entity.Entity, now stime.Time) []entity.Entity {
	defer phase.ActorIndexLocker.RUnlock()
	return inputPhase{phase.RLock(), phase.nextId}.ApplyInputsTo(e, now)
}

func (phase inputPhase) ApplyInputsTo(e entity.Entity, now stime.Time) []entity.Entity {
	switch e := e.(type) {
	case actorEntity:
		var entities []entity.Entity
		actor := phase.index[e.ActorId()]

		entities = append(entities,
			phase.processUseCmd(actor, now)...,
		)

		phase.processMoveCmd(actor, now)

		entities = append(entities,
			phase.processChatCmd(actor, now)...,
		)

		return append(entities, actor.Entity())

	case sayEntity:
		return []entity.Entity{e}

	case wallEntity:
		return []entity.Entity{e}

	case entity.Removed:
		return []entity.Entity{e}

	default:
		panic(fmt.Sprint("unexpected entity type:", e))
	}
}

type MoveRequestType int

//go:generate stringer -type=MoveRequestType
const (
	MR_ERROR MoveRequestType = iota
	MR_MOVE
	MR_MOVE_CANCEL
	MR_SIZE
)

type MoveRequest struct {
	MoveRequestType
	stime.Time
	coord.Direction
}

type moveCmd struct {
	stime.Time
	coord.Direction
}

type UseRequestType int

//go:generate stringer -type=UseRequestType
const (
	UR_ERROR UseRequestType = iota
	UR_USE
	UR_USE_CANCEL
	UR_SIZE
)

type UseRequest struct {
	UseRequestType
	stime.Time
	Skill string
}

type useCmd struct {
	stime.Time
	skill string
}

type ChatRequestType int

//go:generate stringer -type=ChatRequestType
const (
	CR_ERROR ChatRequestType = iota
	CR_SAY
	CR_SIZE
)

type ChatRequest struct {
	ChatRequestType
	stime.Time
	Msg string
}

type chatCmd struct {
	ChatRequestType
	stime.Time
	msg string
}

func newMoveRequest(t MoveRequestType, timeIssued stime.Time, params string) (MoveRequest, error) {
	d, err := coord.NewDirectionWithString(params)
	if err != nil {
		return MoveRequest{}, err
	}

	return MoveRequest{
		t,
		timeIssued,
		d,
	}, nil
}

func newUseRequest(t UseRequestType, timeIssued stime.Time, params string) (UseRequest, error) {
	switch params {
	case "assail":
		return UseRequest{t, timeIssued, params}, nil
	case "charge":
		return UseRequest{t, timeIssued, params}, nil
	default:
		return UseRequest{}, fmt.Errorf("unknown skill: %s", params)
	}
}

func newChatRequest(t ChatRequestType, timeIssued stime.Time, params string) (ChatRequest, error) {
	if len(params) > 120 {
		return ChatRequest{}, errors.New("chat message exceeded 120 char limit")
	}

	return ChatRequest{
		ChatRequestType: t,
		Time:            timeIssued,
		Msg:             params,
	}, nil
}

// If the cmd and params are a valid combination
// a request will be made and passed into the
// IO muxer. SubmitCmd() will return an error
// if the submitted cmd and params are invalid.
func (c actorConn) SubmitCmd(cmd, params string) error {
	parts := strings.Split(cmd, "=")

	if len(parts) != 2 {
		return errors.New("invalid command syntax")
	}

	timeIssued, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return err
	}

	switch parts[0] {
	case "move":
		r, err := newMoveRequest(MR_MOVE, stime.Time(timeIssued), params)
		if err != nil {
			return err
		}

		c.submitMoveRequest <- r

	case "moveCancel":
		r, err := newMoveRequest(MR_MOVE_CANCEL, stime.Time(timeIssued), params)
		if err != nil {
			return err
		}

		c.submitMoveRequest <- r

	case "use":
		r, err := newUseRequest(UR_USE, stime.Time(timeIssued), params)
		if err != nil {
			return err
		}

		c.submitUseRequest <- r

	case "useCancel":
		r, err := newUseRequest(UR_USE_CANCEL, stime.Time(timeIssued), params)
		if err != nil {
			return err
		}

		c.submitUseRequest <- r

	case "say":
		r, err := newChatRequest(CR_SAY, stime.Time(timeIssued), params)
		if err != nil {
			return err
		}

		c.submitChatRequest <- r

	default:
		return fmt.Errorf("unknown command: %s", parts[0])
	}

	return nil
}

func (c actorConn) SubmitMoveRequest(r MoveRequest) {
	select {
	case c.submitMoveRequest <- r:
	default:
	}
}

func (c actorConn) SubmitUseRequest(r UseRequest) {
	select {
	case c.submitUseRequest <- r:
	default:
	}
}

func (c actorConn) SubmitChatRequest(r ChatRequest) {
	select {
	case c.submitChatRequest <- r:
	default:
	}
}

func (c actorConn) ReadMoveCmd() *moveCmd {
	return <-c.readMoveCmd
}

func (a *actor) applyPathAction(pa *coord.PathAction) {
	prevPathAction := a.pathAction
	prevFacing := a.facing
	prevFlags := a.flags

	a.undoLastMoveAction = func() {
		a.pathAction = prevPathAction
		a.facing = prevFacing
		a.flags = prevFlags
		a.undoLastMoveAction = nil
	}

	a.pathAction = pa
	a.facing = pa.Direction()
}

func (a *actor) applyTurnAction(ta coord.TurnAction) {
	prevAction := a.lastMoveAction
	prevFacing := a.facing
	prevFlags := a.flags

	a.undoLastMoveAction = func() {
		a.lastMoveAction = prevAction
		a.facing = prevFacing
		a.flags = prevFlags
		a.undoLastMoveAction = nil
	}

	a.lastMoveAction = ta
	a.facing = ta.To
}

func (a *actor) revertMoveAction() {
	if a.undoLastMoveAction != nil {
		a.undoLastMoveAction()
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

// 1s
// In frames
const assailCooldown = 40

type assailEntity struct {
	id entity.Id

	spawnedBy entity.Id
	spawnedAt stime.Time

	cell  coord.Cell
	flags entity.Flag

	damage int
}

type AssailEntityState struct {
	Type string `json:"type"`

	Id entity.Id `json:"id"`

	SpawnedBy entity.Id  `json:"spawnedBy"`
	SpawnedAt stime.Time `json:"spawnedAt"`

	Cell coord.Cell `json:"cell"`
}

func (e assailEntity) Id() entity.Id    { return e.id }
func (e assailEntity) Cell() coord.Cell { return e.cell }
func (e assailEntity) Bounds() coord.Bounds {
	return coord.Bounds{e.cell, e.cell}
}
func (e assailEntity) Flags() entity.Flag { return 0 }

func (e assailEntity) ToState() entity.State {
	return AssailEntityState{
		Type: "assail",

		Id: e.id,

		SpawnedBy: e.spawnedBy,
		SpawnedAt: e.spawnedAt,

		Cell: e.cell,
	}
}

func (e AssailEntityState) EntityId() entity.Id    { return e.Id }
func (e AssailEntityState) EntityCell() coord.Cell { return e.Cell }
func (e AssailEntityState) Bounds() coord.Bounds {
	return coord.Bounds{e.Cell, e.Cell}
}

func (e AssailEntityState) IsDifferentFrom(entity.State) bool {
	return true
}

func (c actorConn) ReadUseCmd() *useCmd {
	return <-c.readUseCmd
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

			cell:  a.Cell().Neighbor(a.facing),
			flags: entity.FlagNew,

			damage: 25,
		}

		a.lastAssail = e

		return []entity.Entity{e}

	case "charge":
		// Implement a cooldown
		if a.lastStartedCharge+chargeCooldown <= now {
			a.speed = chargeSpeed
			a.lastStartedCharge = now
		}
	}
	return nil
}

// In seconds
const sayEntityDuration = 3

type sayEntity struct {
	id entity.Id

	saidBy entity.Id
	saidAt stime.Time

	cell  coord.Cell
	flags entity.Flag

	msg string
}

type SayEntityState struct {
	Type string `json:"type"`

	Id entity.Id `json:"id"`

	SaidBy entity.Id  `json:"saidBy"`
	SaidAt stime.Time `json:"saidAt"`

	Cell coord.Cell `json:"cell"`

	Msg string `json:"msg"`
}

func (e sayEntity) Id() entity.Id    { return e.id }
func (e sayEntity) Cell() coord.Cell { return e.cell }
func (e sayEntity) Bounds() coord.Bounds {
	return coord.Bounds{e.cell, e.cell}
}

func (e sayEntity) Flags() entity.Flag { return e.flags }

func (e sayEntity) ToState() entity.State {
	return SayEntityState{
		Type: "say",

		Id: e.id,

		SaidBy: e.saidBy,
		SaidAt: e.saidAt,

		Cell: e.cell,

		Msg: e.msg,
	}
}

func (e SayEntityState) EntityId() entity.Id    { return e.Id }
func (e SayEntityState) EntityCell() coord.Cell { return e.Cell }
func (e SayEntityState) IsDifferentFrom(other entity.State) bool {
	switch other := other.(type) {
	case SayEntityState:
		return e.Id != other.Id ||
			e.Msg != other.Msg ||
			e.SaidBy != other.SaidBy ||
			e.SaidAt != other.SaidAt ||
			e.Cell != other.Cell
	}

	return true
}

func (a *actor) ReadChatCmd() *chatCmd {
	return <-a.readChatCmd
}

func (phase inputPhase) processChatCmd(a *actor, now stime.Time) []entity.Entity {
	cmd := a.ReadChatCmd()
	if cmd == nil {
		return nil
	}

	switch cmd.ChatRequestType {
	case CR_SAY:
		return []entity.Entity{sayEntity{
			id: phase.nextId(),

			saidBy: a.actorEntity.Id(),
			saidAt: now,

			cell:  a.Cell(),
			flags: entity.FlagNew | entity.FlagNoCollide,

			msg: cmd.msg,
		}}
	}

	return nil
}
