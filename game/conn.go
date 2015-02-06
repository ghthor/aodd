package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"golang.org/x/net/websocket"

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

	sim sim.RunningSimulation
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

// A implementation of packetHandler that will
// handle an actor logging in.
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
		default:
			goto notLoggedIn
		}
	case encoding.PT_MESSAGE:
		goto notLoggedIn

	default:
		return c, ErrUnexpectedPacket{
			Handler: (actorHandler).loginHandler,
			Packet:  packet,
		}
	}

	return c, nil

notLoggedIn:
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

	// TODO Check the datastore using the login information
	// TODO Create an actor and input into the simulation

	log.Print("actor logged in:", r.Name)

	c.SendMessage("loginSuccess", r.Name)
	return c, nil
}

func newWebsocketActorHandler(sim sim.RunningSimulation) websocket.Handler {
	return func(ws *websocket.Conn) {
		err := actorHandler{
			Conn:         protocol.NewWebsocketConn(ws),
			handlePacket: (actorHandler).loginHandler,

			sim: sim,
		}.run()

		// TODO Maybe send a http response if there is an error
		if err != nil {
			log.Printf("disconnected: %v", err)
		}
	}
}
