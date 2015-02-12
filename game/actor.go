package game

import (
	"errors"
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
	moveRequest
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

	cell   coord.Cell
	facing coord.Direction

	pathAction *coord.PathAction
}

type actorEntityState struct {
	EntityId int64 `json:"id"`

	Name string `json:"name"`

	Facing string     `json:"facing"`
	Cell   coord.Cell `json:"cell"`
	bounds coord.Bounds

	PathAction *coord.PathAction `json:"pathAction"`
}

type actor struct {
	actorEntity
	actorConn
}

func (a actor) Entity() entity.Entity  { return a.actorEntity }
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
	return actorEntityState{
		EntityId: e.id,

		Cell:   e.cell,
		Facing: e.facing.String(),

		bounds: e.Bounds(),

		PathAction: e.pathAction,
	}
}

func (e actorEntityState) Id() int64            { return e.EntityId }
func (e actorEntityState) Bounds() coord.Bounds { return e.bounds }
func (e actorEntityState) IsDifferentFrom(other entity.State) bool {
	o := other.(actorEntityState)

	switch {
	case e.Facing != o.Facing:
		return true

	case e.PathAction != nil && o.PathAction != nil:
		if *e.PathAction != *o.PathAction {
			return true
		}

	case e.PathAction == nil && o.PathAction != nil:
		return true
	case e.PathAction != nil && o.PathAction == nil:
		return true
	}

	return false
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

				cmdReq.moveRequest = moveRequest{
					Time:      stime.Time(c.timeIssued),
					Direction: d,
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

			// Reset the cmdReq object
			cmdReq = actorCmdRequest{}

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

// Culls the world state to the actor's viewport
func (a *actor) WriteState(state rpg2d.WorldState) {
	c := a.Cell()

	state = state.Cull(coord.Bounds{
		c.Add(-26, 26),
		c.Add(26, -26),
	})

	a.actorConn.WriteState(state)
}

// Diffs the world state so only the changes are sent
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
