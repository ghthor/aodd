package game

import (
	"github.com/ghthor/filu/rpg2d"
	"github.com/ghthor/filu/rpg2d/coord"
	"github.com/ghthor/filu/rpg2d/quad/quadstate"
	"github.com/ghthor/filu/rpg2d/worldstate"
	"github.com/ghthor/filu/rpg2d/worldterrain"
	"github.com/ghthor/filu/sim/stime"
)

type InitialStateWriter interface {
	WriteWorldState(*worldstate.Snapshot) DiffWriter
}

type DiffWriter interface {
	WriteWorldStateDiff(*worldstate.Update)
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
	sendState chan<- *worldstate.Snapshot
	sendDiff  chan<- *worldstate.Update

	// Comm interface to muxer used by stopIO() method
	stop chan<- chan<- struct{}

	// External connection used to publish the initial world state
	conn InitialStateWriter

	diffWriter DiffWriter

	initialState *worldstate.Snapshot
	prevState    *worldstate.Snapshot
	nextState    *worldstate.Snapshot

	diff *worldstate.Update
}

func newActorConn(conn InitialStateWriter) actorConn {
	return actorConn{
		conn: conn,
		diff: worldstate.NewUpdate(1),
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

	stateOutputCh := make(chan *worldstate.Snapshot)
	diffOutputCh := make(chan *worldstate.Update)
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

	var newState <-chan *worldstate.Snapshot
	var newDiff <-chan *worldstate.Update
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
					diffWriter = a.conn.WriteWorldState(state)
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
				diffWriter.WriteWorldStateDiff(diff)
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
				diffWriter.WriteWorldStateDiff(diff)
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

// TODO Remove this method requirement from rpg2d.Actor interface
func (a *actor) WriteState(state rpg2d.WorldState) {
}

// TODO Remove this method requirement from rpg2d.Actor interface
func (a *actorConn) WriteState(state rpg2d.WorldState) {
}

// TODO Port all this state initialization work into filu
//      Actor will need to be able to provide the bounds for this to happen.
func (a *actor) WriteStateNext(now stime.Time, quad quadstate.Quad, terrain *worldterrain.MapState, encoder chan<- quadstate.EncodingRequest) {
	bounds := ActorCullBounds(a.Cell())

	if a.initialState == nil {
		state := worldstate.NewSnapshot(now, bounds, 0)
		quad.QueryBounds(bounds, state, quadstate.QueryAll)
		state.TerrainMap = &worldterrain.MapState{Map: terrain.Map.Slice(bounds)}

		a.prevState = state.Clone()
		a.nextState = state.Clone()
		a.diff = worldstate.NewUpdate(1)

		a.actorConn.WriteStateNext(state, encoder)
	} else {
		a.nextState.Clear()
		a.nextState.Time = now
		a.nextState.Bounds = bounds
		quad.QueryBounds(bounds, a.nextState, quadstate.QueryDiff)
		a.nextState.TerrainMap = &worldterrain.MapState{Map: terrain.Map.Slice(bounds)}

		a.actorConn.WriteStateNext(a.nextState, encoder)
	}
}

func (a *actorConn) WriteStateNext(state *worldstate.Snapshot, encoder chan<- quadstate.EncodingRequest) {
	if a.initialState == nil {
		a.initialState = state
		// TODO Enable encoding cachce
		// CacheEncodingsFor([][]*quadstate.Entity{state.Removed, state.New, state.Changed, state.Unchanged}, encoder)

		a.sendState <- a.initialState
		// Only 1 world state will ever be written
		close(a.sendState)
		a.sendState = nil

		// // TODO Enable bypassing the startIO loop and just go straight to writing
		// a.diffWriter = a.conn.WriteWorldState(state)
	} else {
		a.nextState = state
		a.diff.FromSnapshot(a.prevState, a.nextState)

		// TODO Enable encoding cache
		// CacheEncodingsFor([][]*quadstate.Entity{a.diff.Removed, a.diff.Entities}, encoder)

		if len(a.diff.Entities) > 0 || len(a.diff.Removed) > 0 || a.diff.TerrainMapSlices != nil {
			a.sendDiff <- a.diff
			// TODO Enable bypassing the startIO loop and just go straight to writing
			// a.diffWriter.WriteWorldStateDiff(a.diff)
		} else {
			a.sendDiff <- nil
		}
	}

	a.prevState, a.nextState = a.nextState, a.prevState
}

// TODO move this to a more apprioate place
func CacheEncodingsFor(entities [][]*quadstate.Entity, encoder chan<- quadstate.EncodingRequest) {
	done := make(chan [][]*quadstate.Entity)
	encoder <- quadstate.EncodingRequest{entities, done}
	<-done
	close(done)
}
