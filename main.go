package main

import (
	"fmt"
	"log"

	"github.com/ghthor/engine/rpg2d"
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

func (narrowPhase) ResolveCollisions(c quad.Chunk, now stime.Time) quad.Chunk {
	return c
}

func main() {
	simDef := rpg2d.SimulationDef{
		FPS: 40,

		InputPhaseHandler:  inputPhase{},
		NarrowPhaseHandler: narrowPhase{},
	}

	_, err := simDef.Begin()
	if err != nil {
		log.Fatal(err)
	}
}
