package canvas_test

import (
	"fmt"

	"github.com/ghthor/aodd/game/client/canvas"
	"github.com/ghthor/filu/rpg2d"
	"github.com/ghthor/filu/rpg2d/coord"
	"github.com/ghthor/filu/rpg2d/quad"
	"github.com/ghthor/filu/sim/stime"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

type terrainContext struct {
	rpg2d.TerrainMap
}

func (t *terrainContext) Reset(slice rpg2d.TerrainMapStateSlice) {
	tm, err := rpg2d.NewTerrainMap(slice.Bounds, slice.Terrain)
	if err != nil {
		panic(err)
	}

	t.TerrainMap = tm
}

func (t *terrainContext) Shift(dir canvas.TerrainShift, mags canvas.TerrainShiftMagnitudes) {
	switch dir {
	case canvas.TS_NORTH:
		t.shiftNorth(mags[coord.North])

	case canvas.TS_NORTHEAST:
		t.shiftNorth(mags[coord.North])
		t.shiftEast(mags[coord.East])

	case canvas.TS_EAST:
		t.shiftEast(mags[coord.East])

	case canvas.TS_SOUTHEAST:
		t.shiftSouth(mags[coord.South])
		t.shiftEast(mags[coord.East])

	case canvas.TS_SOUTH:
		t.shiftSouth(mags[coord.South])

	case canvas.TS_SOUTHWEST:
		t.shiftSouth(mags[coord.South])
		t.shiftWest(mags[coord.West])

	case canvas.TS_WEST:
		t.shiftWest(mags[coord.West])

	case canvas.TS_NORTHWEST:
		t.shiftNorth(mags[coord.North])
		t.shiftWest(mags[coord.West])

	default:
		panic(fmt.Sprintf("unknown canvas shift direction %v", dir))
	}
}

func joinWithEmptySpace(newBounds coord.Bounds, m rpg2d.TerrainMap, diffs []coord.Bounds) rpg2d.TerrainMap {
	maps := make([]rpg2d.TerrainMap, 0, len(diffs)+1)

	for _, d := range diffs {
		m, err := rpg2d.NewTerrainMap(d, string(' '))
		if err != nil {
			panic(err)
		}

		maps = append(maps, m)
	}

	maps = append(maps, m.Slice(newBounds))

	m, err := rpg2d.JoinTerrain(newBounds, maps...)
	if err != nil {
		panic(err)
	}

	return m
}

// Shifts the image drawn on the canvas to the north
// freeing up space to the south to draw in the new
// tiles.
func (t *terrainContext) shiftNorth(mag int) {
	// move bounds to south
	newBounds := coord.Bounds{
		t.Bounds.TopL.Add(0, -mag),
		t.Bounds.BotR.Add(0, -mag),
	}

	t.TerrainMap = joinWithEmptySpace(newBounds, t.TerrainMap, t.Bounds.DiffFrom(newBounds))
}

func (t *terrainContext) shiftEast(mag int) {
	// move bounds to the west
	newBounds := coord.Bounds{
		t.Bounds.TopL.Add(-mag, 0),
		t.Bounds.BotR.Add(-mag, 0),
	}

	t.TerrainMap = joinWithEmptySpace(newBounds, t.TerrainMap, t.Bounds.DiffFrom(newBounds))
}

func (t *terrainContext) shiftSouth(mag int) {
	// move bounds to the north
	newBounds := coord.Bounds{
		t.Bounds.TopL.Add(0, mag),
		t.Bounds.BotR.Add(0, mag),
	}

	t.TerrainMap = joinWithEmptySpace(newBounds, t.TerrainMap, t.Bounds.DiffFrom(newBounds))
}

func (t *terrainContext) shiftWest(mag int) {
	// move bounds to the east
	newBounds := coord.Bounds{
		t.Bounds.TopL.Add(mag, 0),
		t.Bounds.BotR.Add(mag, 0),
	}

	t.TerrainMap = joinWithEmptySpace(newBounds, t.TerrainMap, t.Bounds.DiffFrom(newBounds))
}

func (t *terrainContext) DrawTile(terrainType rpg2d.TerrainType, cell coord.Cell) {
	t.SetType(terrainType, cell)
}

func DescribeTerrainCanvas(c gospec.Context) {
	c.Specify("a terrain canvas", func() {
		quadTree, err := quad.New(coord.Bounds{
			coord.Cell{-4, 4},
			coord.Cell{3, -3},
		}, 20, nil)
		c.Assume(err, IsNil)

		terrain, err := rpg2d.NewTerrainMap(quadTree.Bounds(), `
DDDDDDDD
DGGGGGGD
DGGRRGGD
DGRRRRGD
DGRRRRGD
DGGRRGGD
DGGGGGGD
DDDDDDDD
`)
		c.Assume(err, IsNil)

		world := rpg2d.NewWorld(stime.Time(0), quadTree, terrain)

		worldState := world.ToState()

		north := coord.Bounds{
			coord.Cell{-2, 4},
			coord.Cell{1, 1},
		}

		northEast := coord.Bounds{
			coord.Cell{0, 4},
			coord.Cell{3, 1},
		}

		east := coord.Bounds{
			coord.Cell{0, 2},
			coord.Cell{3, -1},
		}

		southEast := coord.Bounds{
			coord.Cell{0, 0},
			coord.Cell{3, -3},
		}

		south := coord.Bounds{
			coord.Cell{-2, 0},
			coord.Cell{1, -3},
		}

		southWest := coord.Bounds{
			coord.Cell{-4, 0},
			coord.Cell{-1, -3},
		}

		west := coord.Bounds{
			coord.Cell{-4, 2},
			coord.Cell{-1, -1},
		}

		northWest := coord.Bounds{
			coord.Cell{-4, 4},
			coord.Cell{-1, 1},
		}

		center := coord.Bounds{
			coord.Cell{-2, 2},
			coord.Cell{1, -1},
		}

		c.Specify("can be reset by a diff if there is no overlap", func() {
			initialState := worldState.Cull(northWest)

			context := &terrainContext{
				TerrainMap: initialState.Clone().TerrainMap.TerrainMap,
			}

			nextState := worldState.Cull(northEast)

			canvas.ApplyTerrainDiff(context, initialState, initialState.Diff(nextState))
			c.Expect(context.String(), Equals, nextState.TerrainMap.String())
		})

		c.Specify("can be shifted by a diff to the", func() {
			initialState := worldState.Cull(center)

			context := &terrainContext{
				TerrainMap: initialState.Clone().TerrainMap.TerrainMap,
			}

			var nextState rpg2d.WorldState

			expectContextIsUpdated := func() {
				canvas.ApplyTerrainDiff(context, initialState, initialState.Diff(nextState))
				c.Expect(context.String(), Equals, nextState.TerrainMap.String())
			}

			c.Specify("north", func() {
				nextState = worldState.Cull(north)
				expectContextIsUpdated()
			})

			c.Specify("north & east", func() {
				nextState = worldState.Cull(northEast)
				expectContextIsUpdated()
			})

			c.Specify("east", func() {
				nextState = worldState.Cull(east)
				expectContextIsUpdated()
			})

			c.Specify("south & east", func() {
				nextState = worldState.Cull(southEast)
				expectContextIsUpdated()
			})

			c.Specify("south", func() {
				nextState = worldState.Cull(south)
				expectContextIsUpdated()
			})

			c.Specify("south & west", func() {
				nextState = worldState.Cull(southWest)
				expectContextIsUpdated()
			})

			c.Specify("west", func() {
				nextState = worldState.Cull(west)
				expectContextIsUpdated()
			})

			c.Specify("north & west", func() {
				nextState = worldState.Cull(northWest)
				expectContextIsUpdated()
			})
		})
	})
}
