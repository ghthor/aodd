//go:generate stringer -type=event
//go:generate gopherjs build

// +build js
package main

import (
	"fmt"
	"time"

	"github.com/gopherjs/gopherjs/js"
)

type jsObject map[string]interface{}

type ticker struct {
	EventPublisher
}

func newTicker(pub EventPublisher) ticker {
	return ticker{pub}
}

func (t ticker) start() {
	go func() {
		ticker := time.NewTicker(time.Second * 1)

		for {
			select {
			case time := <-ticker.C:
				t.Emit(EV_CONNECTED, jsObject{
					"time": time,
				})

				ticker.Stop()
			}
		}
	}()
}

type EventPublisher interface {
	Emit(fmt.Stringer, jsObject)
}

type eventPublisher struct {
	*js.Object
}

func (e eventPublisher) Emit(event fmt.Stringer, params jsObject) {
	e.Call("emit", event.String(), [1]jsObject{params})
}

type EventEmitter interface {
	On(string, func(...jsObject))
}

type event int

const (
	EV_CONNECTED event = iota
)

// Key used on the window object
// window.gopherjsApplication
const moduleKey = "gopherjsApplication"

type conn struct{}

func (conn) attemptLogin(name, password string) {
	js.Global.Get("console").Call("log", "TOOD: attempt to login", name, password)
}

func (conn) createActor(name, password string) {
	js.Global.Get("console").Call("log", "TODO: create an actor", name, password)
}

func main() {
	c := conn{}

	js.Global.Set(moduleKey, jsObject{
		"moduleKey": moduleKey,

		"setTickerUI": func(pub *js.Object) {
			ticker := ticker{eventPublisher{pub}}
			ticker.start()
		},

		"attemptLogin": c.attemptLogin,
		"createActor":  c.createActor,

		EV_CONNECTED.String(): EV_CONNECTED.String(),
	})
}
