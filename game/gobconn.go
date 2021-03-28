package game

import (
	"bufio"
	"context"
	"encoding/gob"
	"io"
	"log"
	"net/http"

	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/filu/rpg2d/entity"
	"github.com/ghthor/filu/rpg2d/quad/quadstate"
	"github.com/ghthor/filu/rpg2d/worldstate"
	"github.com/ghthor/filu/rpg2d/worldterrain"
	"nhooyr.io/websocket"
)

type ReqLogin struct{ Name, Password string }
type ReqCreate struct{ Name, Password string }

type RespActorAlreadyConnected struct{ Name string }
type RespAuthFailed struct{ Name string }
type RespActorExists struct{ Name string }
type RespActorDoesntExist struct{ Name, Password string }

type RespLoginSuccess struct{ Name string }
type RespCreateSuccess struct{ Name string }

type ReqConnect struct{ Name string }

const RespDisconnect = "disconnected"

func init() {
	// Pre login Request/Response types
	gob.Register(ReqLogin{})
	gob.Register(ReqCreate{})

	gob.Register(RespActorAlreadyConnected{})
	gob.Register(RespAuthFailed{})
	gob.Register(RespActorExists{})
	gob.Register(RespActorDoesntExist{})

	gob.Register(RespLoginSuccess{})
	gob.Register(RespCreateSuccess{})

	// Engine types
	gob.Register(&worldstate.Snapshot{})
	gob.Register(&worldstate.Update{})
	gob.Register(worldterrain.MapState{})
	gob.Register(entity.RemovedState{})
	gob.Register(quadstate.Entity{})
	gob.Register([]*quadstate.Entity{})

	// ActorEntityState used for response to connect
	gob.Register(ActorEntityState{})

	// Other entity states
	gob.Register(SayEntityState{})
	gob.Register(AssailEntityState{})
	gob.Register(WallEntityState{})

	// Cmd Requests. They have no responses.
	gob.Register(MoveRequest{})
	gob.Register(UseRequest{})
	gob.Register(ChatRequest{})
}

type gobConn struct {
	enc  *gob.Encoder
	wbuf *bufio.Writer

	*gob.Decoder
}

func (c gobConn) EncodeAndSend(t EncodedType, ev interface{}) error {
	err := c.enc.Encode(t)
	if err != nil {
		return err
	}

	err = c.enc.Encode(ev)
	if err != nil {
		return err
	}

	return c.wbuf.Flush()
}

func (c gobConn) ReadNextType() (t EncodedType, err error) {
	err = c.Decoder.Decode(&t)
	return
}

func NewGobConn(rw io.ReadWriter) Conn {
	wbuf := bufio.NewWriter(rw)
	enc := gob.NewEncoder(wbuf)
	dec := gob.NewDecoder(rw)

	return gobConn{
		enc:  enc,
		wbuf: wbuf,

		Decoder: dec,
	}
}

func newGobWebsocketHandler(
	ds datastore.Datastore,
	actorConnector ActorConnector) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns:  []string{"localhost"},
			CompressionMode: websocket.CompressionDisabled,
		})
		if err != nil {
			log.Println(err)
			return
		}
		defer ws.Close(websocket.StatusInternalError, "its all coming down")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		conn := websocket.NetConn(ctx, ws, websocket.MessageBinary)
		c := NewPreLoginConn(NewGobConn(conn), ds)

		// Blocks until the connection has disconnected
		err = LoginAndConnectActor(c, actorConnector)
		if err != nil {
			log.Printf("packet handler terminated: %v", err)
		}
	})
}
