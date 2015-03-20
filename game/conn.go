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

	ET_RESP_ACTOR_ALREADY_CONNECTED
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

type PreLoginConn interface {
	HandleLogin() (LoggedInConn, error)
}

type ActorConnector func(datastore.Actor, InitialStateWriter) (InputReceiver, entity.State)

type LoggedInConn interface {
	HandleConnect(ActorConnector) (ConnectedActorConn, error)
}

type ConnectedActorConn interface {
	HandleIO() error
}

type preLoginConn struct {
	Conn
	datastore datastore.Datastore
}

type preLoginResult struct {
	preLoginConn

	// result set by handling packets
	loggedInActor datastore.Actor
}

type stateFn func() (stateFn, error)

func (c *preLoginResult) handleLogin() (stateFn, error) {
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

func (c *preLoginResult) handleLoginReq() (stateFn, error) {
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

	c.loggedInActor = actor

	err = c.EncodeAndSend(ET_RESP_LOGIN_SUCCESS, RespLoginSuccess{actor.Name})
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (c *preLoginResult) handleCreateReq() (stateFn, error) {
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

	c.loggedInActor = actor

	err = c.EncodeAndSend(ET_RESP_CREATE_SUCCESS, RespLoginSuccess{actor.Name})
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (c preLoginConn) HandleLogin() (LoggedInConn, error) {
	var err error
	result := preLoginResult{preLoginConn: c}

	f := result.handleLogin
	for f != nil && err == nil {
		f, err = f()
	}

	if err != nil {
		return nil, err
	}

	return loggedInConn{
		Conn:          c.Conn,
		loggedInActor: result.loggedInActor,
	}, nil
}

type loggedInConn struct {
	Conn
	loggedInActor datastore.Actor
}

type loggedInResult struct {
	loggedInConn

	connectedConn connectedConn
}

type initialStateWriter struct {
	sendState  chan<- rpg2d.WorldState
	diffWriter <-chan DiffWriter
}

func (c initialStateWriter) WriteWorldState(s rpg2d.WorldState) DiffWriter {
	// Pass state out to connection to be written
	c.sendState <- s
	// Only 1 world state will ever be written
	close(c.sendState)

	// Return a state writer to the muxer
	return <-c.diffWriter
}

func (c loggedInResult) connect(connectActor ActorConnector) (InputReceiver, entity.State, <-chan rpg2d.WorldState, chan<- DiffWriter) {
	initialStateCh := make(chan rpg2d.WorldState)
	diffWriterCh := make(chan DiffWriter)

	actor, entity := connectActor(c.loggedInActor, initialStateWriter{
		sendState:  initialStateCh,
		diffWriter: diffWriterCh,
	})
	return actor, entity, initialStateCh, diffWriterCh
}

func (c *loggedInResult) handleConnect(connectActor ActorConnector) stateFn {
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

		actor, entity, initialState, diffWriter := c.connect(connectActor)

		err = c.EncodeAndSend(ET_CONNECTED, entity)
		if err != nil {
			return nil, err
		}

		err = c.EncodeAndSend(ET_WORLD_STATE, <-initialState)
		if err != nil {
			return nil, err
		}

		c.connectedConn = connectedConn{
			Conn:  c.Conn,
			actor: actor,
		}

		diffWriter <- c.connectedConn

		return nil, nil
	}

	return handleConnect
}

func (c loggedInConn) HandleConnect(connectActor ActorConnector) (ConnectedActorConn, error) {
	var err error
	result := loggedInResult{loggedInConn: c}

	f := result.handleConnect(connectActor)
	for f != nil && err == nil {
		f, err = f()
	}

	if err != nil {
		return nil, err
	}

	return result.connectedConn, nil
}

type connectedConn struct {
	Conn
	actor InputReceiver
}

func (c *connectedConn) handleInputReq() (stateFn, error) {
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

func (c *connectedConn) handleMoveReq() (stateFn, error) {
	var r MoveRequest
	err := c.Decode(&r)
	if err != nil {
		return nil, err
	}

	c.actor.SubmitMoveRequest(r)
	return c.handleInputReq, nil
}

func (c *connectedConn) handleUseReq() (stateFn, error) {
	var r UseRequest
	err := c.Decode(&r)
	if err != nil {
		return nil, err
	}

	c.actor.SubmitUseRequest(r)
	return c.handleInputReq, nil
}

func (c *connectedConn) handleChatReq() (stateFn, error) {
	var r ChatRequest
	err := c.Decode(&r)
	if err != nil {
		return nil, err
	}

	c.actor.SubmitChatRequest(r)
	return c.handleInputReq, nil
}

func (c connectedConn) WriteWorldStateDiff(s rpg2d.WorldStateDiff) {
	// TODO Handle this potentional write error
	c.EncodeAndSend(ET_WORLD_STATE_DIFF, s)
}

func (c connectedConn) HandleIO() (err error) {
	f := c.handleInputReq
	for f != nil && err == nil {
		f, err = f()
	}

	c.actor.Close()

	return
}

func NewPreLoginConn(conn Conn, ds datastore.Datastore) PreLoginConn {
	return preLoginConn{
		Conn:      conn,
		datastore: ds,
	}
}

func RunServer(loginConn PreLoginConn, actorConnector ActorConnector) error {
	loggedInConn, err := loginConn.HandleLogin()
	if err != nil {
		return err
	}

	connectedConn, err := loggedInConn.HandleConnect(actorConnector)
	if err != nil {
		return err
	}

	return connectedConn.HandleIO()
}
