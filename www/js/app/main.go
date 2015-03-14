//go:generate stringer -type=event
//go:generate gopherjs build

// +build js
package main

import (
	"fmt"
	"log"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/aodd/game/client"

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

	EV_ACTOR_DOESNT_EXIST
	EV_ACTOR_EXISTS
	EV_AUTH_FAILED

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

				// Create a concurrent safe client connection
				takeConn, freeConn := func() (<-chan client.LoginConn, chan<- client.LoginConn) {
					connLock := make(chan client.LoginConn, 1)
					return connLock, connLock
				}()
				freeConn <- client.NewLoginConn(game.NewGobConn(ws))

				// Emit a connected event and a object the
				// login form can use to send messages to the
				// server.
				eventPublisher{pub}.Emit(EV_CONNECTED, jsArray{jsObject{
					"attemptLogin": func(name, password string) {
						go func() {
							conn := <-takeConn
							defer func() { freeConn <- conn }()

							pub := eventPublisher{pub}
							trip := conn.AttemptLogin(name, password)

							select {
							case actorDoesntExist := <-trip.ActorDoesntExist:
								pub.Emit(EV_ACTOR_DOESNT_EXIST, jsArray{
									actorDoesntExist.Name,
									actorDoesntExist.Password,
								})

							case authFailed := <-trip.AuthFailed:
								js.Debugger()
								pub.Emit(EV_AUTH_FAILED, jsArray{authFailed.Name})

							case actor := <-trip.Success:
								pub.Emit(EV_LOGIN_SUCCESS, jsArray{actor})

							case err := <-trip.Error:
								pub.Emit(EV_ERROR, jsArray{jsObject{"error": err}})
							}
						}()
					},

					"createActor": func(name, password string) {
						go func() {
							conn := <-takeConn
							defer func() { freeConn <- conn }()

							pub := eventPublisher{pub}
							trip := conn.CreateActor(name, password)

							select {
							case actorExists := <-trip.ActorExists:
								pub.Emit(EV_ACTOR_EXISTS, jsArray{actorExists.Name})

							case actor := <-trip.Success:
								pub.Emit(EV_CREATE_SUCCESS, jsArray{actor})

							case err := <-trip.Error:
								pub.Emit(EV_ERROR, jsArray{jsObject{"error": err}})
							}
						}()
					},
				}})
			}()
		},
	}

	for i := EV_ERROR; i < EV_SIZE; i++ {
		module[event(i).String()] = event(i).String()
	}

	gameModule := make(jsObject, int(game.MR_SIZE)+int(game.UR_SIZE)+int(game.CR_SIZE))

	for i := game.MR_ERROR; i < game.MR_SIZE; i++ {
		gameModule[game.MoveRequestType(i).String()] = game.MoveRequestType(i)
	}

	for i := game.UR_ERROR; i < game.UR_SIZE; i++ {
		gameModule[game.UseRequestType(i).String()] = game.UseRequestType(i)
	}

	for i := game.CR_ERROR; i < game.CR_SIZE; i++ {
		gameModule[game.ChatRequestType(i).String()] = game.ChatRequestType(i)
	}

	// require("github.com/ghthor/aodd/game")
	module["game"] = gameModule

	return module
}
