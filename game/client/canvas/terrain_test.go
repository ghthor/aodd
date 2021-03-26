package canvas_test

import (
	"fmt"

	"github.com/ghthor/aodd/game/client/canvas"
	"github.com/ghthor/filu/rpg2d/coord"
	"github.com/ghthor/filu/rpg2d/quad"
	"github.com/ghthor/filu/rpg2d/worldterrain"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

type terrainContext struct {
	worldterrain.Map
}

func (t *terrainContext) Reset(slice worldterrain.MapStateSlice) {
	tm, err := worldterrain.NewMap(slice.Bounds, slice.Terrain)
	if err != nil {
		panic(err)
	}

	t.Map = tm
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

func joinWithEmptySpace(newBounds coord.Bounds, m worldterrain.Map, diffs []coord.Bounds) worldterrain.Map {
	maps := make([]worldterrain.Map, 0, len(diffs)+1)

	for _, d := range diffs {
		m, err := worldterrain.NewMap(d, string(' '))
		if err != nil {
			panic(err)
		}

		maps = append(maps, m)
	}

	maps = append(maps, m.Slice(newBounds))

	m, err := worldterrain.JoinTerrain(newBounds, maps...)
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

	t.Map = joinWithEmptySpace(newBounds, t.Map, t.Bounds.DiffFrom(newBounds))
}

func (t *terrainContext) shiftEast(mag int) {
	// move bounds to the west
	newBounds := coord.Bounds{
		t.Bounds.TopL.Add(-mag, 0),
		t.Bounds.BotR.Add(-mag, 0),
	}

	t.Map = joinWithEmptySpace(newBounds, t.Map, t.Bounds.DiffFrom(newBounds))
}

func (t *terrainContext) shiftSouth(mag int) {
	// move bounds to the north
	newBounds := coord.Bounds{
		t.Bounds.TopL.Add(0, mag),
		t.Bounds.BotR.Add(0, mag),
	}

	t.Map = joinWithEmptySpace(newBounds, t.Map, t.Bounds.DiffFrom(newBounds))
}

func (t *terrainContext) shiftWest(mag int) {
	// move bounds to the east
	newBounds := coord.Bounds{
		t.Bounds.TopL.Add(mag, 0),
		t.Bounds.BotR.Add(mag, 0),
	}

	t.Map = joinWithEmptySpace(newBounds, t.Map, t.Bounds.DiffFrom(newBounds))
}

func (t *terrainContext) DrawTile(terrainType worldterrain.Type, cell coord.Cell) {
	t.SetType(terrainType, cell)
}

func DescribeTerrainCanvas(c gospec.Context) {
	c.Specify("a terrain canvas", func() {
		quadTree, err := quad.New(coord.Bounds{
			coord.Cell{-4, 4},
			coord.Cell{3, -3},
		}, 20, nil)
		c.Assume(err, IsNil)

		terrain, err := worldterrain.NewMap(quadTree.Bounds(), `
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
			initialMap := terrain.Slice(northWest)
			context := &terrainContext{
				Map: initialMap,
			}
			nextMap := terrain.Slice(northEast)

			canvas.ApplyTerrainDiff(context, initialMap.Bounds, initialMap.ToState().Diff(nextMap.ToState()))
			c.Expect(context.String(), Equals, nextMap.String())
		})

		c.Specify("can be shifted by a diff to the", func() {
			initialMap := terrain.Slice(center)
			context := &terrainContext{
				Map: initialMap,
			}
			var nextMap worldterrain.Map

			expectContextIsUpdated := func() {
				diff := initialMap.ToState().Diff(nextMap.ToState())
				canvas.ApplyTerrainDiff(context, initialMap.Bounds, diff)
				c.Expect(context.String(), Equals, nextMap.String())
			}

			c.Specify("north", func() {
				nextMap = terrain.Slice(north)
				expectContextIsUpdated()
			})

			c.Specify("north & east", func() {
				nextMap = terrain.Slice(northEast)
				expectContextIsUpdated()
			})

			c.Specify("east", func() {
				nextMap = terrain.Slice(east)
				expectContextIsUpdated()
			})

			c.Specify("south & east", func() {
				nextMap = terrain.Slice(southEast)
				expectContextIsUpdated()
			})

			c.Specify("south", func() {
				nextMap = terrain.Slice(south)
				expectContextIsUpdated()
			})

			c.Specify("south & west", func() {
				nextMap = terrain.Slice(southWest)
				expectContextIsUpdated()
			})

			c.Specify("west", func() {
				nextMap = terrain.Slice(west)
				expectContextIsUpdated()
			})

			c.Specify("north & west", func() {
				nextMap = terrain.Slice(northWest)
				expectContextIsUpdated()
			})
		})
	})
}
