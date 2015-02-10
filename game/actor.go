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
	id int64

	actorConn
}

// Object stored in the quad tree
type actorEntity struct {
	id int64

	cell   coord.Cell
	facing coord.Direction

	pathActions []coord.PathAction
}

// Implement sim.Actor
func (a actor) Id() int64 { return a.id }
func (a actor) Conn() sim.StateWriter {
	return sim.StateWriter(a)
}

func (e actorEntity) Id() int64        { return e.id }
func (e actorEntity) Cell() coord.Cell { return e.cell }
func (e actorEntity) Bounds() coord.Bounds {
	bounds := coord.Bounds{
		e.cell,
		e.cell,
	}

	for _, a := range e.pathActions {
		bounds = bounds.Join(a.Bounds())
	}

	return bounds
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
