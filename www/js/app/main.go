//go:generate stringer -type=event
//go:generate gopherjs build

// +build js
package main

import (
	"fmt"
	"log"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/aodd/game/client"
	"github.com/ghthor/engine/net/encoding"

	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/websocket"
)

type jsObject map[string]interface{}
type jsArray []interface{}

type EventPublisher interface {
	Emit(fmt.Stringer, jsArray)
}

type eventPublisher struct {
	*js.Object
}

func (e eventPublisher) Emit(event fmt.Stringer, params jsArray) {
	e.Call("emit", event.String(), params)
}

type event int

const (
	EV_ERROR event = iota
	EV_CONNECTED

	EV_AUTH_FAILED
	EV_ACTOR_DOESNT_EXIST

	EV_LOGIN_SUCCESS
	EV_CREATE_SUCCESS

	EV_PACKET
	EV_SIZE
)

// Key used on the window object
// window.gopherjsApplication
const moduleKey = "gopherjsApplication"

const undefined = "undefined"

func main() {
	js.Global.Set(moduleKey, jsObject{
		"moduleKey":  moduleKey,
		"initialize": initialize,
	})
}

// This function will be called once by requirejs
// from the shim configuration. The object that is
// returned here is what will be available as the
// require("app") module.
func initialize(settings *js.Object) jsObject {
	module := jsObject{
		"moduleKey": moduleKey,

		// Provide a function to dial the server.
		// pub should be an object that has been
		// extended with minpubsub.
		"dial": func(pub *js.Object) {
			if pub.Get("emit").String() == undefined {
				log.Println("invalid publisher: missing emit() function")
				return
			}

			go func() {
				wsUrl := settings.Get("websocketURL").String()

				ws, err := websocket.Dial(wsUrl)
				if err != nil {
					log.Fatal(err)
				}

				conn := client.NewConn(ws)

				// Channel endpoints we'll used to recieve
				// information from the server and emit events
				// in the JS universe.
				var evAuthFailed <-chan client.RespAuthFailed
				var evActorDoesntExist <-chan client.RespActorDoesntExist

				var evLoginSuccess <-chan game.ActorEntity
				var evCreateSuccess <-chan game.ActorEntity

				var evHandledPacket <-chan encoding.Packet
				var evError <-chan error

				func() {
					authFailedCh := make(chan client.RespAuthFailed)
					actorDoesntExistCh := make(chan client.RespActorDoesntExist)

					logginSuccessCh := make(chan game.ActorEntity)
					createSuccessCh := make(chan game.ActorEntity)

					packetCh := make(chan encoding.Packet)
					errCh := make(chan error)

					// Link send and recv comms together
					evAuthFailed, conn.RespAuthFailed = authFailedCh, authFailedCh
					evActorDoesntExist, conn.RespActorDoesntExist = actorDoesntExistCh, actorDoesntExistCh

					evLoginSuccess, conn.RespLoginSuccess = logginSuccessCh, logginSuccessCh
					evCreateSuccess, conn.RespCreateSuccess = createSuccessCh, createSuccessCh

					evHandledPacket, conn.Packet = packetCh, packetCh
					evError, conn.Error = errCh, errCh
				}()

				// Start the packet handlering concurrently
				go conn.PacketHandler()

				// Emit a connected event and a object the
				// login form can use to send messages to the
				// server.
				eventPublisher{pub}.Emit(EV_CONNECTED, jsArray{jsObject{
					"attemptLogin": conn.AttemptLogin,
					"createActor":  conn.CreateActor,
				}})

				// Recv packets and emit events
				for {
					pub := eventPublisher{pub}
					select {
					case actor := <-evAuthFailed:
						pub.Emit(EV_AUTH_FAILED, jsArray{actor.Name})

					case actor := <-evActorDoesntExist:
						pub.Emit(EV_ACTOR_DOESNT_EXIST, jsArray{actor.Name, actor.Password})

					case entity := <-evLoginSuccess:
						pub.Emit(EV_LOGIN_SUCCESS, jsArray{entity})

					case entity := <-evCreateSuccess:
						pub.Emit(EV_CREATE_SUCCESS, jsArray{entity})

					case packet := <-evHandledPacket:
						pub.Emit(EV_PACKET, jsArray{packet})

					case err := <-evError:
						pub.Emit(EV_ERROR, jsArray{jsObject{"error": err}})
					}
				}
			}()
		},
	}

	for i := EV_ERROR; i < EV_SIZE; i++ {
		module[event(i).String()] = event(i).String()
	}

	return module
}
