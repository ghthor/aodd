package game

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ghthor/engine/net/protocol"
	"github.com/ghthor/engine/rpg2d"
	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/rpg2d/entity"
	"github.com/ghthor/engine/sim/stime"
)

type moveRequestType int

const (
	MR_ERROR moveRequestType = iota
	MR_MOVE
	MR_MOVE_CANCEL
)

type moveRequest struct {
	moveRequestType
	stime.Time
	coord.Direction
}

type moveCmd struct {
	stime.Time
	coord.Direction
}

type useRequestType int

const (
	UR_ERROR useRequestType = iota
	UR_USE
	UR_USE_CANCEL
)

type useRequest struct {
	useRequestType
	stime.Time
	skill string
}

type useCmd struct {
	stime.Time
	skill string
}

func newMoveRequest(t moveRequestType, timeIssued stime.Time, params string) (moveRequest, error) {
	d, err := coord.NewDirectionWithString(params)
	if err != nil {
		return moveRequest{}, err
	}

	return moveRequest{
		t,
		timeIssued,
		d,
	}, nil
}

func newUseRequest(t useRequestType, timeIssued stime.Time, params string) (useRequest, error) {
	switch params {
	case "assail":
		return useRequest{t, timeIssued, params}, nil
	default:
		return useRequest{}, fmt.Errorf("unknown skill: %s", params)
	}
}

type actorConn struct {
	// Comm interface to muxer used by SubmitCmd() method
	submitMoveRequest chan<- moveRequest
	submitUseRequest  chan<- useRequest

	readMoveCmd <-chan *moveCmd
	readUseCmd  <-chan *useCmd

	// Comm interface to muxer used by SendState() method
	sendState chan<- *rpg2d.WorldState

	// Comm interface to muxer used by stopIO() method
	stop chan<- chan<- struct{}

	// External connection used to publish the world state
	protocol.Conn

	lastState rpg2d.WorldState
}

func newActorConn(conn protocol.Conn) actorConn {
	return actorConn{Conn: conn}
}

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

type actorEntityState struct {
	EntityId entity.Id `json:"id"`

	Name string `json:"name"`

	// Movement and position
	Facing string     `json:"facing"`
	Cell   coord.Cell `json:"cell"`
	bounds coord.Bounds

	PathAction *coord.PathActionJson `json:"pathAction"`

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

	actorConn
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
	var pathAction *coord.PathActionJson

	if e.pathAction != nil {
		pa := e.pathAction.Json()
		pathAction = &pa
	}

	return actorEntityState{
		EntityId: e.id,

		Name: e.name,

		Cell:   e.cell,
		Facing: e.facing.String(),

		bounds: e.Bounds(),

		PathAction: pathAction,

		Hp:    e.hp,
		HpMax: e.hpMax,
		Mp:    e.mp,
		MpMax: e.mpMax,
	}
}

func (e actorEntity) String() string {
	return fmt.Sprintf("{id %d, cell%v, %v, speed:%d, pathAction:%v}", e.id, e.cell, e.facing, e.speed, e.pathAction)
}

func (e actorEntityState) Id() entity.Id        { return e.EntityId }
func (e actorEntityState) Bounds() coord.Bounds { return e.bounds }
func (e actorEntityState) IsDifferentFrom(other entity.State) (different bool) {
	o := other.(actorEntityState)

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

func (a *actor) applyPathAction(pa *coord.PathAction) {
	prevPathAction := a.pathAction
	prevFacing := a.facing

	a.undoLastMoveAction = func() {
		a.pathAction = prevPathAction
		a.facing = prevFacing
		a.undoLastMoveAction = nil
	}

	a.pathAction = pa
	a.facing = pa.Direction()
}

func (a *actor) applyTurnAction(ta coord.TurnAction) {
	prevAction := a.lastMoveAction
	prevFacing := a.facing

	a.undoLastMoveAction = func() {
		a.lastMoveAction = prevAction
		a.facing = prevFacing
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

func (a *actorConn) startIO() {
	// Setup communication channels
	moveReqCh := make(chan moveRequest)
	useReqCh := make(chan useRequest)

	moveCmdCh := make(chan *moveCmd)
	useCmdCh := make(chan *useCmd)

	outputCh := make(chan *rpg2d.WorldState)
	stopCh := make(chan chan<- struct{})

	// Set the channels accessible to the outside world
	a.submitMoveRequest = moveReqCh
	a.submitUseRequest = useReqCh

	a.readMoveCmd = moveCmdCh
	a.readUseCmd = useCmdCh

	a.sendState = outputCh
	a.stop = stopCh

	// Establish the channel endpoints used inside the go routine
	var newMoveRequest <-chan moveRequest
	var newUseRequest <-chan useRequest

	var sendMoveCmd chan<- *moveCmd
	var sendUseCmd chan<- *useCmd

	var newState <-chan *rpg2d.WorldState
	var stopReq <-chan chan<- struct{}

	newMoveRequest = moveReqCh
	newUseRequest = useReqCh

	sendMoveCmd = moveCmdCh
	sendUseCmd = useCmdCh

	newState = outputCh
	stopReq = stopCh

	go func() {
		var hasStopped chan<- struct{}

		cmd := struct {
			moveCmd *moveCmd
			useCmd  *useCmd
		}{}

		updateMoveCmdWith := func(r moveRequest) {
			switch r.moveRequestType {
			case MR_MOVE:
				cmd.moveCmd = &moveCmd{
					Time:      r.Time,
					Direction: r.Direction,
				}
			case MR_MOVE_CANCEL:
				if cmd.moveCmd != nil {
					if cmd.moveCmd.Direction == r.Direction {
						cmd.moveCmd = nil
					}
				}
			}
		}

		updateUseCmdWith := func(r useRequest) {
			switch r.useRequestType {
			case UR_USE:
				cmd.useCmd = &useCmd{
					Time:  r.Time,
					skill: r.skill,
				}

			case UR_USE_CANCEL:
				if cmd.useCmd != nil {
					if cmd.useCmd.skill == r.skill {
						cmd.useCmd = nil
					}
				}
			}
		}

	unlocked:
		// # This select prioritizes the following events.
		// ## 2 potential events to respond to
		// 1. ReadMoveCmd() method requests the actor's movement cmd
		// 2. ReadUseCmd() method requests the actor's use cmd
		// 3. stopIO() method has been called
		select {
		case sendMoveCmd <- cmd.moveCmd:
			goto locked
		case sendUseCmd <- cmd.useCmd:
			goto locked

		case hasStopped = <-stopReq:
			goto exit
		default:
		}

		// ## 3 potential events to respond to
		// 1. SubmitCmd() method has been called with a new move/use request
		// 2. ReadMoveCmd() method requests the actor's movement cmd
		// 3. ReadUseCmd() method requests the actor's use cmd
		// 4. stopIO() method has been called
		select {
		case r := <-newMoveRequest:
			updateMoveCmdWith(r)
			goto unlocked
		case r := <-newUseRequest:
			updateUseCmdWith(r)
			goto unlocked

		case sendMoveCmd <- cmd.moveCmd:
			goto locked
		case sendUseCmd <- cmd.useCmd:
			goto locked

		case hasStopped = <-stopReq:
			goto exit
		}

		panic("unclosed case in unlocked state select")

	locked:
		// Accepting and processing input commands is now on hold

		// ## 3 potential events to respond to
		// 1. WriteState() method has been called with a new world state
		// 2. ReadMoveCmd() method requests the actor's move command
		// 3. ReadUseCmd() method requests the actor's use command
		// 4. stopIO() method has been called
		select {
		case state := <-newState:
			if state != nil {
				a.SendJson("update", state)
			}
			goto unlocked

		case sendMoveCmd <- cmd.moveCmd:
			goto locked
		case sendUseCmd <- cmd.useCmd:
			goto locked

		case hasStopped = <-stopReq:
			goto exit
		}

		panic("unclosed case in locked state select")

	exit:
		hasStopped <- struct{}{}
	}()
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

	default:
		return fmt.Errorf("unknown command: %s", parts[0])
	}

	return nil
}

func (c actorConn) ReadMoveCmd() *moveCmd {
	return <-c.readMoveCmd
}

func (c actorConn) ReadUseCmd() *useCmd {
	return <-c.readUseCmd
}

// Culls the world state to the actor's viewport.
// Is called before actorConn.WriteState()
func (a *actor) WriteState(state rpg2d.WorldState) {
	c := a.Cell()

	state = state.Cull(coord.Bounds{
		c.Add(-26, 26),
		c.Add(26, -26),
	})

	a.actorConn.WriteState(state)
}

// Diffs the world state so only the changes are sent.
// Is called after actor.WriteState(). Expects the state
// to have been culled already.
func (a *actorConn) WriteState(state rpg2d.WorldState) {
	diff := a.lastState.Diff(state)
	a.lastState = state

	// Will need this when I start comparing for terrain type changes
	// a.lastState.Prepare()

	if len(diff.Entities) > 0 || len(diff.Removed) > 0 || diff.TerrainMap != nil {
		diff.Prepare()
		a.sendState <- &diff
	} else {
		a.sendState <- nil
	}
}

func (a actorConn) stopIO() {
	hasStopped := make(chan struct{})

	a.stop <- hasStopped
	<-hasStopped
}
