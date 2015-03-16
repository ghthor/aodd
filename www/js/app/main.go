//go:generate stringer -type=event
//go:generate gopherjs build

// +build js
package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/aodd/game/client"
	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/sim/stime"

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

	// Come together in the same response from the server
	EV_RECV_INPUT_CONN
	EV_RECV_INITIAL_STATE

	EV_RECV_UPDATE

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

				loginConn := client.NewLoginConn(game.NewGobConn(ws))
				pub := eventPublisher{pub}

				// Emit a connected event and a object the
				// login form can use to send messages to the
				// server.
				pub.Emit(EV_CONNECTED, jsArray{newLoginConn(loginConn, pub)})
			}()
		},
	}

	for i := EV_ERROR; i < EV_SIZE; i++ {
		module[event(i).String()] = event(i).String()
	}

	coordModule := make(jsObject, int(coord.West)+1)

	for i := 0; i <= int(coord.West); i++ {
		coordModule[coord.Direction(i).String()] = coord.Direction(i)
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

	// require("github.com/ghthor/engine/rpg2d/coord")
	module["coord"] = coordModule
	// require("github.com/ghthor/aodd/game")
	module["game"] = gameModule

	return module
}

func newLoginConn(loginConn client.LoginConn, pub eventPublisher) jsObject {
	var conn sync.Mutex

	return jsObject{
		"attemptLogin": func(name, password string) {
			go func() {
				conn.Lock()
				defer conn.Unlock()

				trip := loginConn.AttemptLogin(name, password)

				select {
				case actorDoesntExist := <-trip.ActorDoesntExist:
					pub.Emit(EV_ACTOR_DOESNT_EXIST, jsArray{
						actorDoesntExist.Name,
						actorDoesntExist.Password,
					})

				case authFailed := <-trip.AuthFailed:
					js.Debugger()
					pub.Emit(EV_AUTH_FAILED, jsArray{authFailed.Name})

				case resp := <-trip.Success:
					pub.Emit(EV_LOGIN_SUCCESS, jsArray{resp.Name, newLoggedInConn(resp.Name, resp.LoggedInConn)})

				case err := <-trip.Error:
					pub.Emit(EV_ERROR, jsArray{jsObject{"error": err.Error()}})
				}
			}()
		},

		"createActor": func(name, password string) {
			go func() {
				conn.Lock()
				defer conn.Unlock()

				trip := loginConn.CreateActor(name, password)

				select {
				case actorExists := <-trip.ActorExists:
					pub.Emit(EV_ACTOR_EXISTS, jsArray{actorExists.Name})

				case resp := <-trip.Success:
					pub.Emit(EV_LOGIN_SUCCESS, jsArray{resp.Name, newLoggedInConn(resp.Name, resp.LoggedInConn)})

				case err := <-trip.Error:
					pub.Emit(EV_ERROR, jsArray{jsObject{"error": err.Error()}})
				}
			}()
		},
	}
}

func newLoggedInConn(name string, loggedInConn client.LoggedInConn) jsObject {
	return jsObject{
		"connectActor": func(pub *js.Object) {
			if pub.Get("emit").String() == undefined {
				log.Println("invalid publisher: missing emit() function")
				return
			}

			go func() {
				pub := eventPublisher{pub}

				trip := loggedInConn.ConnectActor(name)

				select {
				case resp := <-trip.Connected:
					pub.Emit(EV_RECV_INPUT_CONN, jsArray{newInputConn(resp.InputConn)})
					pub.Emit(EV_RECV_INITIAL_STATE, jsArray{
						resp.InitialState.Entity,
						resp.InitialState.WorldState,
					})

					for {
						update, err := resp.NextUpdate()
						if err != nil {
							pub.Emit(EV_ERROR, jsArray{jsObject{"error": err.Error()}})
							return
						}
						pub.Emit(EV_RECV_UPDATE, jsArray{update})
					}

				case err := <-trip.Error:
					pub.Emit(EV_ERROR, jsArray{jsObject{"error": err.Error()}})
				}
			}()
		},
	}
}

func newInputConn(conn client.InputConn) jsObject {
	return jsObject{
		"sendMoveRequest": func(typ game.MoveRequestType, t stime.Time, d coord.Direction) {
			go func() {
				conn.SendMoveRequest(game.MoveRequest{
					MoveRequestType: typ,
					Time:            t,
					Direction:       d,
				})
			}()
		},

		"sendUseRequest": func(typ game.UseRequestType, t stime.Time, skill string) {
			go func() {
				conn.SendUseRequest(game.UseRequest{
					UseRequestType: typ,
					Time:           t,
					Skill:          skill,
				})
			}()
		},

		"sendChatRequest": func(typ game.ChatRequestType, t stime.Time, msg string) {
			go func() {
				conn.SendChatRequest(game.ChatRequest{
					ChatRequestType: typ,
					Time:            t,
					Msg:             msg,
				})
			}()
		},
	}
}
