package client

import (
	"encoding/json"
	"io"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/engine/net/encoding"
	"github.com/ghthor/engine/net/protocol"
)

type Conn struct {
	protocol.Conn

	RespAuthFailed       chan<- RespAuthFailed
	RespActorDoesntExist chan<- RespActorDoesntExist

	RespLoginSuccess  chan<- game.ActorEntity
	RespCreateSuccess chan<- game.ActorEntity

	Packet chan<- encoding.Packet

	Error chan<- error
}

type RespAuthFailed struct{ Name string }
type RespActorDoesntExist struct{ Name, Password string }

func NewConn(with io.ReadWriteCloser) Conn {
	return Conn{
		Conn: protocol.NewConn(with),
	}
}

func (c Conn) AttemptLogin(name, password string) {
	c.SendJson(game.REQ_LOGIN.String(), game.LoginReq{name, password})
}

func (c Conn) CreateActor(name, password string) {
	c.SendJson(game.REQ_CREATE.String(), game.LoginReq{name, password})
}

type stateFn func() (stateFn, error)

func (c *Conn) handleLogin() (stateFn, error) {
	p, err := c.Read()
	if err != nil {
		return nil, err
	}

	if p.Type == encoding.PT_DISCONNECT {
		return nil, nil
	}

	switch p.Msg {
	case game.RESP_AUTH_FAILED.String():
		if c.RespAuthFailed != nil {
			c.RespAuthFailed <- RespAuthFailed{p.Payload}
		}

	case game.RESP_ACTOR_DOESNT_EXIST.String():
		if c.RespActorDoesntExist != nil {
			var loginReq game.LoginReq

			err := json.Unmarshal([]byte(p.Payload), &loginReq)
			if err != nil && c.Error != nil {
				c.Error <- err
				return c.handleLogin, nil
			}

			c.RespActorDoesntExist <- RespActorDoesntExist{loginReq.Name, loginReq.Password}
		}

	case game.RESP_LOGIN_SUCCESS.String():
		if c.RespLoginSuccess != nil {
			var actorEntity game.ActorEntity

			err := json.Unmarshal([]byte(p.Payload), &actorEntity)
			if err != nil && c.Error != nil {
				c.Error <- err
				return c.handleLogin, nil
			}

			c.RespLoginSuccess <- actorEntity
		}

	case game.RESP_CREATE_SUCCESS.String():
		if c.RespCreateSuccess != nil {
			var actorEntity game.ActorEntity

			err := json.Unmarshal([]byte(p.Payload), &actorEntity)
			if err != nil && c.Error != nil {
				c.Error <- err
				return c.handleLogin, nil
			}

			c.RespCreateSuccess <- actorEntity
		}

	default:
	}

	if c.Packet != nil {
		c.Packet <- p
	}

	return c.handleLogin, nil
}

// This method blocks in an infinite loop
// that will read packets off the protocol.Conn
// and send the relevant data out through a
// channel. This method is intended invoked as
// a go routine.
func (c *Conn) StartHandling() {
	var err error

	f := c.handleLogin
	for f != nil {
		f, err = f()
	}

	c.Error <- err
}
