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
	"github.com/ghthor/aodd/game/client/canvas"
	"github.com/ghthor/engine/rpg2d"
	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/rpg2d/entity"
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

	EV_RECV_CHAT_SAY
	EV_SENT_CHAT_SAY

	EV_TERRAIN_RESET
	EV_TERRAIN_CANVAS_SHIFT
	EV_TERRAIN_DRAW_TILE

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

type world struct {
	mu sync.RWMutex

	entity game.ActorEntityState
	state  rpg2d.WorldState
}

func (w *world) now() stime.Time {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.state.Time
}

func (w *world) update(diff rpg2d.WorldStateDiff) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Update the state
	w.state.Apply(diff)

	// Update the entity
	for _, e := range w.state.Entities {
		if e.EntityId() == w.entity.EntityId() {
			w.entity = e.(game.ActorEntityState)
			break
		}
	}
}

func (w *world) actorEntityById(id entity.Id) (game.ActorEntityState, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	var err error

	for _, e := range w.state.Entities {
		if e.EntityId() == id {
			switch e := e.(type) {
			case game.ActorEntityState:
				return e, nil

			default:
				err = fmt.Errorf("expected e.(game.ActorEntityState), got e.(%T)", e)
			}

			break
		}
	}

	return game.ActorEntityState{}, err
}

type terrainCanvas struct {
	pub EventPublisher
}

func (c terrainCanvas) Reset(slice rpg2d.TerrainMapStateSlice) {
	c.pub.Emit(EV_TERRAIN_RESET, jsArray{slice})
}

func (c terrainCanvas) Shift(dir canvas.TerrainShift, mags canvas.TerrainShiftMagnitudes) {
	for dir, mag := range mags {
		c.pub.Emit(EV_TERRAIN_CANVAS_SHIFT, jsArray{dir, mag})
	}
}

func (c terrainCanvas) DrawTile(ttype rpg2d.TerrainType, cell coord.Cell) {
	c.pub.Emit(EV_TERRAIN_DRAW_TILE, jsArray{ttype, cell})
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
					world := world{
						entity: resp.InitialState.Entity,
						state:  resp.InitialState.WorldState,
					}

					pub.Emit(EV_RECV_INPUT_CONN, jsArray{newInputConn(&world, resp.InputConn, pub)})
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

						world.update(update)

						for _, e := range update.Entities {
							switch e := e.(type) {
							case game.SayEntityState:
								err := emitChatRecvEvent(&world, e, pub)
								if err != nil {
									pub.Emit(EV_ERROR, jsArray{jsObject{"error": err.Error()}})
								}
							}
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

func emitChatRecvEvent(world *world, say game.SayEntityState, pub EventPublisher) error {
	actor, err := world.actorEntityById(say.SaidBy)
	if err != nil {
		return err
	}

	pub.Emit(EV_RECV_CHAT_SAY, jsArray{say.Id, actor.Name, say.Msg, say.SaidAt})
	return nil
}

func newInputConn(world *world, conn client.InputConn, pub EventPublisher) jsObject {
	return jsObject{
		"sendMoveRequest": func(typ game.MoveRequestType, d coord.Direction) {
			go func() {
				conn.SendMoveRequest(game.MoveRequest{
					MoveRequestType: typ,
					Time:            world.now(),
					Direction:       d,
				})
			}()
		},

		"sendUseRequest": func(typ game.UseRequestType, skill string) {
			go func() {
				conn.SendUseRequest(game.UseRequest{
					UseRequestType: typ,
					Time:           world.now(),
					Skill:          skill,
				})
			}()
		},

		"sendChatRequest": func(typ game.ChatRequestType, msg string) {
			go func() {
				conn.SendChatRequest(game.ChatRequest{
					ChatRequestType: typ,
					Time:            world.now(),
					Msg:             msg,
				})
			}()

			pub.Emit(EV_SENT_CHAT_SAY, nil)
		},
	}
}
