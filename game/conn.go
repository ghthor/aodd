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

type packetHandler func(actorHandler) (actorHandler, error)

type actorHandler struct {
	protocol.Conn
	handlePacket packetHandler

	sim       rpg2d.RunningSimulation
	datastore datastore.Datastore

	actor *actor
}

// Starts the packet handler loop.
// This function is blocking.
func (c actorHandler) run() (err error) {
	for {
		c, err = c.handlePacket(c)
		if err != nil {
			break
		}
	}
	return
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
func (c actorHandler) loginHandler() (actorHandler, error) {
	packet, err := c.Read()
	if err != nil {
		return c, err
	}

	if packet.Type == encoding.PT_DISCONNECT {
		return c, ErrWebsocketClientDisconnected
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
	c.SendMessage("notLoggedIn", "")
	return c, nil
}

// A login request is a event that can modify the
// state of the packet handler. If the login is
// successful the packet handler will transition
// to the input handler..
func (c actorHandler) respondToLoginReq(p encoding.Packet) (actorHandler, error) {
	r := LoginReq{}

	err := json.Unmarshal([]byte(p.Payload), &r)
	if err != nil {
		return c, errors.New(fmt.Sprint("error parsing login request:", err))
	}

	actor, exists := c.datastore.ActorExists(r.Name)
	if !exists {
		log.Printf("login failed: actor %s doesn't exist", r.Name)
		c.SendJson("actorDoesntExist", r)
		return c, nil
	}

	if !actor.Authenticate(r.Name, r.Password) {
		log.Printf("login failed: password for %s was incorrect", r.Name)
		c.SendMessage("authFailed", r.Name)
		return c, nil
	}

	c = c.loginActor(actor)

	log.Print("login success: ", r.Name)
	c.SendJson("loginSuccess", actor)
	return c, nil
}

// A create request is an event that can modify th
// state of the packet handler. If the create is
// successful the packet handler will transition
// in the input handler.
func (c actorHandler) respondToCreateReq(p encoding.Packet) (actorHandler, error) {
	r := LoginReq{}

	err := json.Unmarshal([]byte(p.Payload), &r)
	if err != nil {
		// TODO determine if this an error that should terminate the connection
		return c, errors.New(fmt.Sprint("error parsing login request:", err))
	}

	_, exists := c.datastore.ActorExists(r.Name)
	if exists {
		log.Printf("create failed: actor %s already exists", r.Name)
		c.SendMessage("actorAlreadyExists", "actor already exists")
		return c, nil
	}

	actor, err := c.datastore.AddActor(r.Name, r.Password)
	if err != nil {
		// TODO Instead of terminating the connection here
		// we should retry contacting the database or something
		return c, err
	}

	c = c.loginActor(actor)

	log.Print("created actor: ", actor.Name)

	c.SendJson("createSuccess", actor)

	return c, nil
}

// Creates a new actor struct using a datastore.Actor struct.
// Adds this new actor into the simulation.
func (c actorHandler) loginActor(dsactor datastore.Actor) actorHandler {
	// Set the actor this connection is now associated with
	// Mutate the packet handler into the next state
	c.handlePacket = (actorHandler).inputHandler

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
	return c
}

// An implementation of packetHandler which will
// process input requests and prepare them
// for consumption by the input phase.
func (c actorHandler) inputHandler() (actorHandler, error) {
	packet, err := c.Read()
	if err != nil {
		return c, err
	}

	if packet.Type == encoding.PT_DISCONNECT {
		return c, ErrWebsocketClientDisconnected
	}

	switch packet.Type {
	case encoding.PT_MESSAGE:
		if strings.Contains(packet.Msg, "move") {
			err := c.actor.SubmitCmd(packet.Msg, packet.Payload)
			if err != nil {
				c.SendError("invalidActorCommand", err.Error())
			}
			return c, nil
		}
	default:
	}

	c.SendMessage("alreadyLoggedIn", "an actor has already been logged into this connection")
	return c, nil
}

// Return the actor bound to the connection.
func (c actorHandler) Actor() datastore.Actor {
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

func newWebsocketActorHandler(sim rpg2d.RunningSimulation, datastore datastore.Datastore) websocket.Handler {
	return func(ws *websocket.Conn) {
		err := actorHandler{
			Conn:         protocol.NewWebsocketConn(ws),
			handlePacket: (actorHandler).loginHandler,

			sim:       sim,
			datastore: datastore,
		}.run()

		// TODO Maybe send a http response if there is an error
		if err != nil {
			log.Printf("disconnected: %v", err)
		}
	}
}
