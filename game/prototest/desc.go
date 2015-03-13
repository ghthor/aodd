package prototest

import (
	"io"
	"sync"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/aodd/game/client"
	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/engine/rpg2d"
	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/rpg2d/entity"
	"github.com/ghthor/engine/rpg2d/rpg2dtest"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

type mockConn struct {
	pr [2]*io.PipeReader
	pw [2]*io.PipeWriter

	nextEndpoint func() io.ReadWriter
}

type mockReadWriter struct {
	io.Reader
	io.Writer
}

func newMockConn() *mockConn {
	c := &mockConn{}
	c.pr[0], c.pw[0] = io.Pipe()
	c.pr[1], c.pw[1] = io.Pipe()

	c.nextEndpoint = func() io.ReadWriter {
		c.nextEndpoint = func() io.ReadWriter {
			c.nextEndpoint = func() io.ReadWriter {
				return nil
			}

			return mockReadWriter{c.pr[0], c.pw[1]}
		}

		return mockReadWriter{c.pr[1], c.pw[0]}
	}
	return c
}

type mockActor struct {
	actor datastore.Actor

	stateWriter game.InitialStateWriter

	entityState game.ActorEntityState

	lastMoveRequest chan game.MoveRequest
	lastUseRequest  chan game.UseRequest
	lastChatRequest chan game.ChatRequest

	wasClosed bool
}

func (a *mockActor) SubmitMoveRequest(r game.MoveRequest) {
	a.lastMoveRequest <- r
}
func (a *mockActor) SubmitUseRequest(r game.UseRequest) {
	a.lastUseRequest <- r
}
func (a *mockActor) SubmitChatRequest(r game.ChatRequest) {
	a.lastChatRequest <- r
}

func (a mockActor) Close() { a.wasClosed = true }

func DescribeActorGobConn(c gospec.Context) {
	ds := datastore.NewMemDatastore()
	ds.AddActor("actor", "password")

	conn := newMockConn()

	nextId := entity.NewIdGenerator()

	var (
		actorConnected chan<- *mockActor
		connectedActor <-chan *mockActor

		serverExitError <-chan error
		exitWithError   chan<- error
	)

	func() {
		actorCh := make(chan *mockActor, 1)
		errorCh := make(chan error)

		actorConnected, connectedActor =
			actorCh, actorCh
		exitWithError, serverExitError =
			errorCh, errorCh
	}()

	// Start the server and return the error
	// so we can sync before the function scope
	// is exitted.
	func() {
		loginConn := game.NewPreLoginConn(game.NewGobConn(conn.nextEndpoint()), ds)

		go func() {
			exitWithError <- game.RunServer(loginConn,
				func(dsactor datastore.Actor, stateWriter game.InitialStateWriter) (game.InputReceiver, entity.State) {
					actor := &mockActor{
						actor:       dsactor,
						stateWriter: stateWriter,
						entityState: game.ActorEntityState{
							EntityId: nextId(),
							Name:     dsactor.Name,
						},

						lastMoveRequest: make(chan game.MoveRequest),
						lastUseRequest:  make(chan game.UseRequest),
						lastChatRequest: make(chan game.ChatRequest),
					}
					actorConnected <- actor
					return actor, actor.entityState
				})
		}()
	}()

	// Close down the io.Pipe which will bring down
	// the read loops on both the server and client.
	var stopServer func() = func() func() {
		var stopServer sync.Once
		return func() {
			stopServer.Do(func() {
				for _, pw := range conn.pw {
					c.Assume(pw.Close(), IsNil)
				}

				for _, pr := range conn.pr {
					c.Assume(pr.Close(), IsNil)
				}

				c.Assume(<-serverExitError, Equals, io.ErrClosedPipe)
			})
		}
	}()
	defer stopServer()

	loginConn := client.NewLoginConn(game.NewGobConn(conn.nextEndpoint()))
	c.Assume(conn.nextEndpoint(), IsNil)

	c.Specify("an actor conn", func() {
		c.Specify("can request to login", func() {
			c.Specify("and the request will succeed", func() {
				trip := loginConn.AttemptLogin("actor", "password")
				c.Assume(<-trip.Error, IsNil)
				loginResp := <-trip.Success
				c.Expect(loginResp.Name, Equals, "actor")
			})

			c.Specify("and the request will fail", func() {
				c.Specify("if the actor doesn't exist", func() {
					trip := loginConn.AttemptLogin("newActor", "password")
					c.Expect(<-trip.ActorDoesntExist, Equals, game.RespActorDoesntExist{
						"newActor",
						"password",
					})
					c.Expect(<-trip.Error, IsNil)
				})

				c.Specify("if the password is incorrect", func() {
					trip := loginConn.AttemptLogin("actor", "wrongpassword")
					c.Expect(<-trip.AuthFailed, Equals, game.RespAuthFailed{
						"actor",
					})
					c.Expect(<-trip.Error, IsNil)
				})
			})
		})

		c.Specify("can create a new actor", func() {
			c.Specify("and the request will succeed", func() {
				trip := loginConn.CreateActor("newActor", "password")
				c.Assume(<-trip.Error, IsNil)
				loginResp := <-trip.Success
				c.Expect(loginResp.Name, Equals, "newActor")
			})

			c.Specify("and the request will fail", func() {
				c.Specify("if the actor already exists", func() {
					trip := loginConn.CreateActor("actor", "password")
					c.Expect(<-trip.ActorExists, Equals, game.RespActorExists{"actor"})
					c.Expect(<-trip.Error, IsNil)
				})
			})
		})

		login := func() client.RespLoggedIn {
			trip := loginConn.AttemptLogin("actor", "password")
			loginResp := <-trip.Success
			c.Assume(<-trip.Error, IsNil)
			c.Assume(loginResp.Name, Equals, "actor")
			return loginResp
		}

		c.Specify("that is logged in", func() {
			loginResp := login()

			c.Specify("can be connected to the simulation", func() {
				trip := loginResp.ConnectActor(loginResp.Name)
				actor := <-connectedActor

				initialState := func() rpg2d.WorldState {
					state := rpg2d.WorldState{}
					state.Time = 2
					state.Bounds = coord.Bounds{
						coord.Cell{-10, 10},
						coord.Cell{10, -10},
					}
					state.Entities = append(state.Entities, actor.entityState)
					terrainMap, err := rpg2d.NewTerrainMap(state.Bounds, string(rpg2d.TT_GRASS))
					c.Assume(err, IsNil)
					state.TerrainMap = &rpg2d.TerrainMapState{terrainMap}

					return state
				}

				actor.stateWriter.WriteWorldState(initialState())

				var err error
				var connectResp client.RespConnected

				select {
				case err = <-trip.Error:
				case connectResp = <-trip.Connected:
				}

				c.Assume(err, IsNil)
				c.Expect(connectResp.InitialState.Entity, Equals, actor.entityState)
				c.Expect(connectResp.InitialState.WorldState, rpg2dtest.StateEquals, initialState())
			})
		})

		c.Specify("that is connected to the simulation", func() {
			loginResp := login()

			trip := loginResp.ConnectActor(loginResp.Name)
			actor := <-connectedActor

			initialState := func() rpg2d.WorldState {
				state := rpg2d.WorldState{}
				state.Time = 2
				state.Bounds = coord.Bounds{
					coord.Cell{-10, 10},
					coord.Cell{10, -10},
				}
				state.Entities = append(state.Entities, actor.entityState)
				terrainMap, err := rpg2d.NewTerrainMap(state.Bounds, string(rpg2d.TT_GRASS))
				c.Assume(err, IsNil)
				state.TerrainMap = &rpg2d.TerrainMapState{terrainMap}

				return state
			}

			actor.stateWriter.WriteWorldState(initialState())

			var err error
			var connectResp client.RespConnected

			select {
			case err = <-trip.Error:
			case connectResp = <-trip.Connected:
			}

			c.Assume(err, IsNil)

			c.Specify("will receive an initial world state", func() {
				c.Expect(connectResp.InitialState.Entity, Equals, actor.entityState)
				c.Expect(connectResp.InitialState.WorldState, rpg2dtest.StateEquals, initialState())
			})

			c.Specify("can submit a move request", func() {
				r := game.MoveRequest{
					MoveRequestType: game.MR_MOVE,
					Time:            2,
					Direction:       coord.North,
				}
				connectResp.InputConn.SendMoveRequest(r)
				c.Expect(<-actor.lastMoveRequest, Equals, r)
			})

			c.Specify("can submit a use request", func() {
				r := game.UseRequest{
					UseRequestType: game.UR_USE,
					Time:           2,
					Skill:          "dion",
				}
				connectResp.InputConn.SendUseRequest(r)
				c.Expect(<-actor.lastUseRequest, Equals, r)
			})

			c.Specify("can submit a chat request", func() {
				r := game.ChatRequest{
					ChatRequestType: game.CR_SAY,
					Time:            2,
					Msg:             "a chat msg",
				}
				connectResp.InputConn.SendChatRequest(r)
				c.Expect(<-actor.lastChatRequest, Equals, r)
			})
		})
	})
}
