package game

import (
	"fmt"

	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/sim"
)

type actorConn struct {
	readInput <-chan string
	sendState chan<- interface{}
	stop      chan<- chan<- struct{}
}

type actor struct {
	id   int64
	cell coord.Cell

	actorConn
}

// Implement sim.Actor
func (a actor) Id() int64 { return a.id }
func (a actor) Conn() sim.StateWriter {
	return sim.StateWriter(a)
}

// Implement entity.Entity
func (a actor) Cell() coord.Cell { return a.cell }
func (a actor) Bounds() coord.Bounds {
	return coord.Bounds{}
}

func (a actorConn) startIO() {
	// Setup communication channels
	inputCh := make(chan string)
	outputCh := make(chan interface{})
	stopCh := make(chan chan<- struct{})

	// Set the channels accessible to the outside world
	a.readInput = inputCh
	a.sendState = outputCh
	a.stop = stopCh

	// Establish the channel endpoints used inside the loop
	var sendInput chan<- string
	var newState <-chan interface{}
	var stopReq <-chan chan<- struct{}

	sendInput = inputCh
	newState = outputCh
	stopReq = stopCh

	go func() {
		var hasStopped chan<- struct{}

	inputLoop:
		for {
			select {
			case hasStopped = <-stopReq:
				break inputLoop
			default:
			}

			select {
			case sendInput <- "":
			case s := <-newState:
				fmt.Println(s)
			case hasStopped = <-stopReq:
				break inputLoop
			}
		}

		hasStopped <- struct{}{}
	}()
}

func (c actorConn) ReadInput() string {
	return <-c.readInput
}

func (c actorConn) WriteState(s interface{}) error {
	c.sendState <- s
	return nil
}

func (a actorConn) stopIO() {
	hasStopped := make(chan struct{})

	a.stop <- hasStopped
	<-hasStopped
}

type warrior struct {
	actor
}

type wizard struct {
	actor
}

type priest struct {
	actor
}
