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

// An implementation of the LoginConn interface
type loginConn struct {
	conn game.GobConn
}

func NewLoginConn(with io.ReadWriter) LoginConn {
	return &loginConn{
		conn: game.NewGobConn(with),
	}
}

type LoggedInConn interface {
	// Signals the server to connect the actor
	// into simulation. This extra step after
	// a successful login enables the client to
	// prepare the renderer, load all the assets,
	// before the actor is actually placed into the
	// world and becomes vulnerable.
	ConnectActor(name string) ConnectRoundTrip
}

type actorConnector struct {
	conn game.GobConn
}

type RespLoggedIn struct {
	Name string
	LoggedInConn
}

type UpdateConn interface {
	NextUpdate() (rpg2d.WorldStateDiff, error)
}

type InputConn interface {
	SendMoveRequest(game.MoveRequest)
	SendUseRequest(game.UseRequest)
	SendChatRequest(game.ChatRequest)
}

// An implementation of the ConnectedConn interface
type requestSender struct {
	conn game.GobConn
}

type InitialState struct {
	// The entity that represents the actor in the world.
	Entity game.ActorEntityState

	// The initial state of the world.
	WorldState rpg2d.WorldState
}

type RespConnected struct {
	UpdateConn
	InputConn
	InitialState InitialState
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
			var resp game.RespLoginSuccess
			err := trip.conn.Decode(&resp)
			if err != nil {
				hadError <- err
				return
			}

			success <- RespLoggedIn{
				Name:         resp.Name,
				LoggedInConn: actorConnector{trip.conn},
			}

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
			var resp game.RespCreateSuccess
			err := trip.conn.Decode(&resp)
			if err != nil {
				hadError <- err
				return
			}

			success <- RespLoggedIn{
				Name:         resp.Name,
				LoggedInConn: actorConnector{trip.conn},
			}

		default:
			hadError <- fmt.Errorf("unexpected create request resp type: %v", eType)
		}
	}()

	return trip
}

func (c *loginConn) CreateActor(name, password string) CreateRoundTrip {
	return CreateRoundTrip{conn: c.conn}.run(game.ReqCreate{name, password})
}

type ConnectRoundTrip struct {
	conn game.GobConn

	Connected <-chan RespConnected
	Error     <-chan error
}

func (trip ConnectRoundTrip) run(r game.ReqConnect) ConnectRoundTrip {
	var (
		actorConnected chan<- RespConnected
		hadError       chan<- error
	)

	var closeChans func() = func() func() {
		connectedCh := make(chan RespConnected)
		errorCh := make(chan error, 1)

		trip.Connected, actorConnected =
			connectedCh, connectedCh
		trip.Error, hadError =
			errorCh, errorCh

		return func() {
			close(connectedCh)
			close(errorCh)
		}
	}()

	go func() {
		defer closeChans()

		err := trip.conn.EncodeAndSend(game.ET_REQ_CONNECT, r)
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
		default:
			hadError <- fmt.Errorf("unexpected encoded type{%v} waiting for connected entity", eType)
			return

		case game.ET_CONNECTED:
		}

		var actorEntity game.ActorEntityState

		err = trip.conn.Decode(&actorEntity)
		if err != nil {
			hadError <- err
			return
		}

		eType, err = trip.conn.ReadNextType()
		if err != nil {
			hadError <- err
			return
		}

		switch eType {
		default:
			hadError <- fmt.Errorf("unexpected encoded type{%v} waiting for initial state", eType)
			return

		case game.ET_WORLD_STATE:
		}

		var state rpg2d.WorldState

		err = trip.conn.Decode(&state)
		if err != nil {
			hadError <- err
			return
		}

		actorConnected <- RespConnected{
			// TODO Return an UpdateConn
			UpdateConn: nil,
			InputConn:  requestSender{trip.conn},
			InitialState: InitialState{
				Entity:     actorEntity,
				WorldState: state,
			},
		}
	}()

	return trip
}

func (c actorConnector) ConnectActor(name string) ConnectRoundTrip {
	return ConnectRoundTrip{conn: c.conn}.run(game.ReqConnect{name})
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
