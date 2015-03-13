package game

// Used to determine the next type that's in the
// buffer so we can decode it into a real value.
// We'll decode an encoded type and switch on its
// value so we'll have the correct value to decode
// into.
type EncodedType int

//go:generate stringer -type=EncodedType
const (
	ET_ERROR EncodedType = iota
	ET_DISCONNECT

	ET_REQ_LOGIN
	ET_REQ_CREATE

	ET_RESP_AUTH_FAILED
	ET_RESP_ACTOR_EXISTS
	ET_RESP_ACTOR_DOESNT_EXIST

	ET_RESP_LOGIN_SUCCESS
	ET_RESP_CREATE_SUCCESS

	ET_REQ_CONNECT
	ET_CONNECTED

	ET_WORLD_STATE
	ET_WORLD_STATE_DIFF

	ET_REQ_MOVE
	ET_REQ_USE
	ET_REQ_CHAT
)

type Conn interface {
	EncodeAndSend(EncodedType, interface{}) error
	ReadNextType() (EncodedType, error)
	Decode(interface{}) error
}
