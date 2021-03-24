package game

import (
	"github.com/ghthor/filu/rpg2d"
	"github.com/ghthor/filu/rpg2d/coord"
	"github.com/ghthor/filu/rpg2d/entity"
)

type InitialStateWriter interface {
	WriteWorldState(rpg2d.WorldState) DiffWriter
}

type DiffWriter interface {
	WriteWorldStateDiff(rpg2d.WorldStateDiff)
}

type actorConn struct {
	// Comm interface to muxer used by SubmitCmd() method
	submitMoveRequest chan<- MoveRequest
	submitUseRequest  chan<- UseRequest
	submitChatRequest chan<- ChatRequest

	readMoveCmd <-chan *moveCmd
	readUseCmd  <-chan *useCmd
	readChatCmd <-chan *chatCmd

	// Comm interface to muxer used to send world states
	sendState chan<- *rpg2d.WorldState
	sendDiff  chan<- *rpg2d.WorldStateDiff

	// Comm interface to muxer used by stopIO() method
	stop chan<- chan<- struct{}

	// External connection used to publish the initial world state
	conn InitialStateWriter

	initialState *rpg2d.WorldState
	prevState    rpg2d.WorldState
	nextState    rpg2d.WorldState

	diff rpg2d.WorldStateDiff
}

func newActorConn(conn InitialStateWriter) actorConn {
	return actorConn{
		conn: conn,
		prevState: rpg2d.WorldState{
			Entities: make(entity.StateSlice, 0, 1),
		},
		nextState: rpg2d.WorldState{
			Entities: make(entity.StateSlice, 0, 1),
		},
		diff: rpg2d.WorldStateDiff{
			Entities: make(entity.StateSlice, 0, 1),
			Removed:  make(entity.StateSlice, 0, 1),
		},
	}
}

func (a *actorConn) startIO() {
	// Setup communication channels
	moveReqCh := make(chan MoveRequest, 2)
	useReqCh := make(chan UseRequest, 2)
	chatReqCh := make(chan ChatRequest, 2)

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
	var newMoveRequest <-chan MoveRequest
	var newUseRequest <-chan UseRequest
	var newChatRequest <-chan ChatRequest

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

		updateMoveCmdWith := func(r MoveRequest) {
			switch r.MoveRequestType {
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

		updateUseCmdWith := func(r UseRequest) {
			switch r.UseRequestType {
			case UR_USE:
				cmd.useCmd = &useCmd{
					Time:  r.Time,
					skill: r.Skill,
				}

			case UR_USE_CANCEL:
				if cmd.useCmd != nil {
					if cmd.useCmd.skill == r.Skill {
						cmd.useCmd = nil
					}
				}
			}
		}

		updateChatCmdWith := func(r ChatRequest) {
			chatCmd := chatCmd{
				ChatRequestType: r.ChatRequestType,
				Time:            r.Time,
				msg:             r.Msg,
			}
			cmd.chatCmd = &chatCmd
		}

		var diffWriter DiffWriter

		// Wait for the initial world state
		// and send it out to the client.
		for {
			select {
			case sendMoveCmd <- cmd.moveCmd:
			case sendUseCmd <- cmd.useCmd:
			case sendChatCmd <- cmd.chatCmd:
				cmd.chatCmd = nil
			case state := <-newState:
				if state != nil {
					diffWriter = a.conn.WriteWorldState(*state)
					// Only 1 world state will ever be written
					a.conn = nil
				}

				goto unlocked

			case hasStopped = <-stopReq:
				goto exit
			}
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
			cmd.chatCmd = nil
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

		case diff := <-newDiff:
			if diff != nil {
				diffWriter.WriteWorldStateDiff(*diff)
			}

			goto unlocked

		case sendMoveCmd <- cmd.moveCmd:
			goto locked
		case sendUseCmd <- cmd.useCmd:
			goto locked
		case sendChatCmd <- cmd.chatCmd:
			cmd.chatCmd = nil
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
		case diff := <-newDiff:
			if diff != nil {
				diffWriter.WriteWorldStateDiff(*diff)
			}

			goto unlocked

		case sendMoveCmd <- cmd.moveCmd:
			goto locked
		case sendUseCmd <- cmd.useCmd:
			goto locked
		case sendChatCmd <- cmd.chatCmd:
			cmd.chatCmd = nil
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

func ActorCullBounds(center coord.Cell) coord.Bounds {
	return coord.Bounds{
		center.Add(-26, 26),
		center.Add(26, -26),
	}
}

// This is the first stage in writing out the state where we cull the
// state down by the viewport bounds of an actor.
func (a *actor) WriteState(state rpg2d.WorldState) {
	a.actorConn.WriteState(
		state.CullInto(a.actorConn.nextState, ActorCullBounds(a.Cell())),
	)
}

func (a *actorConn) WriteState(state rpg2d.WorldState) {
	a.nextState = state

	if a.initialState == nil {
		initialState := state.Clone()
		a.initialState = &initialState
		a.sendState <- a.initialState

		// Only 1 world state will ever be written
		close(a.sendState)
		a.sendState = nil
	} else {
		a.diff.Between(a.prevState, a.nextState)

		if len(a.diff.Entities) > 0 || len(a.diff.Removed) > 0 || a.diff.TerrainMapSlices != nil {
			a.sendDiff <- &a.diff
		} else {
			a.sendDiff <- nil
		}
	}

	a.prevState, a.nextState = a.nextState, a.prevState
}
