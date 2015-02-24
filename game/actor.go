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

type actorCmd struct {
	timeIssued stime.Time
	cmd        string
	params     string
}

type moveRequest struct {
	stime.Time
	coord.Direction
}

type actorCmdRequest struct {
	*moveRequest
}

type actorConn struct {
	// Comm interface to muxer used by WriteInput() method
	submitCmd chan<- actorCmd

	// Comm interface to muxer used by getEntity() method
	readCmdReq <-chan actorCmdRequest

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
	id int64

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
	EntityId int64 `json:"id"`

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
	actorCmdRequest
	actorEntity
	undoLastMoveAction func()

	actorConn
}

func (a *actor) Entity() entity.Entity { return a.actorEntity }
func (e actorEntity) Id() int64        { return e.id }
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

func (e actorEntityState) Id() int64            { return e.EntityId }
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
	prevMoveRequest := a.actorCmdRequest.moveRequest

	a.undoLastMoveAction = func() {
		a.pathAction = prevPathAction
		a.facing = prevFacing
		a.actorCmdRequest.moveRequest = prevMoveRequest
		a.undoLastMoveAction = nil
	}

	a.pathAction = pa
	a.facing = pa.Direction()
	a.actorCmdRequest.moveRequest = nil
}

func (a *actor) applyTurnAction(ta coord.TurnAction) {
	prevAction := a.lastMoveAction
	prevFacing := a.facing
	prevMoveRequest := a.actorCmdRequest.moveRequest

	a.undoLastMoveAction = func() {
		a.lastMoveAction = prevAction
		a.facing = prevFacing
		a.actorCmdRequest.moveRequest = prevMoveRequest
		a.undoLastMoveAction = nil
	}

	a.lastMoveAction = ta
	a.facing = ta.To
	a.actorCmdRequest.moveRequest = nil
}

func (a *actor) revertMoveAction() {
	if a.undoLastMoveAction != nil {
		a.undoLastMoveAction()
	}
}

func (a *actorConn) startIO() {
	// Setup communication channels
	cmdCh := make(chan actorCmd)
	cmdReqCh := make(chan actorCmdRequest)
	outputCh := make(chan *rpg2d.WorldState)
	stopCh := make(chan chan<- struct{})

	// Set the channels accessible to the outside world
	a.submitCmd = cmdCh
	a.readCmdReq = cmdReqCh
	a.sendState = outputCh
	a.stop = stopCh

	// Establish the channel endpoints used inside the go routine
	var newCmd <-chan actorCmd
	var sendCmdReq chan<- actorCmdRequest
	var newState <-chan *rpg2d.WorldState
	var stopReq <-chan chan<- struct{}

	newCmd = cmdCh
	sendCmdReq = cmdReqCh
	newState = outputCh
	stopReq = stopCh

	go func() {
		var hasStopped chan<- struct{}

		// Buffer of 1 used to store the most recently
		// received actor cmd from the network.
		var cmdReq actorCmdRequest

		updateCmdReqWith := func(c actorCmd) {
			switch c.cmd {
			case "move":
				// TODO This is a shit place to be having an error to deal with
				// TODO It needs to be dealt with at the packet handler level
				d, _ := coord.NewDirectionWithString(c.params)

				cmdReq.moveRequest = &moveRequest{
					Time:      stime.Time(c.timeIssued),
					Direction: d,
				}
			case "moveCancel":
				d, _ := coord.NewDirectionWithString(c.params)
				if cmdReq.moveRequest != nil {
					if cmdReq.moveRequest.Direction == d {
						cmdReq.moveRequest = nil
					}
				}
			}
		}

	unlocked:
		// # This select prioritizes the following events.
		// ## 2 potential events to respond to
		// 1. ReadCmdRequest() method requests the actor command request
		// 2. stopIO() method has been called
		select {
		case sendCmdReq <- cmdReq:
			// Transition: unlocked -> locked
			goto locked

		case hasStopped = <-stopReq:
			// Transition: unlocked -> exit
			goto exit
		default:
		}

		// ## 3 potential events to respond to
		// 1. SubmitInput() method has been called with a new command
		// 2. ReadCmdRequest() method requests the actor command request
		// 3. stopIO() method has been called
		select {
		case c := <-newCmd:
			updateCmdReqWith(c)

			// Transition: unlocked -> unlocked
			goto unlocked

		case sendCmdReq <- cmdReq:
			// Transition: unlocked -> locked
			goto locked

		case hasStopped = <-stopReq:
			// Transition: unlocked -> exit
			goto exit
		}

		panic("unclosed case in unlocked state select")

	locked:
		// Accepting and processing input commands is now on hold

		// ## 3 potential events to respond to
		// 1. WriteState() method has been called with a new world state
		// 2. ReadCmdRequest() method requests the actor command request.. again!
		// 3. stopIO() method has been called
		select {
		case state := <-newState:
			if state != nil {
				a.SendJson("update", state)
			}

			// Transition: locked -> unlocked
			goto unlocked

		case sendCmdReq <- cmdReq:
			// Transition: locked -> locked
			goto locked

		case hasStopped = <-stopReq:
			// Transition: locked -> exit
			goto exit
		}

		panic("unclosed case in locked state select")
	exit:
		hasStopped <- struct{}{}
	}()
}

func (c actorConn) SubmitCmd(cmd, params string) error {
	parts := strings.Split(cmd, "=")

	if len(parts) != 2 {
		return errors.New("invalid command")
	}

	timeIssued, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return err
	}

	c.submitCmd <- actorCmd{
		timeIssued: stime.Time(timeIssued),
		cmd:        parts[0],
		params:     params,
	}

	return nil
}

func (c actorConn) ReadCmdRequest() actorCmdRequest {
	return <-c.readCmdReq
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
