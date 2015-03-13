package client

import (
	"fmt"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/engine/rpg2d"
)

type UpdateConn interface {
	NextUpdate() (rpg2d.WorldStateDiff, error)
}

type InputConn interface {
	SendMoveRequest(game.MoveRequest)
	SendUseRequest(game.UseRequest)
	SendChatRequest(game.ChatRequest)
}

type InitialState struct {
	// The entity that represents the actor in the world.
	Entity game.ActorEntityState

	// The initial state of the world.
	WorldState rpg2d.WorldState
}

func receiveUpdates(conn game.Conn, sendUpdate chan<- rpg2d.WorldStateDiff) (err error) {
	var diff rpg2d.WorldStateDiff
	for {
		diff, err = receiveUpdate(conn)
		if err != nil {
			break
		}
		sendUpdate <- diff
	}
	return
}

func receiveUpdate(conn game.Conn) (diff rpg2d.WorldStateDiff, err error) {
	eType, err := conn.ReadNextType()
	if err != nil {
		return diff, err
	}

	switch eType {
	default:
		return diff, fmt.Errorf("unexpected encoded type %v waiting for state update diffs", eType)

	case game.ET_WORLD_STATE_DIFF:
	}

	err = conn.Decode(&diff)
	if err != nil {
		return diff, err
	}

	return diff, nil

}

// An implementation of the UpdateConn interface
type updateReceiver struct {
	conn game.Conn
}

func (c updateReceiver) NextUpdate() (rpg2d.WorldStateDiff, error) {
	return receiveUpdate(c.conn)
}

// An implementation of the InputConn interface
type requestSender struct {
	conn game.Conn
}

func (c requestSender) SendMoveRequest(r game.MoveRequest) {
	// TODO handle errors
	c.conn.EncodeAndSend(game.ET_REQ_MOVE, r)
}

func (c requestSender) SendUseRequest(r game.UseRequest) {
	// TODO handle errors
	c.conn.EncodeAndSend(game.ET_REQ_USE, r)
}

func (c requestSender) SendChatRequest(r game.ChatRequest) {
	// TODO handle errors
	c.conn.EncodeAndSend(game.ET_REQ_CHAT, r)
}
