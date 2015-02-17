package game

import (
	"fmt"
	"net"
	"net/http"

	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/engine/net/encoding"
	"github.com/ghthor/engine/net/protocol"
	"github.com/ghthor/engine/rpg2d"
	"golang.org/x/net/websocket"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

var nextPort = 45456

// Abstracts the internals of creating 2 Websocket endpoints
func twoWebsockets() (*websocket.Conn, *websocket.Conn, chan<- bool, <-chan bool, error) {
	testServerAddr := fmt.Sprintf("localhost:%v", nextPort)
	nextPort++

	listener, err := net.Listen("tcp", testServerAddr)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	ws2Chan := make(chan *websocket.Conn)
	mux := http.NewServeMux()
	mux.Handle("/", websocket.Handler(func(ws *websocket.Conn) {
		ws2Chan <- ws
		// Wait till the server has been signaled to close
		<-ws2Chan
	}))

	server := &http.Server{Handler: mux}

	// Start a Server that signal's when its finished listening
	serverClosed := make(chan bool)
	go func() {
		server.Serve(listener)
		// Signal that server has shutdown
		serverClosed <- true
	}()

	// Get the Second Websocket
	ws1, err := websocket.Dial("ws://"+testServerAddr+"/", "", "http://localhost")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// Get the First Websocket
	ws2 := <-ws2Chan

	closeServer := make(chan bool)
	go func() {
		// Wait for signal to shutdown
		<-closeServer

		// Close the websocket in the http.Handler
		ws2Chan <- ws2

		// Close the listen server
		listener.Close()
	}()

	return ws1, ws2, closeServer, serverClosed, nil
}

type mockSimulation struct{}

func (mockSimulation) ConnectActor(rpg2d.Actor)              {}
func (mockSimulation) RemoveActor(rpg2d.Actor)               {}
func (mockSimulation) Halt() (rpg2d.HaltedSimulation, error) { return nil, nil }

func DescribeActorConn(c gospec.Context) {
	// Setup
	ws, wsServer, closeServer, serverClosed, err := twoWebsockets()

	c.Assume(err, IsNil)
	c.Assume(ws, Not(IsNil))
	c.Assume(wsServer, Not(IsNil))

	defer func() {
		select {
		case closeServer <- true:
			<-serverClosed
		case <-serverClosed:
		}
	}()

	ds := datastore.NewMemDatastore()
	ds.AddActor("actor", "password")

	conn := &actorHandler{
		Conn: protocol.NewWebsocketConn(wsServer),

		sim:       mockSimulation{},
		datastore: ds,
	}

	client := protocol.NewWebsocketConn(ws)

	var packet encoding.Packet

	getResp := func(client protocol.Conn) encoding.Packet {
		packet, err := client.Read()
		c.Assume(err, IsNil)
		return packet
	}

	// Convenience function for sending requests
	login := func() {
		client.SendJson("login", LoginReq{"actor", "password"})
		packet := getResp(client)
		c.Assume(packet.Type, Equals, encoding.PT_JSON)
		c.Assume(packet.Msg, Equals, "loginSuccess")
	}

	createActor := func() {
		client.SendJson("create", LoginReq{"newActor", "password"})
		packet := getResp(client)
		c.Assume(packet.Type, Equals, encoding.PT_JSON)
		c.Assume(packet.Msg, Equals, "createSuccess")
	}

	c.Specify("packet processing should terminate", func() {
		c.Specify("when a client disconnects", func() {
			go func() {
				c.Assume(client.Send(encoding.Packet{}), IsNil)
				c.Assume(ws.Close(), IsNil)
			}()

			err := conn.run()
			c.Assume(err, Not(IsNil))

			c.Expect(err, Equals, ErrWebsocketClientDisconnected)
		})

		c.Specify("when a client disconnects after the actor logs in", func() {

			go func() {
				login()
				c.Assume(client.Send(encoding.Packet{}), IsNil)
				c.Assume(ws.Close(), IsNil)
			}()

			err := conn.run()
			c.Assume(err, Not(IsNil))

			c.Expect(err, Equals, ErrWebsocketClientDisconnected)
		})

		c.Specify("when the connection is lost", func() {
			go func() {
				c.Assume(ws.Close(), IsNil)
			}()

			err := conn.run()
			c.Assume(err, Not(IsNil))

			_, isAnDisconnectionError := err.(*protocol.DisconnectionError)
			c.Expect(isAnDisconnectionError, IsTrue)
		})

		c.Specify("when the connection is lost after the actor logs in", func() {

			go func() {
				login()
				c.Assume(ws.Close(), IsNil)
			}()

			err := conn.run()
			c.Assume(err, Not(IsNil))

			_, isAnDisconnectionError := err.(*protocol.DisconnectionError)
			c.Expect(isAnDisconnectionError, IsTrue)
		})
	})

	c.Specify("when the client sends a request", func() {
		handlerHasTerminated := make(chan struct{})

		// Run the packet handler
		go func() {
			c.Assume(conn.run(), Equals, ErrWebsocketClientDisconnected)

			// Signal the handler has terminated
			handlerHasTerminated <- struct{}{}
		}()

		// Close the packet handler and clean up
		defer func() {
			c.Assume(client.Send(encoding.Packet{}), IsNil)
			c.Assume(ws.Close(), IsNil)

			// Wait for the handler to exit the packetHandler loop
			<-handlerHasTerminated
		}()

		c.Specify("to login", func() {
			c.Specify("the request should fail", func() {
				c.Specify("if the actor doesn't exist", func() {
					c.Assume(client.SendJson("login", LoginReq{"notAnActor", "anything"}), IsNil)
					packet := getResp(client)

					c.Expect(packet.Type, Equals, encoding.PT_JSON)
					c.Expect(packet.Msg, Equals, "actorDoesntExist")
					c.Expect(packet.Payload, Equals, `{"name":"notAnActor","password":"anything"}`)

					c.Expect(conn.Actor(), Equals, datastore.Actor{})
				})

				c.Specify("if the password is incorrect", func() {
					c.Assume(client.SendJson("login", LoginReq{"actor", "wrongpassword"}), IsNil)
					packet := getResp(client)

					c.Expect(packet.Type, Equals, encoding.PT_MESSAGE)
					c.Expect(packet.Msg, Equals, "authFailed")

					c.Expect(conn.Actor(), Equals, datastore.Actor{})
				})

				c.Specify("if the client is already logged in", func() {
					login()
					c.Assume(conn.Actor().Name, Equals, "actor")

					c.Assume(client.SendJson("login", LoginReq{"actor", "password"}), IsNil)
					packet := getResp(client)

					c.Expect(packet.Type, Equals, encoding.PT_MESSAGE)
					c.Expect(packet.Msg, Equals, "alreadyLoggedIn")
				})

				c.Specify("if the login request payload json", func() {
					c.Specify("is invalid", func() {
						c.Assume(client.Send(encoding.Packet{
							Type:    encoding.PT_JSON,
							Msg:     "login",
							Payload: "{malformed json",
						}), IsNil)
						packet := getResp(client)

						c.Expect(packet.Type, Equals, encoding.PT_ERROR)
						c.Expect(packet.Msg, Equals, "invalidLoginRequest")
						c.Expect(packet.Payload, Equals, "{malformed json")
					})
				})
			})

			c.Specify("the request should succeed", func() {
				c.Assume(client.SendJson("login", LoginReq{"actor", "password"}), IsNil)
				packet = getResp(client)

				c.Expect(packet.Type, Equals, encoding.PT_JSON)
				c.Expect(packet.Msg, Equals, "loginSuccess")

				c.Expect(conn.Actor().Name, Equals, "actor")
			})
		})

		c.Specify("to create a new actor", func() {
			c.Specify("the request should fail", func() {
				c.Specify("if the actor already exists", func() {
					c.Assume(client.SendJson("create", LoginReq{"actor", "password"}), IsNil)
					packet = getResp(client)

					c.Expect(packet.Type, Equals, encoding.PT_MESSAGE)
					c.Expect(packet.Msg, Equals, "actorAlreadyExists")
				})

				c.Specify("if the an actor has been logged in", func() {
					login()
					c.Assume(client.SendJson("create", LoginReq{"newActor", "password"}), IsNil)
					packet = getResp(client)

					c.Expect(packet.Type, Equals, encoding.PT_MESSAGE)
					c.Expect(packet.Msg, Equals, "alreadyLoggedIn")
				})

				c.Specify("if an actor has already been created", func() {
					createActor()
					c.Assume(client.SendJson("create", LoginReq{"newActor", "password"}), IsNil)
					packet = getResp(client)

					c.Expect(packet.Type, Equals, encoding.PT_MESSAGE)
					c.Expect(packet.Msg, Equals, "alreadyLoggedIn")
				})

				c.Specify("if the create request payload json", func() {
					c.Specify("is invalid", func() {
						c.Assume(client.Send(encoding.Packet{
							Type:    encoding.PT_JSON,
							Msg:     "create",
							Payload: "{malformed json",
						}), IsNil)
						packet := getResp(client)

						c.Expect(packet.Type, Equals, encoding.PT_ERROR)
						c.Expect(packet.Msg, Equals, "invalidCreateRequest")
						c.Expect(packet.Payload, Equals, "{malformed json")
					})
				})
			})

			c.Specify("the request should succeed", func() {
				c.Assume(client.SendJson("create", LoginReq{"newActor", "password"}), IsNil)
				packet = getResp(client)

				c.Expect(packet.Type, Equals, encoding.PT_JSON)
				c.Expect(packet.Msg, Equals, "createSuccess")
				c.Expect(conn.Actor().Name, Equals, "newActor")

				_, exists := ds.ActorExists("newActor")
				c.Expect(exists, IsTrue)
			})
		})
	})
}
