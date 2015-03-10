package client

import (
	"fmt"
	"io"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/engine/rpg2d"
)

type LoginConn interface {
	AttemptLogin(name, password string) LoginRoundTrip
	CreateActor(name, password string) CreateRoundTrip
}

type loginConn struct {
	conn game.GobConn
}

type requestSender struct {
	conn game.GobConn
}

func NewLoginConn(with io.ReadWriter) LoginConn {
	return &loginConn{
		conn: game.NewGobConn(with),
	}
}

type Conn interface {
	SendMoveRequest(game.MoveRequest)
	SendUseRequest(game.UseRequest)
	SendChatRequest(game.ChatRequest)
}

type InitialState struct {
	// The entity that represents the actor in the world.
	Entity game.ActorEntityState

	// The initial state of the world.
	WorldState rpg2d.WorldState

	// A channel that will recv updates to the world onA.
	Updates <-chan rpg2d.WorldStateDiff

	// A channel if there was an error reading an update
	Error <-chan error
}

type RespLoggedIn struct {
	Conn
	InitialState <-chan InitialState
	Error        <-chan error
}

// Represents a login request -> response roundtrip.
// The caller should select from all the channels to
// recv the response.
type LoginRoundTrip struct {
	conn game.GobConn

	Success          <-chan RespLoggedIn
	ActorDoesntExist <-chan game.RespActorDoesntExist
	AuthFailed       <-chan game.RespAuthFailed
	Error            <-chan error
}

func (trip LoginRoundTrip) run(r game.ReqLogin) LoginRoundTrip {
	var (
		success          chan<- RespLoggedIn
		actorDoesntExist chan<- game.RespActorDoesntExist
		authFailed       chan<- game.RespAuthFailed
		hadError         chan<- error
	)

	closeChans := func() func() {
		var (
			successCh          = make(chan RespLoggedIn, 1)
			actorDoesntExistCh = make(chan game.RespActorDoesntExist, 1)
			authFailedCh       = make(chan game.RespAuthFailed, 1)
			errorCh            = make(chan error, 1)
		)

		trip.Success, success =
			successCh, successCh
		trip.ActorDoesntExist, actorDoesntExist =
			actorDoesntExistCh, actorDoesntExistCh
		trip.AuthFailed, authFailed =
			authFailedCh, authFailedCh
		trip.Error, hadError =
			errorCh, errorCh

		return func() {
			close(successCh)
			close(actorDoesntExistCh)
			close(authFailedCh)
			close(errorCh)
		}
	}()

	go func() {
		defer closeChans()

		err := trip.conn.EncodeAndSend(game.ET_REQ_LOGIN, r)
		if err != nil {
			hadError <- err
			return
		}

		eType, err := trip.conn.ReadNextType()
		if err != nil {
			hadError <- err
			return
		}

		switch eType {
		case game.ET_RESP_AUTH_FAILED:
			var r game.RespAuthFailed
			err := trip.conn.Decode(&r)
			if err != nil {
				hadError <- err
				return
			}

			authFailed <- r

		case game.ET_RESP_ACTOR_DOESNT_EXIST:
			var r game.RespActorDoesntExist
			err := trip.conn.Decode(&r)
			if err != nil {
				hadError <- err
				return
			}

			actorDoesntExist <- r

		case game.ET_RESP_LOGIN_SUCCESS:
			var actorEntity game.ActorEntityState
			err := trip.conn.Decode(&actorEntity)
			if err != nil {
				hadError <- err
				return
			}

			loggedIn(trip.conn, actorEntity, success)

		default:
			hadError <- fmt.Errorf("unexpected login request resp type: %v", eType)
		}
	}()

	return trip
}

func (c *loginConn) AttemptLogin(name, password string) LoginRoundTrip {
	return LoginRoundTrip{conn: c.conn}.run(game.ReqLogin{name, password})
}

// Represents a create request -> response roundtrip.
// The caller should select from all the channels to
// recv the response.
type CreateRoundTrip struct {
	conn game.GobConn

	Success     <-chan RespLoggedIn
	ActorExists <-chan game.RespActorExists
	Error       <-chan error
}

func (trip CreateRoundTrip) run(r game.ReqCreate) CreateRoundTrip {
	var (
		success     chan<- RespLoggedIn
		actorExists chan<- game.RespActorExists
		hadError    chan<- error
	)

	closeChans := func() func() {
		var (
			successCh     = make(chan RespLoggedIn, 1)
			actorExistsCh = make(chan game.RespActorExists, 1)
			errorCh       = make(chan error, 1)
		)

		trip.Success, success =
			successCh, successCh
		trip.ActorExists, actorExists =
			actorExistsCh, actorExistsCh
		trip.Error, hadError =
			errorCh, errorCh

		return func() {
			close(successCh)
			close(actorExistsCh)
			close(errorCh)
		}
	}()

	go func() {
		defer closeChans()

		err := trip.conn.EncodeAndSend(game.ET_REQ_CREATE, r)
		if err != nil {
			hadError <- err
			return
		}

		eType, err := trip.conn.ReadNextType()
		if err != nil {
			hadError <- err
			return
		}

		switch eType {
		case game.ET_RESP_ACTOR_EXISTS:
			var r game.RespActorExists
			err := trip.conn.Decode(&r)
			if err != nil {
				hadError <- err
				return
			}

			actorExists <- r

		case game.ET_RESP_CREATE_SUCCESS:
			var actorEntity game.ActorEntityState
			err := trip.conn.Decode(&actorEntity)
			if err != nil {
				hadError <- err
				return
			}

			loggedIn(trip.conn, actorEntity, success)

		default:
			hadError <- fmt.Errorf("unexpected create request resp type: %v", eType)
		}
	}()

	return trip
}

func (c *loginConn) CreateActor(name, password string) CreateRoundTrip {
	return CreateRoundTrip{conn: c.conn}.run(game.ReqCreate{name, password})
}

func loggedIn(conn game.GobConn, entity game.ActorEntityState, success chan<- RespLoggedIn) {
	var (
		recvInitialState <-chan InitialState
		sendInitialState chan<- InitialState

		hadErrorReceivingState chan<- error
		wasErrorReceivingState <-chan error
	)

	closeInitialStateCh, closeErrorCh := func() (func(), func()) {
		initialStateCh := make(chan InitialState)
		errorCh := make(chan error, 1)

		sendInitialState, recvInitialState =
			initialStateCh, initialStateCh
		wasErrorReceivingState, hadErrorReceivingState =
			errorCh, errorCh

		return func() {
				close(initialStateCh)
			}, func() {
				close(errorCh)
			}
	}()

	success <- RespLoggedIn{
		Conn:         requestSender{conn},
		InitialState: recvInitialState,
		Error:        wasErrorReceivingState,
	}

	go func() {
		defer closeErrorCh()

		recv := initialStateReceiver{
			conn:   conn,
			entity: entity,

			sendInitialState: sendInitialState,
		}

		err := recv.run()
		if err != nil {
			hadErrorReceivingState <- err

			go func() {
				defer closeInitialStateCh()
				sendInitialState <- InitialState{
					Entity: entity,
				}
			}()
		}
	}()
}

type stateFn func() (stateFn, error)

type initialStateReceiver struct {
	conn game.GobConn

	entity game.ActorEntityState
	state  rpg2d.WorldState

	sendInitialState chan<- InitialState
}

func (recv *initialStateReceiver) run() (err error) {
	f := recv.initialState
	for f != nil && err == nil {
		f, err = f()
	}

	return err
}

func (recv *initialStateReceiver) initialState() (stateFn, error) {
	eType, err := recv.conn.ReadNextType()
	if err != nil {
		return nil, err
	}

	switch eType {
	default:
		return nil, fmt.Errorf("unexpected encoded type %v waiting for initial state", eType)

	case game.ET_WORLD_STATE:
	}

	var state rpg2d.WorldState
	err = recv.conn.Decode(&state)
	if err != nil {
		return nil, err
	}

	recv.state = state

	return recv.receivedInitialState, nil
}

// Starts the WorldStateDiff read loop
func (recv *initialStateReceiver) receivedInitialState() (stateFn, error) {
	var (
		sendUpdate chan<- rpg2d.WorldStateDiff
		newUpdate  <-chan rpg2d.WorldStateDiff

		wasErrorRecvStateUpdate chan<- error
		hadErrorRecvStateUpdate <-chan error
	)

	closeChans := func() func() {
		updateCh := make(chan rpg2d.WorldStateDiff)
		errorCh := make(chan error)

		sendUpdate, newUpdate =
			updateCh, updateCh
		wasErrorRecvStateUpdate, hadErrorRecvStateUpdate =
			errorCh, errorCh

		return func() {
			close(updateCh)
			close(errorCh)
		}
	}()

	go func() {
		defer closeChans()

		// Starts an infinite read loop
		err := receiveUpdates(recv.conn, sendUpdate)
		if err != nil {
			wasErrorRecvStateUpdate <- err
		}
	}()

	return recv.mergeUpdates(newUpdate, hadErrorRecvStateUpdate), nil
}

// Create a closure that will infinite loop,
// merging state diffs with the initial world
// state. It terminates when the initial state
// if requested.
func (recv *initialStateReceiver) mergeUpdates(newUpdate <-chan rpg2d.WorldStateDiff, hadError <-chan error) stateFn {
	var mergeUpdates stateFn

	mergeUpdates = func() (stateFn, error) {
		select {
		case recv.sendInitialState <- InitialState{
			Entity:     recv.entity,
			WorldState: recv.state,
			Updates:    newUpdate,
			Error:      hadError,
		}:

			return nil, nil

		case err := <-hadError:
			return nil, err

		case diff := <-newUpdate:
			fmt.Println(diff)
		}

		// TODO merge diff into state

		return mergeUpdates, nil
	}

	return mergeUpdates
}

func receiveUpdates(conn game.GobConn, sendUpdate chan<- rpg2d.WorldStateDiff) (err error) {
	for err == nil {
		err = receiveUpdate(conn, sendUpdate)
	}
	return
}

func receiveUpdate(conn game.GobConn, sendUpdate chan<- rpg2d.WorldStateDiff) error {
	eType, err := conn.ReadNextType()
	if err != nil {
		return err
	}

	switch eType {
	default:
		return fmt.Errorf("unexpected encoded type %v waiting for state update diffs", eType)

	case game.ET_WORLD_STATE_DIFF:
	}

	var diff rpg2d.WorldStateDiff

	err = conn.Decode(&diff)
	if err != nil {
		return err
	}

	sendUpdate <- diff

	return nil
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
