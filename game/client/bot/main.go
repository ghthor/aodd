package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/ghthor/aodd/game"
	"github.com/ghthor/aodd/game/client"
	"github.com/ghthor/filu/rpg2d/coord"
	"golang.org/x/net/websocket"
)

const (
	origin = "http://localhost"
	url    = "ws://localhost:45001/actor/socket/gob"
)

const MaxRandDir = int(coord.West + 1)

func RandDir() coord.Direction {
	return coord.Direction(rand.Intn(MaxRandDir))
}

func loginActor(conn game.Conn, name, password string) client.RespLoggedIn {
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

func connectBot(name, password string) Bot {
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	conn := game.NewGobConn(ws)
	loggedIn := loginActor(conn, name, password)
	resp := connectActor(loggedIn)

	return Bot{
		name,
		resp,
	}
}

type Bot struct {
	Name string
	client.RespConnected
}

func (b Bot) startRandomMove(ctx context.Context) {
	go func() {
		next := time.Tick(2 * time.Second)
		for {
			dir := RandDir()
			b.SendMoveRequest(game.MoveRequest{game.MR_MOVE, 0, dir})

			select {
			case <-ctx.Done():
				return
			case <-next:
				continue
			}
		}
	}()
}

func main() {
	// names := []string{
	// 	"maxbotter",
	// 	"arlanna",
	// 	"zyperis",
	// 	"stefie",
	// 	"permafrost",
	// 	"dhalsim",
	// 	"gerald",
	// 	"river",
	// 	"storm",
	// 	"summer",
	// 	"hawk",
	// 	"rogue",
	// }

	password := "hellohitacopie"

	nameCount := 100
	nameMap := make(map[string]int, nameCount)

	for i := 0; i < nameCount; i++ {
		name := petname.Generate(2, " ")
		nameMap[name] = i
	}

	names := make([]string, 0, len(nameMap))
	for name, _ := range nameMap {
		names = append(names, name)
	}

	ctx := context.Background()

	bots := make([]Bot, 0, len(names))

	for i, name := range names {
		b := connectBot(name, password)
		bots = append(bots, b)

		log.Printf("%v: %s starting movement", i, b.Name)
		b.startRandomMove(ctx)
	}

	c := make(chan struct{})
	<-c

	// TODO: Monitor world state changes
	// TODO: Monitor actor state
	// TODO: Implement a behavior to follow the closest actor entity
}
