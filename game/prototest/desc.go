package prototest

import (
	"io"
	"log"
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

type mockSimulation struct {
	*game.ActorIndexLocker

	actorsThatConnected chan rpg2d.Actor
}

func (s *mockSimulation) ConnectActor(actor rpg2d.Actor) {
	s.actorsThatConnected <- actor
}

func (s *mockSimulation) RemoveActor(actor rpg2d.Actor) {}

func (s *mockSimulation) connectedActors() (actors []rpg2d.Actor) {
	defer s.ActorIndexLocker.RUnlock()
	index := s.ActorIndexLocker.RLock()

	for _, a := range index {
		actors = append(actors, a)
	}

	return actors
}

func (s *mockSimulation) sendState(state rpg2d.WorldState) {
	defer s.ActorIndexLocker.RUnlock()
	index := s.ActorIndexLocker.RLock()

	for _, a := range index {
		a.WriteState(state)
	}
}

func (s *mockSimulation) toState() (state rpg2d.WorldState) {
	defer s.ActorIndexLocker.RUnlock()
	index := s.ActorIndexLocker.RLock()

	state.Bounds = coord.Bounds{
		coord.Cell{-128, 128},
		coord.Cell{127, -127},
	}

	for _, a := range index {
		state.Entities = append(state.Entities, a.Entity().ToState())
	}

	terrainMap, err := rpg2d.NewTerrainMap(state.Bounds, string(rpg2d.TT_GRASS))
	if err != nil {
		log.Fatal(err)
	}

	state.TerrainMap = &rpg2d.TerrainMapState{terrainMap}

	return
}

func (mockSimulation) Halt() (rpg2d.HaltedSimulation, error) { return nil, nil }

func DescribeActorGobConn(c gospec.Context) {
	ds := datastore.NewMemDatastore()
	ds.AddActor("actor", "password")

	actorIndex := game.NewActorIndexLocker(make(game.ActorIndex))
	mockSim := &mockSimulation{
		ActorIndexLocker:    actorIndex,
		actorsThatConnected: make(chan rpg2d.Actor, 100),
	}

	gameSim := game.NewSimulation(actorIndex, mockSim)
	// Defer removing any actors that haven't
	// been removed yet. This ensures that all
	// IO muxers have been halted by stopIO()
	var haltGameSim func() = func() func() {
		var haltGameSim sync.Once
		return func() {
			haltGameSim.Do(func() {
				actors := mockSim.connectedActors()
				for _, a := range actors {
					gameSim.RemoveActor(a)
				}

				// TODO maybe the cleanup above should be moved
				//      into the game.simulation composed type.
				_, err := gameSim.Halt()
				c.Assume(err, IsNil)
			})
		}
	}()
	defer haltGameSim()

	conn := newMockConn()
	server := game.NewActorGobConn(conn.nextEndpoint(), gameSim, ds, entity.NewIdGenerator())

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
					c.Expect(initialState.Entity, Equals, (<-mockSim.actorsThatConnected).Entity().ToState())
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
					c.Expect(initialState.Entity, Equals, (<-mockSim.actorsThatConnected).Entity().ToState())
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

			c.Specify("will recieve an initial world state", func() {
				state := mockSim.toState()
				mockSim.sendState(state)

				initialState := <-loginResp.InitialState
				c.Expect(initialState.Entity, Equals, (<-mockSim.actorsThatConnected).Entity().ToState())
				c.Expect(initialState.WorldState, rpg2dtest.StateEquals, state.Cull(game.ActorCullBounds(initialState.Entity.Cell)))

				stopServer()
				c.Expect(<-initialState.Error, Equals, io.ErrClosedPipe)

				c.Specify("followed by world state diffs", func() {
				})
			})
		})
	})
}
