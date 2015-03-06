//go:generate stringer -type=event
//go:generate gopherjs build

// +build js
package main

import (
	"fmt"
	"log"

	"github.com/ghthor/aodd/game/client"

	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/websocket"
)

type jsObject map[string]interface{}

type EventPublisher interface {
	Emit(fmt.Stringer, jsObject)
}

type eventPublisher struct {
	*js.Object
}

func (e eventPublisher) Emit(event fmt.Stringer, params jsObject) {
	e.Call("emit", event.String(), [1]jsObject{params})
}

type event int

const (
	EV_CONNECTED event = iota
)

// Key used on the window object
// window.gopherjsApplication
const moduleKey = "gopherjsApplication"

func main() {
	js.Global.Set(moduleKey, jsObject{
		"moduleKey":  moduleKey,
		"initialize": initialize,
	})
}

func initialize(settings *js.Object) jsObject {
	return jsObject{
		"moduleKey": moduleKey,

		"dial": func(pub *js.Object) {
			go func() {
				wsUrl := settings.Get("websocketURL").String()

				ws, err := websocket.Dial(wsUrl)
				if err != nil {
					log.Fatal(err)
				}

				conn := client.NewConn(ws)

				eventPublisher{pub}.Emit(EV_CONNECTED, jsObject{
					"attemptLogin": conn.AttemptLogin,
					"createActor":  conn.CreateActor,
				})
			}()
		},

		EV_CONNECTED.String(): EV_CONNECTED.String(),
	}
}
