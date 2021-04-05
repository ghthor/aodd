package prototest

import (
	"io"
	"sync"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/aodd/game/client"
	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/filu/rpg2d/coord"
	"github.com/ghthor/filu/rpg2d/entity"
	"github.com/ghthor/filu/rpg2d/quad/quadstate"
	"github.com/ghthor/filu/rpg2d/rpg2dtest"
	"github.com/ghthor/filu/rpg2d/worldstate"
	"github.com/ghthor/filu/rpg2d/worldterrain"

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
			exitWithError <- game.LoginAndConnectActor(loginConn,
				func(dsactor datastore.Actor, stateWriter game.InitialStateWriter) (game.ConnectedActor, entity.State) {
					actor := &mockActor{
						actor:       dsactor,
						stateWriter: stateWriter,
						entityState: game.ActorEntityState{
							Id:   nextId(),
							Name: dsactor.Name,
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

	var stopServerOnce sync.Once
	// Close down the io.Pipe which will bring down
	// the read loops on both the server and client.
	var stopServer func() = func() func() {
		return func() {
			stopServerOnce.Do(func() {
				for _, pw := range conn.pw {
					c.Assume(pw.Close(), IsNil)
				}

				for _, pr := range conn.pr {
					c.Assume(pr.Close(), IsNil)
				}

				c.Assume(<-serverExitError, Not(IsNil))
				actor, _ := ds.ActorExists("actor")
				c.Assume(actor.CanBeConnected(), IsTrue)
			})
		}
	}()

	withStopServer := func(f func()) func() {
		return func() { defer stopServer(); f() }
	}

	defer stopServer()

	loginConn := client.NewLoginConn(game.NewGobConn(conn.nextEndpoint()))
	c.Assume(conn.nextEndpoint(), IsNil)

	c.Specify("an actor conn", withStopServer(func() {
		c.Specify("can request to login", withStopServer(func() {
			c.Specify("and the request will succeed", withStopServer(func() {
				trip := loginConn.AttemptLogin("actor", "password")
				c.Assume(<-trip.Error, IsNil)
				loginResp := <-trip.Success
				c.Expect(loginResp.Name, Equals, "actor")
			}))

			c.Specify("and the request will fail", withStopServer(func() {
				c.Specify("if the actor is already connected", withStopServer(func() {
					dsactor, exists := ds.ActorExists("actor")
					c.Assume(exists, IsTrue)

					<-dsactor.IsConnected
					dsactor.IsConnected <- true

					trip := loginConn.AttemptLogin("actor", "password")
					c.Expect(<-trip.ActorAlreadyConnected, Equals, game.RespActorAlreadyConnected{
						"actor",
					})
					c.Expect(<-trip.Error, IsNil)

					<-dsactor.IsConnected
					dsactor.IsConnected <- false
				}))

				c.Specify("if the actor doesn't exist", withStopServer(func() {
					trip := loginConn.AttemptLogin("newActor", "password")
					c.Expect(<-trip.ActorDoesntExist, Equals, game.RespActorDoesntExist{
						"newActor",
						"password",
					})
					c.Expect(<-trip.Error, IsNil)
				}))

				c.Specify("if the password is incorrect", withStopServer(func() {
					trip := loginConn.AttemptLogin("actor", "wrongpassword")
					c.Expect(<-trip.AuthFailed, Equals, game.RespAuthFailed{
						"actor",
					})
					c.Expect(<-trip.Error, IsNil)
				}))
			}))
		}))

		c.Specify("can create a new actor", withStopServer(func() {
			c.Specify("and the request will succeed", withStopServer(func() {
				trip := loginConn.CreateActor("newActor", "password")
				c.Assume(<-trip.Error, IsNil)
				loginResp := <-trip.Success
				c.Expect(loginResp.Name, Equals, "newActor")
			}))

			c.Specify("and the request will fail", withStopServer(func() {
				c.Specify("if the actor already exists", withStopServer(func() {
					trip := loginConn.CreateActor("actor", "password")
					c.Expect(<-trip.ActorExists, Equals, game.RespActorExists{"actor"})
					c.Expect(<-trip.Error, IsNil)
				}))
			}))
		}))

		login := func() client.RespLoggedIn {
			trip := loginConn.AttemptLogin("actor", "password")
			loginResp := <-trip.Success
			c.Assume(<-trip.Error, IsNil)
			c.Assume(loginResp.Name, Equals, "actor")
			return loginResp
		}

		c.Specify("that is logged in", withStopServer(func() {
			loginResp := login()

			var actor *mockActor

			initialState := func() *worldstate.Snapshot {
				bounds := coord.Bounds{
					coord.Cell{-10, 10},
					coord.Cell{10, -10},
				}
				state := worldstate.NewSnapshot(2, bounds, 2)
				state.Entities.New = append(state.Entities.New, &quadstate.Entity{State: actor.entityState, Type: quadstate.TypeNew})
				terrainMap, err := worldterrain.NewMap(state.Bounds, string(worldterrain.TT_GRASS))
				c.Assume(err, IsNil)
				state.TerrainMap = &worldterrain.MapState{terrainMap}

				return state
			}

			// Must write the initial state first or this will block
			getResponse := func(trip client.ConnectRoundTrip) (*game.RespActorAlreadyConnected, client.RespConnected, error) {
				var actorAlreadyConnected game.RespActorAlreadyConnected
				var connectResp client.RespConnected
				var err error

				select {
				case actorAlreadyConnected = <-trip.ActorAlreadyConnected:
				case connectResp = <-trip.Connected:
				case err = <-trip.Error:
				}

				return &actorAlreadyConnected, connectResp, err
			}

			c.Specify("can be connected to the simulation", withStopServer(func() {
				trip := loginResp.ConnectActor(loginResp.Name)
				actor = <-connectedActor

				actor.stateWriter.WriteWorldState(initialState())
				_, connectResp, err := getResponse(trip)

				c.Assume(err, IsNil)
				c.Expect(connectResp.InitialState.Entity, Equals, actor.entityState)
				c.Expect(connectResp.InitialState.WorldState, rpg2dtest.SnapshotEquals, initialState())

				dsactor, exists := ds.ActorExists("actor")
				c.Assume(exists, IsTrue)
				c.Expect(dsactor.CanBeConnected(), IsFalse)
			}))

			c.Specify("cannot be connected if it is already connected", withStopServer(func() {
				// Open a second connection that will use the
				// same datastore as the existing connection.
				conn := newMockConn()
				var (
					serverExitError <-chan error
					exitWithError   chan<- error
				)

				func() {
					errorCh := make(chan error)
					exitWithError, serverExitError =
						errorCh, errorCh
				}()

				// Start the server and return the error
				// so we can sync before the function scope
				// is exitted.
				func() {
					loginConn := game.NewPreLoginConn(game.NewGobConn(conn.nextEndpoint()), ds)

					go func() {
						exitWithError <- game.LoginAndConnectActor(loginConn, nil)
					}()
				}()

				defer func() {
					for _, pw := range conn.pw {
						c.Assume(pw.Close(), IsNil)
					}

					for _, pr := range conn.pr {
						c.Assume(pr.Close(), IsNil)
					}
					c.Assume(<-serverExitError, Not(IsNil))
				}()

				loginConn := client.NewLoginConn(game.NewGobConn(conn.nextEndpoint()))
				c.Assume(conn.nextEndpoint(), IsNil)

				// Log the second connection in
				login := func() client.RespLoggedIn {
					trip := loginConn.AttemptLogin("actor", "password")
					loginResp := <-trip.Success
					c.Assume(<-trip.Error, IsNil)
					c.Assume(loginResp.Name, Equals, "actor")
					return loginResp
				}
				loggedInCantConnect := login()

				// Connect the first connection that logged in
				trip := loginResp.ConnectActor(loginResp.Name)
				actor = <-connectedActor

				actor.stateWriter.WriteWorldState(initialState())
				_, _, err := getResponse(trip)
				c.Assume(err, IsNil)

				trip = loggedInCantConnect.ConnectActor(loggedInCantConnect.Name)

				resp, _, err := getResponse(trip)
				c.Assume(err, IsNil)
				c.Expect(resp.Name, Equals, loginResp.Name)
			}))
		}))

		c.Specify("that is connected to the simulation", withStopServer(func() {
			loginResp := login()

			trip := loginResp.ConnectActor(loginResp.Name)
			actor := <-connectedActor

			worldBounds := coord.Bounds{
				coord.Cell{-10, 10},
				coord.Cell{10, -10},
			}
			initialQuad := func() quadstate.Quad {
				quad, err := quadstate.New(worldBounds, 10)
				c.Assume(err, IsNil)
				quad = quad.Insert(&quadstate.Entity{State: actor.entityState, Type: quadstate.TypeNew})
				return quad
			}
			initialTerrain := func() *worldterrain.MapState {
				terrainMap, err := worldterrain.NewMap(worldBounds, string(worldterrain.TT_GRASS))
				c.Assume(err, IsNil)
				return &worldterrain.MapState{terrainMap}
			}

			newSnapshot := func(bounds coord.Bounds) *worldstate.Snapshot {
				return worldstate.CullForInitialState(2, bounds, initialQuad(), initialTerrain(), 1)
			}

			diffWriter := actor.stateWriter.WriteWorldState(newSnapshot(coord.Bounds{
				coord.Cell{-2, 2},
				coord.Cell{2, -2},
			}))

			var err error
			var connectResp client.RespConnected

			select {
			case err = <-trip.Error:
			case connectResp = <-trip.Connected:
			}

			c.Assume(err, IsNil)

			c.Specify("will receive an initial world state", withStopServer(func() {
				c.Expect(connectResp.InitialState.Entity, Equals, actor.entityState)
				c.Expect(connectResp.InitialState.WorldState, rpg2dtest.SnapshotEquals,
					newSnapshot(coord.Bounds{
						coord.Cell{-2, 2},
						coord.Cell{2, -2},
					}),
				)

				c.Specify("followed by world state diffs", withStopServer(func() {
					diff := worldstate.NewUpdate(0)
					diff.FromSnapshot(newSnapshot(coord.Bounds{
						coord.Cell{-2, 2},
						coord.Cell{2, -2},
					}), newSnapshot(coord.Bounds{
						coord.Cell{-3, 3},
						coord.Cell{1, -1},
					}))

					go func() {
						diffWriter.WriteWorldStateDiff(diff)
					}()

					update, err := connectResp.NextUpdate()
					c.Assume(err, IsNil)
					c.Expect(update, rpg2dtest.StateEquals, diff)
				}))
			}))

			c.Specify("can submit a move request", withStopServer(func() {
				r := game.MoveRequest{
					MoveRequestType: game.MR_MOVE,
					Time:            2,
					Direction:       coord.North,
				}
				connectResp.InputConn.SendMoveRequest(r)
				c.Expect(<-actor.lastMoveRequest, Equals, r)
			}))

			c.Specify("can submit a use request", withStopServer(func() {
				r := game.UseRequest{
					UseRequestType: game.UR_USE,
					Time:           2,
					Skill:          "dion",
				}
				connectResp.InputConn.SendUseRequest(r)
				c.Expect(<-actor.lastUseRequest, Equals, r)
			}))

			c.Specify("can submit a chat request", withStopServer(func() {
				r := game.ChatRequest{
					ChatRequestType: game.CR_SAY,
					Time:            2,
					Msg:             "a chat msg",
				}
				connectResp.InputConn.SendChatRequest(r)
				c.Expect(<-actor.lastChatRequest, Equals, r)
			}))
		}))
	}))
}
