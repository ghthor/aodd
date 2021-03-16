package main

import (
	"log"
	"time"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/aodd/game/client"
	"github.com/ghthor/filu/rpg2d/coord"
	"golang.org/x/net/websocket"
)

func login(conn game.Conn, name, password string) client.RespLoggedIn {
	trip := client.NewLoginConn(conn).
		AttemptLogin(name, password)

	select {
	case actorConnected := <-trip.ActorAlreadyConnected:
		log.Fatal("actor already connected:", actorConnected)

	case actorDoesntExist := <-trip.ActorDoesntExist:
		return createActor(conn, actorDoesntExist.Name, actorDoesntExist.Password)

	case authFailed := <-trip.AuthFailed:
		log.Fatal("auth failure:", authFailed)

	case resp := <-trip.Success:
		return resp

	case err := <-trip.Error:
		log.Fatal("error:", err)
	}

	panic("login() not reached")
}

func createActor(conn game.Conn, name, password string) client.RespLoggedIn {
	trip := client.NewLoginConn(conn).
		CreateActor(name, password)

	select {
	case actorExists := <-trip.ActorExists:
		log.Fatal("actor exists:", actorExists)
	case resp := <-trip.Success:
		return resp
	case err := <-trip.Error:
		log.Fatal("error:", err)
	}

	panic("createActor() not reached")
}

func connectActor(conn client.RespLoggedIn) client.RespConnected {
	trip := conn.ConnectActor(conn.Name)

	select {
	case actorAlreadyConnected := <-trip.ActorAlreadyConnected:
		log.Fatal("actor already connected:", actorAlreadyConnected)
	case resp := <-trip.Connected:
		return resp
	case err := <-trip.Error:
		log.Fatal("error:", err)
	}

	panic("connectActor() not reached")
}

func main() {
	origin := "http://localhost"
	url := "ws://localhost:45001/actor/socket/gob"

	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	conn := game.NewGobConn(ws)
	loggedIn := login(conn, "test1", "test")
	actor := connectActor(loggedIn)
	for {
		actor.SendMoveRequest(game.MoveRequest{
			game.MR_MOVE,
			0,
			coord.N,
		})

		time.Sleep(1000)
	}

	// TODO: Monitor world state changes
	// TODO: Monitor actor state
	// TODO: Implement a behavior to follow the closest actor entity
}
