package game

import (
	"bufio"
	"encoding/gob"
	"io"

	"github.com/ghthor/aodd/game/datastore"
	"github.com/ghthor/engine/rpg2d"
)

// Used to determine the next type that's in the
// buffer so we can decode it into a real value.
// We'll decode an encoded type and switch on its
// value so we'll have the correct value to decode
// into.
type EncodedType int

const (
	ET_ERROR EncodedType = iota

	ET_REQ_LOGIN
	ET_REQ_CREATE

	ET_RESP_AUTH_FAILED
	ET_RESP_ACTOR_EXISTS
	ET_RESP_ACTOR_DOESNT_EXIST

	ET_RESP_LOGIN_SUCCESS
	ET_RESP_CREATE_SUCCESS

	ET_WORLD_STATE
	ET_WORLD_STATE_DIFF
)

type ReqLogin struct{ Name, Password string }
type ReqCreate struct{ Name, Password string }

type RespAuthFailed struct{ Name string }
type RespActorExists struct{ Name string }
type RespActorDoesntExist struct{ Name, Password string }


func init() {
	// Pre login Request/Response types
	gob.Register(ReqLogin{})
	gob.Register(ReqCreate{})

	gob.Register(RespAuthFailed{})
	gob.Register(RespActorExists{})
	gob.Register(RespActorDoesntExist{})

	// ActorEntityState used for login/create success
	gob.Register(ActorEntityState{})

	// Engine types
	gob.Register(rpg2d.WorldState{})
	gob.Register(rpg2d.WorldStateDiff{})
	gob.Register(rpg2d.TerrainMapState{})

	// Other entity states
	gob.Register(SayEntityState{})
	gob.Register(AssailEntityState{})
}

type GobConn interface {
	EncodeAndSend(EncodedType, interface{}) error
	ReadNextType() (EncodedType, error)
	Decode(interface{}) error
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

func NewGobConn(rw io.ReadWriter) GobConn {
	wbuf := bufio.NewWriter(rw)
	enc := gob.NewEncoder(wbuf)
	dec := gob.NewDecoder(rw)

	return gobConn{
		enc:  enc,
		wbuf: wbuf,

		Decoder: dec,
	}
}

type serverConn struct {
	GobConn

	sim       rpg2d.RunningSimulation
	datastore datastore.Datastore

	actor *actor
}
