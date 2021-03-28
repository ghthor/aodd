package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/aodd/game/client"
	"github.com/ghthor/aodd/game/client/canvas"
	"github.com/ghthor/filu/rpg2d/coord"
	"github.com/ghthor/filu/rpg2d/entity"
	"github.com/ghthor/filu/rpg2d/quad/quadstate"
	"github.com/ghthor/filu/rpg2d/worldstate"
	"github.com/ghthor/filu/rpg2d/worldterrain"
	"github.com/ghthor/filu/sim/stime"

	"syscall/js"

	"nhooyr.io/websocket"
)

func jsValueTypeMustBeString(v js.Value) string {
	if v.Type() != js.TypeString {
		panic(fmt.Sprintf("expected string type, got %v", v.Type()))
	}

	return v.String()
}

type EventPublisher interface {
	Emit(fmt.Stringer, jsArrayWrapped)
}

type eventPublisher struct {
	js.Value
}

type jsArrayWrapped struct {
	js.Value
}

func jsArray(elements ...interface{}) jsArrayWrapped {
	result := js.Global().Get("Array").New(len(elements))
	for i, e := range elements {
		result.SetIndex(i, e)
	}

	return jsArrayWrapped{result}
}

func newJSObject() js.Value {
	return js.Global().Get("Object").New()
}

func errorObj(err error) js.Value {
	result := newJSObject()
	result.Set("error", err.Error())
	return result
}

func (e eventPublisher) Emit(event fmt.Stringer, params jsArrayWrapped) {
	e.Call("emit", js.ValueOf(event.String()), params)
}

// Key used on the window object
// window.gopherjsApplication
const moduleKey = "gopherjsApplication"

const undefined = "undefined"

func main() {
	js.Global().Set(moduleKey, map[string]interface{}{
		"moduleKey":  moduleKey,
		"initialize": js.FuncOf(js_initialize),
	})

	// TODO Figure out what work I can do here instead of block forever
	<-make(chan struct{})
}

func js_initialize(this js.Value, args []js.Value) interface{} {
	return initialize(args[0])
}

// This function will be called once by requirejs
// from the shim configuration. The object that is
// returned here is what will be available as the
// require("app") module.
func initialize(settings js.Value) js.Value {
	module := newJSObject()
	module.Set("moduleKey", moduleKey)
	// Provide a function to dial the server.
	// pub should be an object that has been
	// extended with minpubsub.
	module.Set("dial", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		pub := args[0]
		func() {
			if pub.Get("emit").String() == undefined {
				log.Println("invalid publisher: missing emit() function")
				return
			}

			go func() {
				ctx := context.Background()
				wsUrl := settings.Get("websocketURL").String()

				ws, _, err := websocket.Dial(ctx, wsUrl, nil)
				if err != nil {
					log.Print(err)
					return
				}
				// TODO move this into a configuration option
				ws.SetReadLimit(32768 * 4)
				wsConn := websocket.NetConn(ctx, ws, websocket.MessageBinary)

				loginConn := client.NewLoginConn(game.NewGobConn(wsConn))
				pub := eventPublisher{pub}

				// Emit a connected event and a object the
				// login form can use to send messages to the
				// server.
				pub.Emit(EV_CONNECTED, jsArray(newLoginConn(loginConn, pub)))
			}()
		}()

		return nil
	}))

	for i := EV_ERROR; i < EV_SIZE; i++ {
		module.Set(event(i).String(), event(i).String())
	}

	coordModule := newJSObject()

	for i := 0; i <= int(coord.West); i++ {
		coordModule.Set(coord.Direction(i).String(), int(coord.Direction(i)))
	}

	gameModule := newJSObject()

	for i := game.MR_ERROR; i < game.MR_SIZE; i++ {
		gameModule.Set(game.MoveRequestType(i).String(), int(game.MoveRequestType(i)))
	}

	for i := game.UR_ERROR; i < game.UR_SIZE; i++ {
		gameModule.Set(game.UseRequestType(i).String(), int(game.UseRequestType(i)))
	}

	for i := game.CR_ERROR; i < game.CR_SIZE; i++ {
		gameModule.Set(game.ChatRequestType(i).String(), int(game.ChatRequestType(i)))
	}

	// require("github.com/ghthor/filu/rpg2d/coord")
	module.Set("coord", coordModule)
	// require("github.com/ghthor/aodd/game")
	module.Set("game", gameModule)

	return module
}

func newLoginConn(loginConn client.LoginConn, pub eventPublisher) (result js.Value) {
	var conn sync.Mutex

	result = newJSObject()
	result.Set("attemptLogin", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		name := jsValueTypeMustBeString(args[0])
		password := jsValueTypeMustBeString(args[1])

		func(name, password string) {
			go func() {
				conn.Lock()
				defer conn.Unlock()

				trip := loginConn.AttemptLogin(name, password)

				select {
				case actorConnected := <-trip.ActorAlreadyConnected:
					pub.Emit(EV_ACTOR_ALREADY_CONNECTED, jsArray(actorConnected.Name))

				case actorDoesntExist := <-trip.ActorDoesntExist:
					pub.Emit(EV_ACTOR_DOESNT_EXIST, jsArray(
						actorDoesntExist.Name,
						actorDoesntExist.Password,
					))

				case authFailed := <-trip.AuthFailed:
					pub.Emit(EV_AUTH_FAILED, jsArray(authFailed.Name))

				case resp := <-trip.Success:
					pub.Emit(EV_LOGIN_SUCCESS, jsArray(
						resp.Name,
						newLoggedInConn(resp.Name, resp.LoggedInConn),
					))

				case err := <-trip.Error:
					pub.Emit(EV_ERROR, jsArray(errorObj(err)))
				}
			}()
		}(name, password)

		return nil
	}))

	result.Set("createActor", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		name := jsValueTypeMustBeString(args[0])
		password := jsValueTypeMustBeString(args[1])
		func(name, password string) {
			go func() {
				conn.Lock()
				defer conn.Unlock()

				trip := loginConn.CreateActor(name, password)

				select {
				case actorExists := <-trip.ActorExists:
					pub.Emit(EV_ACTOR_EXISTS, jsArray(actorExists.Name))

				case resp := <-trip.Success:
					pub.Emit(EV_CREATE_SUCCESS, jsArray(
						resp.Name,
						newLoggedInConn(resp.Name, resp.LoggedInConn),
					))

				case err := <-trip.Error:
					pub.Emit(EV_ERROR, jsArray(errorObj(err)))
				}
			}()
		}(name, password)

		return nil
	}))

	return
}

type world struct {
	mu sync.RWMutex

	entity game.ActorEntityState

	initialSnapshot *worldstate.Snapshot
	lastUpdate      *worldstate.Update
	entities        map[entity.Id]*quadstate.Entity
}

func (w *world) now() stime.Time {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.lastUpdate == nil {
		if w.initialSnapshot == nil {
			return 0
		}

		return w.initialSnapshot.Time
	}

	return w.lastUpdate.Time
}

func (w *world) apply(snapshot *worldstate.Snapshot) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, e := range snapshot.Entities.New {
		// Update the actors entity
		if e.EntityId() == w.entity.EntityId() {
			w.entity = e.State.(game.ActorEntityState)
			break
		}

		w.entities[e.EntityId()] = e
	}

	for _, e := range snapshot.Entities.Changed {
		// Update the actors entity
		if e.EntityId() == w.entity.EntityId() {
			w.entity = e.State.(game.ActorEntityState)
			break
		}

		w.entities[e.EntityId()] = e
	}

	for _, e := range snapshot.Entities.Unchanged {
		// Update the actors entity
		if e.EntityId() == w.entity.EntityId() {
			w.entity = e.State.(game.ActorEntityState)
			break
		}

		w.entities[e.EntityId()] = e
	}

	for _, e := range snapshot.Entities.Removed {
		delete(w.entities, e.EntityId())
	}

	w.initialSnapshot = snapshot
}

func (w *world) update(update *worldstate.Update) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Update the state
	//w.state.Apply(diff)
	for _, e := range update.Entities {
		// Update the actors entity
		if e.EntityId() == w.entity.EntityId() {
			w.entity = e.State.(game.ActorEntityState)
			break
		}

		w.entities[e.EntityId()] = e
	}

	for _, e := range update.Removed {
		delete(w.entities, e.EntityId())
	}

	// TODO Fix the rendering of entities at edge of view bounds
	//      I might need to be running this even when there are no updates?
	// This is an attempt to fix that, but they are still building up
	// At the edges of the viewport.
	for id, e := range w.entities {
		if !update.Bounds.Contains(e.State.EntityCell()) {
			update.Removed = append(update.Removed, &quadstate.Entity{State: e.State, Type: quadstate.TypeRemoved})
			delete(w.entities, id)
		}
	}

	w.lastUpdate = update
}

func (w *world) actorEntityById(id entity.Id) (game.ActorEntityState, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	var err error

	for k, e := range w.entities {
		if k == id {
			switch e := e.State.(type) {
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

func (c terrainCanvas) Reset(slice worldterrain.MapStateSlice) {
	c.pub.Emit(EV_TERRAIN_RESET, jsArray(slice))
}

func (c terrainCanvas) Shift(dir canvas.TerrainShift, mags canvas.TerrainShiftMagnitudes) {
	for dir, mag := range mags {
		c.pub.Emit(EV_TERRAIN_CANVAS_SHIFT, jsArray(int(dir), mag))
	}
}

func (c terrainCanvas) DrawTile(ttype worldterrain.Type, cell coord.Cell) {
	c.pub.Emit(EV_TERRAIN_DRAW_TILE, jsArray(string(ttype), cell))
}

func newLoggedInConn(name string, loggedInConn client.LoggedInConn) (result js.Value) {
	result = newJSObject()
	result.Set("connectActor", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		pub := args[0]

		if pub.Get("emit").String() == undefined {
			panic("invalid publisher: missing emit() function")
		}

		func() {
			go func() {
				pub := eventPublisher{pub}

				trip := loggedInConn.ConnectActor(name)

				select {
				case resp := <-trip.Connected:
					world := world{
						entity:   resp.InitialState.Entity,
						entities: make(map[entity.Id]*quadstate.Entity, len(resp.InitialState.WorldState.Entities.Changed)),
					}
					world.apply(resp.InitialState.WorldState)

					log.Printf("%#v", resp.InitialState)
					log.Printf("%#v", resp.InitialState.WorldState)
					pub.Emit(EV_RECV_INPUT_CONN, jsArray(newInputConn(&world, resp.InputConn, pub)))
					pub.Emit(EV_RECV_INITIAL_STATE, jsArray(
						resp.InitialState.Entity,
						resp.InitialState.WorldState,
						resp.InitialState.WorldState.TerrainMap,
					))

					for {
						update, err := resp.NextUpdate()
						if err != nil {
							pub.Emit(EV_ERROR, jsArray(errorObj(err)))
							return
						}

						// TODO Fix unsafe concurrent access of world.state
						var bounds coord.Bounds
						if world.lastUpdate == nil {
							bounds = world.initialSnapshot.Bounds
						} else {
							bounds = world.lastUpdate.Bounds
						}

						err = canvas.ApplyTerrainDiff(terrainCanvas{pub}, bounds, update.TerrainMapSlices)
						if err != nil {
							pub.Emit(EV_ERROR, jsArray(errorObj(err)))
						}

						world.update(update)

						for _, e := range update.Entities {
							switch e := e.State.(type) {
							case game.SayEntityState:
								err := emitChatRecvEvent(&world, e, pub)
								if err != nil {
									pub.Emit(EV_ERROR, jsArray(errorObj(err)))
								}
							}
						}

						pub.Emit(EV_RECV_UPDATE, jsArray(update))
					}

				case resp := <-trip.ActorAlreadyConnected:
					pub.Emit(EV_ACTOR_ALREADY_CONNECTED, jsArray(resp.Name))

				case err := <-trip.Error:
					pub.Emit(EV_ERROR, jsArray(errorObj(err)))
				}
			}()
		}()

		return nil
	}))

	return result
}

func emitChatRecvEvent(world *world, say game.SayEntityState, pub EventPublisher) error {
	actor, err := world.actorEntityById(say.SaidBy)
	if err != nil {
		return err
	}

	pub.Emit(EV_RECV_CHAT_SAY, jsArray(
		int64(say.Id),
		actor.Name,
		say.Msg,
		int64(say.SaidAt),
	))
	return nil
}

func newInputConn(world *world, conn client.InputConn, pub EventPublisher) (result js.Value) {
	result = newJSObject()
	result.Set("sendMoveRequest", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		typ := game.MoveRequestType(args[0].Int())
		d := coord.Direction(args[1].Int())

		func(typ game.MoveRequestType, d coord.Direction) {
			go func() {
				conn.SendMoveRequest(game.MoveRequest{
					MoveRequestType: typ,
					Time:            world.now(),
					Direction:       d,
				})
			}()
		}(typ, d)

		return nil
	}))

	result.Set("sendUseRequest", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		typ := game.UseRequestType(args[0].Int())
		skill := args[1].String()

		func(typ game.UseRequestType, skill string) {
			go func() {
				conn.SendUseRequest(game.UseRequest{
					UseRequestType: typ,
					Time:           world.now(),
					Skill:          skill,
				})
			}()
		}(typ, skill)
		return nil
	}))

	result.Set("sendChatRequest", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		typ := game.ChatRequestType(args[0].Int())
		msg := args[1].String()

		func(typ game.ChatRequestType, msg string) {
			go func() {
				conn.SendChatRequest(game.ChatRequest{
					ChatRequestType: typ,
					Time:            world.now(),
					Msg:             msg,
				})
			}()

			pub.Emit(EV_SENT_CHAT_SAY, jsArray())
		}(typ, msg)
		return nil
	}))

	return result
}
