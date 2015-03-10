package prototest

import (
	"io"
	"sync"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/aodd/game/client"
	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/engine/rpg2d"
	"github.com/ghthor/engine/rpg2d/entity"

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
	sync.Mutex
	actors map[rpg2d.ActorId]rpg2d.Actor
}

func (s *mockSimulation) ConnectActor(actor rpg2d.Actor) {
	s.Lock()
	defer s.Unlock()

	if s.actors == nil {
		s.actors = make(map[rpg2d.ActorId]rpg2d.Actor, 1)
	}

	s.actors[actor.Id()] = actor
}

func (s mockSimulation) RemoveActor(actor rpg2d.Actor) {
	s.Lock()
	defer s.Unlock()

	delete(s.actors, actor.Id())
}

func (s mockSimulation) connectedActors() (actors []rpg2d.Actor) {
	s.Lock()
	defer s.Unlock()

	actors = make([]rpg2d.Actor, 0, len(s.actors))
	for _, a := range s.actors {
		actors = append(actors, a)
	}

	return actors
}

func (mockSimulation) Halt() (rpg2d.HaltedSimulation, error) { return nil, nil }

func DescribeActorGobConn(c gospec.Context) {
	ds := datastore.NewMemDatastore()
	ds.AddActor("actor", "password")

	conn := newMockConn()

	sim := &mockSimulation{}
	server := game.NewActorGobConn(conn.nextEndpoint(), sim, ds, entity.NewIdGenerator())
	go func() {
		c.Assume(server.Run(), IsNil)
	}()

	client := client.NewLoginConn(conn.nextEndpoint())
	c.Assume(conn.nextEndpoint(), IsNil)

	c.Specify("an actor conn", func() {
		c.Specify("can request to login", func() {
			c.Specify("and the request will succeed", func() {
				trip := client.AttemptLogin("actor", "password")
				c.Expect(<-trip.Error, IsNil)

				c.Specify("and an actor will be added to the simulation", func() {
					c.Expect(<-trip.Success, Equals, sim.connectedActors()[0].Entity().ToState())
				})
			})

			c.Specify("and the request will fail", func() {
				c.Specify("if the actor doesn't exist", func() {
					trip := client.AttemptLogin("newActor", "password")
					c.Expect(<-trip.ActorDoesntExist, Equals, game.RespActorDoesntExist{
						"newActor",
						"password",
					})
					c.Expect(<-trip.Error, IsNil)
				})

				c.Specify("if the password is incorrect", func() {
					trip := client.AttemptLogin("actor", "wrongpassword")
					c.Expect(<-trip.AuthFailed, Equals, game.RespAuthFailed{
						"actor",
					})
					c.Expect(<-trip.Error, IsNil)
				})
			})
		})

		c.Specify("can create a new actor", func() {
			c.Specify("and the request will succeed", func() {
				trip := client.CreateActor("newActor", "password")
				c.Expect(<-trip.Error, IsNil)

				c.Specify("and an actor will be added to the simulation", func() {
					c.Expect(<-trip.Success, Equals, sim.connectedActors()[0].Entity().ToState())
				})
			})

			c.Specify("and the request will fail", func() {
				c.Specify("unless the actor already exists", func() {
					trip := client.CreateActor("actor", "password")
					c.Expect(<-trip.ActorExists, Equals, game.RespActorExists{"actor"})
					c.Expect(<-trip.Error, IsNil)
				})
			})
		})
	})
}
