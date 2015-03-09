package prototest

import (
	"io"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/aodd/game/client"
	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/engine/rpg2d"

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

type mockSimulation struct{}

func (mockSimulation) ConnectActor(rpg2d.Actor) {}
func (mockSimulation) RemoveActor(rpg2d.Actor)  {}

func (mockSimulation) Halt() (rpg2d.HaltedSimulation, error) { return nil, nil }

func DescribeActorGobConn(c gospec.Context) {
	ds := datastore.NewMemDatastore()
	ds.AddActor("actor", "password")

	conn := newMockConn()

	server := game.NewActorGobConn(conn.nextEndpoint(), mockSimulation{}, ds)
	go func() {
		c.Assume(server.Run(), IsNil)
	}()

	client := client.NewConn(conn.nextEndpoint())
	c.Assume(conn.nextEndpoint(), IsNil)

	c.Specify("an actor conn", func() {
		c.Specify("can request to login", func() {
			c.Specify("and the request will succeed", func() {
				trip := client.AttemptLogin("actor", "password")
				c.Expect(<-trip.Success, Equals, game.ActorEntityState{})
				c.Expect(<-trip.Error, IsNil)
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
				c.Expect(<-trip.Success, Equals, game.ActorEntityState{})
				c.Expect(<-trip.Error, IsNil)
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
