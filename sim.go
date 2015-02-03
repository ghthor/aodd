package main

import (
	"fmt"

	"github.com/ghthor/engine/rpg2d/quad"
	"github.com/ghthor/engine/sim/stime"
)

type inputPhase struct{}
type narrowPhase struct{}

func (inputPhase) ApplyInputsIn(c quad.Chunk, now stime.Time) quad.Chunk {
	for _, e := range c.Entities {
		switch a := e.(type) {
		case actor:
			input := a.ReadInput()
			fmt.Println(input)

			// Naively apply input to actor
		}
	}
	return c
}

func (narrowPhase) ResolveCollisions(c quad.CollisionGroup, now stime.Time) quad.CollisionGroup {
	return c
}
