package client

import (
	"fmt"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/engine/rpg2d"
)

type LoggedInConn interface {
	// Signals the server to connect the actor
	// into simulation. This extra step after
	// a successful login enables the client to
	// prepare the renderer, load all the assets,
	// before the actor is actually placed into the
	// world and becomes vulnerable.
	ConnectActor(name string) ConnectRoundTrip
}

// Implementation of the LoggedInConn interface
type actorConnector struct {
	conn game.Conn
}

type RespConnected struct {
	UpdateConn
	InputConn
	InitialState InitialState
}

type ConnectRoundTrip struct {
	conn game.Conn

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
			UpdateConn: updateReceiver{trip.conn},
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
