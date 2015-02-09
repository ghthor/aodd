package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"golang.org/x/net/websocket"

	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/engine/net/encoding"
	"github.com/ghthor/engine/net/protocol"
	"github.com/ghthor/engine/sim"
)

type LoginReq struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type packetHandler func(actorHandler) (actorHandler, error)

type actorHandler struct {
	protocol.Conn
	handlePacket packetHandler

	sim       sim.RunningSimulation
	datastore datastore.Datastore

	actor datastore.Actor
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
		c.SendMessage("authFailed", "invalid actor/password")
		return c, nil
	}

	c = c.loginActor(actor)

	log.Print("login success:", r.Name)
	c.SendMessage("loginSuccess", r.Name)
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

	c.SendMessage("createSuccess", r.Name)

	return c, nil
}

func (c actorHandler) loginActor(actor datastore.Actor) actorHandler {
	// Set the actor this connection is now associated with
	c.actor = actor
	// Mutate the packet handler into the next state
	c.handlePacket = (actorHandler).inputHandler

	// TODO hook the actor into the simulation
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
	case encoding.PT_JSON:
		switch packet.Msg {
		case "login", "create":
			goto alreadyLoggedIn
		}
	}

alreadyLoggedIn:
	c.SendMessage("alreadyLoggedIn", "an actor has already been logged into this connection")
	return c, nil
}

// Return the actor bound to the connection.
func (c actorHandler) Actor() datastore.Actor { return c.actor }

func newWebsocketActorHandler(sim sim.RunningSimulation, datastore datastore.Datastore) websocket.Handler {
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
