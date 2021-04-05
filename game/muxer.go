package game

import (
	"github.com/ghthor/filu/rpg2d"
	"github.com/ghthor/filu/rpg2d/coord"
	"github.com/ghthor/filu/rpg2d/quad/quadstate"
	"github.com/ghthor/filu/rpg2d/worldstate"
	"github.com/ghthor/filu/rpg2d/worldterrain"
	"github.com/ghthor/filu/sim/stime"
)

type InitialStateWriter interface {
	WriteWorldState(*worldstate.Snapshot) DiffWriter
}

type DiffWriter interface {
	WriteWorldStateDiff(*worldstate.Update)
}

type actorInputState struct {
	*moveCmd
	*useCmd
	*chatCmd
}

type actorConn struct {
	inputState chan actorInputState

	// External connection used to publish the initial world state
	conn       InitialStateWriter
	diffWriter DiffWriter

	initialState         *worldstate.Snapshot
	prevState, nextState *worldstate.Snapshot
	prevBloom, nextBloom worldstate.InverseBloomMap

	diff *worldstate.Update
}

func newActorConn(conn InitialStateWriter) actorConn {
	c := actorConn{
		conn:       conn,
		inputState: make(chan actorInputState, 1),

		prevBloom: worldstate.NewInverseBloomMap(100),
		nextBloom: worldstate.NewInverseBloomMap(100),
		diff:      worldstate.NewUpdate(1),
	}

	c.inputState <- actorInputState{}
	return c
}

func ActorCullBounds(center coord.Cell) coord.Bounds {
	return coord.Bounds{
		center.Add(-26, 26),
		center.Add(26, -26),
	}
}

// TODO Remove this method requirement from rpg2d.Actor interface
func (a *actor) WriteState(state rpg2d.WorldState) {
}

// TODO Remove this method requirement from rpg2d.Actor interface
func (a *actorConn) WriteState(state rpg2d.WorldState) {
}

// TODO Port all this state initialization work into filu
//      Actor will need to be able to provide the bounds for this to happen.
func (a *actor) WriteStateNext(now stime.Time, quad quadstate.Quad, terrain *worldterrain.MapState, encoder chan<- quadstate.EncodingRequest) {
	bounds := ActorCullBounds(a.Cell())

	if a.initialState == nil {
		state := worldstate.NewSnapshot(now, bounds, 0)
		quad.QueryBounds(bounds, state, quadstate.QueryAll)
		state.TerrainMap = &worldterrain.MapState{Map: terrain.Map.Slice(bounds)}

		a.prevState = state.Clone()
		a.nextState = state.Clone()
		a.diff = worldstate.NewUpdate(1)

		a.actorConn.WriteStateNext(state, encoder)
	} else {
		a.nextState.Clear()
		a.nextState.Time = now
		a.nextState.Bounds = bounds
		quad.QueryBounds(bounds, a.nextState, quadstate.QueryAll)
		a.nextState.TerrainMap = &worldterrain.MapState{Map: terrain.Map.Slice(bounds)}

		a.actorConn.WriteStateNext(a.nextState, encoder)
	}
}

func (a *actorConn) WriteStateNext(state *worldstate.Snapshot, encoder chan<- quadstate.EncodingRequest) {
	if a.initialState == nil {
		a.initialState = state
		CacheEncodingsFor([][]*quadstate.Entity{
			state.Entities.ByType[quadstate.TypeRemoved],
			state.Entities.ByType[quadstate.TypeNew],
			state.Entities.ByType[quadstate.TypeChanged],
			state.Entities.ByType[quadstate.TypeUnchanged],
		}, encoder)
		a.diffWriter = a.conn.WriteWorldState(state)
	} else {
		a.nextState = state
		a.nextBloom.Reset()
		a.diff.FromSnapshot(
			a.prevState, a.nextState,
			a.prevBloom, a.nextBloom,
		)

		if len(a.diff.Entities) > 0 || len(a.diff.Removed) > 0 || a.diff.TerrainMapSlices != nil {
			CacheEncodingsFor([][]*quadstate.Entity{a.diff.Removed, a.diff.Entities}, encoder)
			a.diffWriter.WriteWorldStateDiff(a.diff)
		}
	}

	a.prevState, a.nextState = a.nextState, a.prevState
	a.prevBloom, a.nextBloom = a.nextBloom, a.prevBloom
}

// TODO move this to a more apprioate place
func CacheEncodingsFor(entities [][]*quadstate.Entity, encoder chan<- quadstate.EncodingRequest) {
	done := make(chan [][]*quadstate.Entity)
	encoder <- quadstate.EncodingRequest{entities, done}
	<-done
	close(done)
}
