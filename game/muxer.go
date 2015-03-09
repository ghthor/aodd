package game

import (
	"github.com/ghthor/engine/rpg2d"
	"github.com/ghthor/engine/rpg2d/coord"
)

type stateWriter interface {
	WriteWorldState(rpg2d.WorldState) error
	WriteWorldStateDiff(rpg2d.WorldStateDiff) error
}

type actorConn struct {
	// Comm interface to muxer used by SubmitCmd() method
	submitMoveRequest chan<- moveRequest
	submitUseRequest  chan<- useRequest
	submitChatRequest chan<- chatRequest

	readMoveCmd <-chan *moveCmd
	readUseCmd  <-chan *useCmd
	readChatCmd <-chan *chatCmd

	// Comm interface to muxer used to send world states
	sendState chan<- *rpg2d.WorldState
	sendDiff  chan<- *rpg2d.WorldStateDiff

	// Comm interface to muxer used by stopIO() method
	stop chan<- chan<- struct{}

	// External connection used to publish the world state
	conn stateWriter

	lastState *rpg2d.WorldState
}

func newActorConn(conn stateWriter) actorConn {
	return actorConn{conn: conn}
}

func (a *actorConn) startIO() {
	// Setup communication channels
	moveReqCh := make(chan moveRequest)
	useReqCh := make(chan useRequest)
	chatReqCh := make(chan chatRequest)

	moveCmdCh := make(chan *moveCmd)
	useCmdCh := make(chan *useCmd)
	chatCmdCh := make(chan *chatCmd)

	stateOutputCh := make(chan *rpg2d.WorldState)
	diffOutputCh := make(chan *rpg2d.WorldStateDiff)
	stopCh := make(chan chan<- struct{})

	// Set the channels accessible to the outside world
	a.submitMoveRequest = moveReqCh
	a.submitUseRequest = useReqCh
	a.submitChatRequest = chatReqCh

	a.readMoveCmd = moveCmdCh
	a.readUseCmd = useCmdCh
	a.readChatCmd = chatCmdCh

	a.sendState = stateOutputCh
	a.sendDiff = diffOutputCh
	a.stop = stopCh

	// Establish the channel endpoints used inside the go routine
	var newMoveRequest <-chan moveRequest
	var newUseRequest <-chan useRequest
	var newChatRequest <-chan chatRequest

	var sendMoveCmd chan<- *moveCmd
	var sendUseCmd chan<- *useCmd
	var sendChatCmd chan<- *chatCmd

	var newState <-chan *rpg2d.WorldState
	var newDiff <-chan *rpg2d.WorldStateDiff
	var stopReq <-chan chan<- struct{}

	newMoveRequest = moveReqCh
	newUseRequest = useReqCh
	newChatRequest = chatReqCh

	sendMoveCmd = moveCmdCh
	sendUseCmd = useCmdCh
	sendChatCmd = chatCmdCh

	newState = stateOutputCh
	newDiff = diffOutputCh
	stopReq = stopCh

	go func() {
		var hasStopped chan<- struct{}

		cmd := struct {
			moveCmd *moveCmd
			useCmd  *useCmd
			chatCmd *chatCmd
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

		updateChatCmdWith := func(r chatRequest) {
			chatCmd := chatCmd(r)
			cmd.chatCmd = &chatCmd
		}

	unlocked:
		// # This select prioritizes the following events.
		// ## 2 potential events to respond to
		// 1. ReadMoveCmd() method requests the actor's movement cmd
		// 2. ReadUseCmd() method requests the actor's use cmd
		// 3. ReadChatCmd() method requests the actor's chat cmd
		// 4. stopIO() method has been called
		select {
		case sendMoveCmd <- cmd.moveCmd:
			goto locked
		case sendUseCmd <- cmd.useCmd:
			goto locked
		case sendChatCmd <- cmd.chatCmd:
			goto locked

		case hasStopped = <-stopReq:
			goto exit
		default:
		}

		// ## 3 potential events to respond to
		// 1. SubmitCmd() method has been called with a new move/use/chat request
		// 2. ReadMoveCmd() method requests the actor's movement cmd
		// 3. ReadUseCmd() method requests the actor's use cmd
		// 4. ReadChatCmd() method requests the actor's chat cmd
		// 5. stopIO() method has been called
		select {
		case r := <-newMoveRequest:
			updateMoveCmdWith(r)
			goto unlocked
		case r := <-newUseRequest:
			updateUseCmdWith(r)
			goto unlocked
		case r := <-newChatRequest:
			updateChatCmdWith(r)
			goto unlocked

		case sendMoveCmd <- cmd.moveCmd:
			goto locked
		case sendUseCmd <- cmd.useCmd:
			goto locked
		case sendChatCmd <- cmd.chatCmd:
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
		// 4. ReadChatCmd() method requests the actor's chat command
		// 5. stopIO() method has been called
		select {
		case state := <-newState:
			if state != nil {
				a.conn.WriteWorldState(*state)
			}

			cmd.chatCmd = nil
			goto unlocked

		case diff := <-newDiff:
			if diff != nil {
				a.conn.WriteWorldStateDiff(*diff)
			}

			cmd.chatCmd = nil
			goto unlocked

		case sendMoveCmd <- cmd.moveCmd:
			goto locked
		case sendUseCmd <- cmd.useCmd:
			goto locked
		case sendChatCmd <- cmd.chatCmd:
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
	if a.lastState == nil {
		a.lastState = &state
		a.sendState <- &state
		return
	}

	diff := a.lastState.Diff(state)
	a.lastState = &state

	if len(diff.Entities) > 0 || len(diff.Removed) > 0 || diff.TerrainMapSlices != nil {
		a.sendDiff <- &diff
	} else {
		a.sendDiff <- nil
	}
}
