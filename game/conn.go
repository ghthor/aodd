package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"golang.org/x/net/websocket"

	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/engine/net/encoding"
	"github.com/ghthor/engine/net/protocol"
	"github.com/ghthor/engine/rpg2d"
)

type LoginReq struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type packetHandler func(*conn) (packetHandler, error)

type conn struct {
	protocol.Conn

	sim       rpg2d.RunningSimulation
	datastore datastore.Datastore

	actor *actor
}

// Starts the packet handling state machine loop.
// This function is blocking.
func (c *conn) startPacketHandler() (err error) {
	handlePacket := loginHandler

	for err == nil {
		handlePacket, err = handlePacket(c)
	}

	if c.actor != nil {
		c.sim.RemoveActor(c.actor)
	}
	return err
}

var ErrWebsocketClientDisconnected = errors.New("websocket client disconnected")

type ErrUnexpectedPacket struct {
	Handler packetHandler
	Packet  encoding.Packet
}

func (e ErrUnexpectedPacket) String() string {
	return fmt.Sprint("unexpected packet {%v} in %v", e.Packet, e.Handler)
}

func (e ErrUnexpectedPacket) Error() string {
	return e.String()
}

// An implementation of packetHandler which
// will handle an actor logging in.
func loginHandler(c *conn) (packetHandler, error) {
	packet, err := c.Read()
	if err != nil {
		return nil, err
	}

	if packet.Type == encoding.PT_DISCONNECT {
		return nil, ErrWebsocketClientDisconnected
	}

	switch packet.Type {
	case encoding.PT_JSON:
		switch packet.Msg {
		case "login":
			return c.respondToLoginReq(packet)
		case "create":
			return c.respondToCreateReq(packet)
		default:
		}
	default:
	}

	// TODO Improve this message with how to login
	serr := c.SendMessage("notLoggedIn", "")
	if serr != nil {
		return nil, serr
	}

	return loginHandler, nil
}

// An implementation of packetHandler which will
// process input requests and prepare them
// for consumption by the input phase.
func inputHandler(c *conn) (packetHandler, error) {
	packet, err := c.Read()
	if err != nil {
		return nil, err
	}

	if packet.Type == encoding.PT_DISCONNECT {
		return nil, ErrWebsocketClientDisconnected
	}

	switch packet.Type {
	case encoding.PT_MESSAGE:
		if strings.Contains(packet.Msg, "move") {
			err := c.actor.SubmitCmd(packet.Msg, packet.Payload)
			if err != nil {
				serr := c.SendError("invalidActorCommand", err.Error())
				if serr != nil {
					return nil, serr
				}
			}
			return inputHandler, nil
		}
	default:
	}

	serr := c.SendMessage(
		"alreadyLoggedIn",
		"an actor has already been logged into this connection",
	)
	if serr != nil {
		return nil, serr
	}

	return inputHandler, nil
}

// A login request is a event that can modify the
// state of the packet handler. If the login is
// successful the packet handler will transition
// to the input handler..
func (c *conn) respondToLoginReq(p encoding.Packet) (packetHandler, error) {
	r := LoginReq{}

	err := json.Unmarshal([]byte(p.Payload), &r)
	if err != nil {
		serr := c.SendError("invalidLoginRequest", p.Payload)
		if serr != nil {
			return nil, serr
		}

		return loginHandler, nil
	}

	actor, exists := c.datastore.ActorExists(r.Name)
	if !exists {
		serr := c.SendJson("actorDoesntExist", r)
		if serr != nil {
			return nil, serr
		}

		return loginHandler, nil
	}

	if !actor.Authenticate(r.Name, r.Password) {
		serr := c.SendMessage("authFailed", r.Name)
		if serr != nil {
			return nil, serr
		}

		return loginHandler, nil
	}

	c.loginActor(actor)

	serr := c.SendJson("loginSuccess", c.actor.ToState())
	if serr != nil {
		return nil, serr
	}

	return inputHandler, nil
}

// A create request is an event that can modify th
// state of the packet handler. If the create is
// successful the packet handler will transition
// in the input handler.
func (c *conn) respondToCreateReq(p encoding.Packet) (packetHandler, error) {
	r := LoginReq{}

	err := json.Unmarshal([]byte(p.Payload), &r)
	if err != nil {
		serr := c.SendError("invalidCreateRequest", p.Payload)
		if serr != nil {
			return nil, serr
		}

		return loginHandler, nil
	}

	_, exists := c.datastore.ActorExists(r.Name)
	if exists {
		serr := c.SendMessage("actorAlreadyExists", "actor already exists")
		if serr != nil {
			return nil, serr
		}

		return loginHandler, nil
	}

	actor, err := c.datastore.AddActor(r.Name, r.Password)
	if err != nil {
		// TODO Instead of terminating the connection here
		// we should retry contacting the database or something
		return loginHandler, err
	}

	c.loginActor(actor)

	serr := c.SendJson("createSuccess", c.actor.ToState())
	if serr != nil {
		return nil, serr
	}

	return inputHandler, nil
}

// Creates a new actor struct using a datastore.Actor struct.
// Adds this new actor into the simulation.
func (c *conn) loginActor(dsactor datastore.Actor) {
	// Create an actorEntity for this object
	c.actor = &actor{
		actorEntity{
			id: dsactor.Id,

			name: dsactor.Name,

			cell:   dsactor.Loc,
			facing: dsactor.Facing,
		},

		newActorConn(c),

		actorCmdRequest{},
	}

	c.sim.ConnectActor(c.actor)
}

// Return the actor bound to the connection.
func (c conn) Actor() datastore.Actor {
	if c.actor == nil {
		return datastore.Actor{}
	}

	return datastore.Actor{
		Id: c.actor.id,

		Name: c.actor.name,

		Loc:    c.actor.cell,
		Facing: c.actor.facing,
	}
}

func newWebsocketHandler(sim rpg2d.RunningSimulation, datastore datastore.Datastore) websocket.Handler {
	return func(ws *websocket.Conn) {
		c := conn{
			Conn: protocol.NewWebsocketConn(ws),

			sim:       sim,
			datastore: datastore,
		}

		// Blocks until the connection is disconnected
		err := c.startPacketHandler()

		if err != nil {
			log.Printf("packet handler terminated: %v", err)
		}
	}
}
