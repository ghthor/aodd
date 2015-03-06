package client

import (
	"io"
	"log"

	"github.com/ghthor/engine/net/protocol"
)

type Conn struct {
	protocol.Conn
}

func NewConn(with io.ReadWriteCloser) Conn {
	return Conn{protocol.NewConn(with)}
}

func (c Conn) AttemptLogin(name, password string) {
	log.Printf("TODO: send message {%s, %s}", name, password)
}

func (c Conn) CreateActor(name, password string) {
	log.Printf("TODO: send message {%s, %s}", name, password)
}
