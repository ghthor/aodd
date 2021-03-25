package game

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"

	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/filu/rpg2d"
	"github.com/ghthor/filu/rpg2d/entity"
	"golang.org/x/net/websocket"
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

	// ActorEntityState used for response to connect
	gob.Register(ActorEntityState{})

	// Engine types
	gob.Register(rpg2d.WorldState{})
	gob.Register(rpg2d.WorldStateDiff{})
	gob.Register(rpg2d.TerrainMapState{})
	gob.Register(entity.RemovedState{})

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
	actorConnector ActorConnector) websocket.Handler {
	return func(ws *websocket.Conn) {
		ws.PayloadType = websocket.BinaryFrame

		c := NewPreLoginConn(NewGobConn(ws), ds)

		// Blocks until the connection has disconnected
		err := LoginAndConnectActor(c, actorConnector)

		if err != nil {
			log.Printf("packet handler terminated: %v", err)
		}
	}
}
