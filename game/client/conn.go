package client

import (
	"fmt"
	"io"

	"github.com/ghthor/aodd/game"
)

type Conn interface {
	AttemptLogin(name, password string) LoginRoundTrip
	CreateActor(name, password string) CreateRoundTrip
}

type conn struct {
	game.GobConn
}

func NewConn(with io.ReadWriter) Conn {
	return &conn{
		GobConn: game.NewGobConn(with),
	}
}

// Represents a login request -> response roundtrip.
// The caller should select from all the channels to
// recv the response.
type LoginRoundTrip struct {
	conn *conn

	Success          <-chan game.ActorEntityState
	ActorDoesntExist <-chan game.RespActorDoesntExist
	AuthFailed       <-chan game.RespAuthFailed
	Error            <-chan error
}

func (trip LoginRoundTrip) run(r game.ReqLogin) LoginRoundTrip {
	var (
		success          chan<- game.ActorEntityState
		actorDoesntExist chan<- game.RespActorDoesntExist
		authFailed       chan<- game.RespAuthFailed
		hadError         chan<- error
	)

	closeChans := func() func() {
		var (
			successCh          = make(chan game.ActorEntityState, 1)
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

			success <- actorEntity

		default:
			hadError <- fmt.Errorf("unexpected login request resp type: %v", eType)
		}
	}()

	return trip
}

func (c *conn) AttemptLogin(name, password string) LoginRoundTrip {
	return LoginRoundTrip{conn: c}.run(game.ReqLogin{name, password})
}

// Represents a create request -> response roundtrip.
// The caller should select from all the channels to
// recv the response.
type CreateRoundTrip struct {
	conn *conn

	Success     <-chan game.ActorEntityState
	ActorExists <-chan game.RespActorExists
	Error       <-chan error
}

func (trip CreateRoundTrip) run(r game.ReqCreate) CreateRoundTrip {
	var (
		success     chan<- game.ActorEntityState
		actorExists chan<- game.RespActorExists
		hadError    chan<- error
	)

	closeChans := func() func() {
		var (
			successCh     = make(chan game.ActorEntityState, 1)
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

			success <- actorEntity

		default:
			hadError <- fmt.Errorf("unexpected create request resp type: %v", eType)
		}
	}()

	return trip
}

func (c *conn) CreateActor(name, password string) CreateRoundTrip {
	return CreateRoundTrip{conn: c}.run(game.ReqCreate{name, password})
}
