package game

import (
	"fmt"
	"log"

	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/engine/rpg2d"
	"github.com/ghthor/engine/rpg2d/entity"
)

// Used to determine the next type that's in the
// buffer so we can decode it into a real value.
// We'll decode an encoded type and switch on its
// value so we'll have the correct value to decode
// into.
type EncodedType int

//go:generate stringer -type=EncodedType
const (
	ET_ERROR EncodedType = iota
	ET_DISCONNECT

	ET_REQ_LOGIN
	ET_REQ_CREATE

	ET_RESP_AUTH_FAILED
	ET_RESP_ACTOR_EXISTS
	ET_RESP_ACTOR_DOESNT_EXIST

	ET_RESP_LOGIN_SUCCESS
	ET_RESP_CREATE_SUCCESS

	ET_REQ_CONNECT
	ET_CONNECTED

	ET_WORLD_STATE
	ET_WORLD_STATE_DIFF

	ET_REQ_MOVE
	ET_REQ_USE
	ET_REQ_CHAT
)

type Conn interface {
	EncodeAndSend(EncodedType, interface{}) error
	ReadNextType() (EncodedType, error)
	Decode(interface{}) error
}

type InputReceiver interface {
	SubmitMoveRequest(MoveRequest)
	SubmitUseRequest(UseRequest)
	SubmitChatRequest(ChatRequest)

	Close()
}

type serverConn struct {
	Conn

	datastore datastore.Datastore

	newActor func(datastore.Actor, InitialStateWriter) (InputReceiver, entity.State)
	actor    InputReceiver
}

type ActorConn interface {
	Run() error
}

type stateFn func() (stateFn, error)

func (c *serverConn) handleLogin() (stateFn, error) {
	eType, err := c.ReadNextType()
	if err != nil {
		return nil, err
	}

	switch eType {
	case ET_REQ_LOGIN:
		return c.handleLoginReq, nil
	case ET_REQ_CREATE:
		return c.handleCreateReq, nil

	default:
		log.Println("unexpected encoded type: ", eType)
	}

	return c.handleLogin, nil
}

func (c *serverConn) handleLoginReq() (stateFn, error) {
	var r ReqLogin
	err := c.Decode(&r)
	if err != nil {
		return nil, err
	}

	actor, exists := c.datastore.ActorExists(r.Name)
	if !exists {
		err := c.EncodeAndSend(ET_RESP_ACTOR_DOESNT_EXIST, RespActorDoesntExist{
			r.Name, r.Password,
		})
		if err != nil {
			return nil, err
		}

		return c.handleLogin, nil
	}

	if !actor.Authenticate(r.Name, r.Password) {
		err := c.EncodeAndSend(ET_RESP_AUTH_FAILED, RespAuthFailed{r.Name})
		if err != nil {
			return nil, err
		}

		return c.handleLogin, nil
	}

	err = c.EncodeAndSend(ET_RESP_LOGIN_SUCCESS, RespLoginSuccess{actor.Name})
	if err != nil {
		return nil, err
	}

	return c.handleConnect(actor), nil
}

func (c *serverConn) handleCreateReq() (stateFn, error) {
	var r ReqCreate
	err := c.Decode(&r)
	if err != nil {
		return nil, err
	}

	_, exists := c.datastore.ActorExists(r.Name)
	if exists {
		err := c.EncodeAndSend(ET_RESP_ACTOR_EXISTS, RespActorExists{r.Name})
		if err != nil {
			return nil, err
		}

		return c.handleLogin, nil
	}

	actor, err := c.datastore.AddActor(r.Name, r.Password)
	if err != nil {
		// TODO Instead of terminating the connection here
		//      we should retry contacting the database a
		//      few times
		return nil, err
	}

	err = c.EncodeAndSend(ET_RESP_CREATE_SUCCESS, RespLoginSuccess{actor.Name})
	if err != nil {
		return nil, err
	}

	return c.handleConnect(actor), nil
}

func (c *serverConn) handleConnect(dsactor datastore.Actor) stateFn {
	var handleConnect stateFn

	handleConnect = func() (stateFn, error) {
		eType, err := c.ReadNextType()
		if err != nil {
			return nil, err
		}

		switch eType {
		default:
			// TODO Send a protocol error to the client
			return handleConnect, nil

		case ET_REQ_CONNECT:
		}

		var r ReqConnect
		err = c.Decode(&r)
		if err != nil {
			return nil, err
		}

		actor, entity, initialState := c.connect(dsactor)
		c.actor = actor

		err = c.EncodeAndSend(ET_CONNECTED, entity)
		if err != nil {
			return nil, err
		}

		err = c.EncodeAndSend(ET_WORLD_STATE, <-initialState)
		if err != nil {
			return nil, err
		}

		return c.handleInputReq, nil
	}

	return handleConnect
}

type initialStateWriter struct {
	sendState chan<- rpg2d.WorldState
}

func (c initialStateWriter) WriteWorldState(s rpg2d.WorldState) StateWriter {
	c.sendState <- s

	// TODO Return a conncurent safe StateWriter
	return nil
}

func (c *serverConn) connect(dsactor datastore.Actor) (actor InputReceiver, entity entity.State, initialState <-chan rpg2d.WorldState) {
	initialStateCh := make(chan rpg2d.WorldState)
	actor, entity = c.newActor(dsactor, initialStateWriter{initialStateCh})
	return actor, entity, initialStateCh
}

func (c *serverConn) handleInputReq() (stateFn, error) {
	eType, err := c.ReadNextType()
	if err != nil {
		return nil, err
	}

	switch eType {
	default:
		return c.handleInputReq, fmt.Errorf("unexpected type %v when handling input", eType)

	case ET_REQ_MOVE:
		return c.handleMoveReq, nil
	case ET_REQ_USE:
		return c.handleUseReq, nil
	case ET_REQ_CHAT:
		return c.handleChatReq, nil
	}

	return c.handleInputReq, nil
}

func (c *serverConn) handleMoveReq() (stateFn, error) {
	var r MoveRequest
	err := c.Decode(&r)
	if err != nil {
		return nil, err
	}

	c.actor.SubmitMoveRequest(r)
	return c.handleInputReq, nil
}

func (c *serverConn) handleUseReq() (stateFn, error) {
	var r UseRequest
	err := c.Decode(&r)
	if err != nil {
		return nil, err
	}

	c.actor.SubmitUseRequest(r)
	return c.handleInputReq, nil
}

func (c *serverConn) handleChatReq() (stateFn, error) {
	var r ChatRequest
	err := c.Decode(&r)
	if err != nil {
		return nil, err
	}

	c.actor.SubmitChatRequest(r)
	return c.handleInputReq, nil
}

func (c serverConn) Run() (err error) {
	f := c.handleLogin
	for f != nil && err == nil {
		f, err = f()
	}

	if c.actor != nil {
		c.actor.Close()
	}

	return
}

func (c serverConn) WriteWorldStateDiff(s rpg2d.WorldStateDiff) error {
	return c.EncodeAndSend(ET_WORLD_STATE_DIFF, s)
}

func NewPreLoginConn(
	conn Conn,
	ds datastore.Datastore,
	newActor func(datastore.Actor, InitialStateWriter) (InputReceiver, entity.State)) ActorConn {
	return serverConn{
		Conn:      conn,
		datastore: ds,
		newActor:  newActor,
	}
}
