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
				t.Emit(EV_TICK, jsObject{
					"time": time,
				})
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

type LoginFn func(name, password string)
type CreateFn func(name, password string)

type event int

const (
	EV_TICK event = iota
)

// Key used on the window object
// window.gopherjsApplication
const moduleKey = "gopherjsApplication"

func main() {
	js.Global.Set(moduleKey, jsObject{
		"moduleKey": moduleKey,

		"setTickerUI": func(pub *js.Object) {
			ticker := ticker{eventPublisher{pub}}
			ticker.start()
		},

		EV_TICK.String(): EV_TICK.String(),
	})
}
