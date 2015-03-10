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

	stateWriter game.StateWriter

	entityState game.ActorEntityState

	lastMoveRequest game.MoveRequest
	lastUseRequest  game.UseRequest
	lastChatRequest game.ChatRequest

	wasClosed bool
}

func (a *mockActor) SubmitMoveRequest(r game.MoveRequest) {
	a.lastMoveRequest = r
}
func (a *mockActor) SubmitUseRequest(r game.UseRequest) {
	a.lastUseRequest = r
}
func (a *mockActor) SubmitChatRequest(r game.ChatRequest) {
	a.lastChatRequest = r
}

func (a mockActor) Close() { a.wasClosed = true }

func DescribeActorGobConn(c gospec.Context) {
	ds := datastore.NewMemDatastore()
	ds.AddActor("actor", "password")

	conn := newMockConn()

	nextId := entity.NewIdGenerator()
	var connectedActor *mockActor

	server := game.NewActorGobConn(conn.nextEndpoint(), ds,
		func(dsactor datastore.Actor, stateWriter game.StateWriter) (game.InputReceiver, entity.State) {
			actor := &mockActor{
				actor:       dsactor,
				stateWriter: stateWriter,
				entityState: game.ActorEntityState{
					EntityId: nextId(),
					Name:     dsactor.Name,
				},
			}
			connectedActor = actor
			return actor, actor.entityState
		},
	)

	serverClosed := make(chan error)

	// Start the server and return the error
	// so we can sync before the function scope
	// is exitted.
	go func() {
		serverClosed <- server.Run()
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

				c.Assume(<-serverClosed, Equals, io.ErrClosedPipe)
			})
		}
	}()
	defer stopServer()

	loginConn := client.NewLoginConn(conn.nextEndpoint())
	c.Assume(conn.nextEndpoint(), IsNil)

	c.Specify("an actor conn", func() {
		c.Specify("can request to login", func() {
			c.Specify("and the request will succeed", func() {
				trip := loginConn.AttemptLogin("actor", "password")
				c.Assume(<-trip.Error, IsNil)
				loginResp := <-trip.Success
				c.Expect(loginResp, Not(Equals), client.RespLoggedIn{})

				c.Specify("and an actor will be added to the simulation", func() {
					stopServer()
					initialState := <-loginResp.InitialState
					c.Assume(<-loginResp.Error, Equals, io.ErrClosedPipe)

					c.Expect(initialState.Entity, Not(Equals), game.ActorEntityState{})
					c.Expect(initialState.Entity, Equals, connectedActor.entityState)
				})
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
				c.Expect(loginResp, Not(Equals), client.RespLoggedIn{})

				c.Specify("and an actor will be added to the simulation", func() {
					stopServer()
					initialState := <-loginResp.InitialState
					c.Assume(<-loginResp.Error, Equals, io.ErrClosedPipe)

					c.Expect(initialState.Entity, Not(Equals), game.ActorEntityState{})
					c.Expect(initialState.Entity, Equals, connectedActor.entityState)
				})
			})

			c.Specify("and the request will fail", func() {
				c.Specify("if the actor already exists", func() {
					trip := loginConn.CreateActor("actor", "password")
					c.Expect(<-trip.ActorExists, Equals, game.RespActorExists{"actor"})
					c.Expect(<-trip.Error, IsNil)
				})
			})
		})

		c.Specify("that is logged in", func() {
			trip := loginConn.AttemptLogin("actor", "password")
			loginResp := <-trip.Success
			c.Assume(<-trip.Error, IsNil)
			c.Assume(loginResp, Not(Equals), client.RespLoggedIn{})
			c.Assume(connectedActor, Not(IsNil))

			initialState := func() rpg2d.WorldState {
				state := rpg2d.WorldState{}
				state.Time = 2
				state.Bounds = coord.Bounds{
					coord.Cell{-2, 2},
					coord.Cell{1, -1},
				}
				terrainMap, err := rpg2d.NewTerrainMap(state.Bounds, string(rpg2d.TT_GRASS))
				c.Assume(err, IsNil)
				state.TerrainMap = &rpg2d.TerrainMapState{terrainMap}

				return state
			}

			c.Specify("will recieve an initial world state", func() {
				state := initialState()
				connectedActor.stateWriter.WriteWorldState(state)
				initialState := <-loginResp.InitialState
				c.Expect(initialState.Entity, Equals, connectedActor.entityState)
				c.Expect(initialState.WorldState, rpg2dtest.StateEquals, state)

				stopServer()
				c.Expect(<-initialState.Error, Equals, io.ErrClosedPipe)

				c.Specify("followed by world state diffs", func() {
				})
			})

			c.Specify("can submit a move request", func() {
				r := game.MoveRequest{
					MoveRequestType: game.MR_MOVE,
					Time:            2,
					Direction:       coord.North}
				loginResp.Conn.SendMoveRequest(r)
				c.Expect(connectedActor.lastMoveRequest, Equals, r)
			})

			c.Specify("can submit a use request", func() {
				r := game.UseRequest{
					UseRequestType: game.UR_USE,
					Time:           2,
					Skill:          "dion",
				}
				loginResp.Conn.SendUseRequest(r)
				c.Expect(connectedActor.lastUseRequest, Equals, r)
			})

			c.Specify("can submit a chat request", func() {
				r := game.ChatRequest{
					ChatRequestType: game.CR_SAY,
					Time:            2,
					Msg:             "a chat msg",
				}
				loginResp.Conn.SendChatRequest(r)
				c.Expect(connectedActor.lastChatRequest, Equals, r)
			})
		})
	})
}
