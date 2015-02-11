package game

import (
	"errors"
	"strconv"
	"strings"

	"github.com/ghthor/engine/net/protocol"
	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/sim"
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
	sendState chan<- interface{}

	// Comm interface to muxer used by stopIO() method
	stop chan<- chan<- struct{}

	// External connection used to publish the world state
	protocol.Conn
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

	pathActions []coord.PathAction
}

type actor struct {
	*actorEntity
	actorConn
}

// Implement sim.Actor
func (a actor) Conn() sim.StateWriter {
	return sim.StateWriter(a)
}

func (e actorEntity) Id() int64        { return e.id }
func (e actorEntity) Cell() coord.Cell { return e.cell }
func (e actorEntity) Bounds() coord.Bounds {
	bounds := coord.Bounds{
		e.cell,
		e.cell,
	}

	for _, a := range e.pathActions {
		bounds = bounds.Join(a.Bounds())
	}

	return bounds
}

func (a *actorConn) startIO() {
	// Setup communication channels
	cmdCh := make(chan actorCmd)
	cmdReqCh := make(chan actorCmdRequest)
	outputCh := make(chan interface{})
	stopCh := make(chan chan<- struct{})

	// Set the channels accessible to the outside world
	a.submitCmd = cmdCh
	a.readCmdReq = cmdReqCh
	a.sendState = outputCh
	a.stop = stopCh

	// Establish the channel endpoints used inside the go routine
	var newCmd <-chan actorCmd
	var sendCmdReq chan<- actorCmdRequest
	var newState <-chan interface{}
	var stopReq <-chan chan<- struct{}

	newCmd = cmdCh
	sendCmdReq = cmdReqCh
	newState = outputCh
	stopReq = stopCh

	go func() {
		var hasStopped chan<- struct{}
		var cmdReq actorCmdRequest

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
		case _ = <-newCmd:
			// TODO update the entities movement state

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
		case _ = <-newState:
			// TODO send state out over connection

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

func (c actorConn) WriteState(s interface{}) error {
	c.sendState <- s
	return nil
}

func (a actorConn) stopIO() {
	hasStopped := make(chan struct{})

	a.stop <- hasStopped
	<-hasStopped
}

type warrior struct {
	actor
}

type wizard struct {
	actor
}

type priest struct {
	actor
}
