package game

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/ghthor/engine/rpg2d"
	"github.com/ghthor/engine/rpg2d/coord"
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

type serveIndex struct {
	tmpl     *template.Template
	settings clientSettings
}

type clientSettings struct {
	WebsocketURL string
	Simulation   simulationSettings
}

type simulationSettings struct {
	Width, Height int
}

func (html serveIndex) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := html.tmpl.Execute(w, html.settings)

	if err != nil {
		http.Error(w, fmt.Sprint("template error:", err), http.StatusInternalServerError)
	}
}

func NewSimShard(laddr string, indexTmpl *template.Template) (*http.Server, error) {
	// TODO pull this information from a datastore
	quadTree, err := quad.New(coord.Bounds{
		coord.Cell{-1024, 1024},
		coord.Cell{1023, -1023},
	}, 40, nil)

	if err != nil {
		return nil, err
	}

	now := stime.Time(0)

	simDef := rpg2d.SimulationDef{
		FPS: 40,

		// Initial World State
		QuadTree: quadTree,
		Now:      now,

		InputPhaseHandler:  inputPhase{},
		NarrowPhaseHandler: narrowPhase{},
	}

	runningSim, err := simDef.Begin()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()

	wsRoute := "/actor/socket"

	indexHandler := serveIndex{
		indexTmpl,
		clientSettings{
			"wss://" + laddr + wsRoute,
			simulationSettings{
				Width:  quadTree.Bounds().Width(),
				Height: quadTree.Bounds().Height(),
			},
		},
	}

	mux.Handle("/", indexHandler)
	mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("www/js/"))))
	mux.Handle(wsRoute, newWebsocketActorHandler(runningSim))

	return &http.Server{
		Addr:    laddr,
		Handler: mux,
	}, nil
}
