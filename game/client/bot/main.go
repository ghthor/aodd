package main

import (
	"context"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/ghthor/aodd/game"
	"github.com/ghthor/aodd/game/client"
	"github.com/ghthor/filu/rpg2d/coord"
	"nhooyr.io/websocket"
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
	ctx := context.Background()
	ws, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	// TODO move this into a configuration option
	ws.SetReadLimit(32768 * 4)
	wsConn := websocket.NetConn(ctx, ws, websocket.MessageBinary)

	conn := game.NewGobConn(wsConn)
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
	stopCh := make(chan struct{}, 1)
	startCh := make(chan struct{}, 1)
	go func() {
		for {
			diff, err := b.NextUpdate()
			if err != nil {
				log.Println(b.Name, err)
			}

			for _, e := range diff.Entities {
				switch e := e.State.(type) {
				case game.SayEntityState:
					switch e.Msg {
					case "stop":
						stopCh <- struct{}{}
						break
					case "start":
						startCh <- struct{}{}
						break
					default:
					}
				}
			}
		}
	}()

	go func() {
		next := time.Tick(2 * time.Second)

		running := true
		curDir := coord.Direction(0)

		for {
			if running {
				dir := RandDir()
				curDir = dir
				b.SendMoveRequest(game.MoveRequest{game.MR_MOVE, 0, dir})
			} else {
				b.SendMoveRequest(game.MoveRequest{game.MR_MOVE_CANCEL, 0, curDir})
			}

			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				if running {
					b.SendChatRequest(game.ChatRequest{game.CR_SAY, 0, "stop"})
				}
				running = false
			case <-startCh:
				if !running {
					b.SendChatRequest(game.ChatRequest{game.CR_SAY, 0, "start"})
				}
				running = true
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

	botCount := flag.Int("n", 200, "number of bots to spawn")
	flag.Parse()

	password := "hellohitacopie"

	nameMap := make(map[string]int, *botCount)

	for i := 0; i < *botCount; i++ {
	generate:
		name := petname.Generate(2, " ")

		if _, exists := nameMap[name]; exists {
			goto generate
		}

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
