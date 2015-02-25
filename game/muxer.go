package game

import (
	"github.com/ghthor/engine/net/protocol"
	"github.com/ghthor/engine/rpg2d"
	"github.com/ghthor/engine/rpg2d/coord"
)

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

func (a actorConn) stopIO() {
	hasStopped := make(chan struct{})

	a.stop <- hasStopped
	<-hasStopped
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
