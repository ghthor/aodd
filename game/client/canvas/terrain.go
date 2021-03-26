package canvas

import (
	"errors"
	"fmt"

	"github.com/ghthor/filu/rpg2d/coord"
	"github.com/ghthor/filu/rpg2d/worldterrain"
)

type TerrainShift int
type TerrainShiftMagnitudes map[coord.Direction]int

//go:generate stringer -type=TerrainShift
const (
	TS_NORTH TerrainShift = iota
	TS_NORTHEAST
	TS_EAST
	TS_SOUTHEAST
	TS_SOUTH
	TS_SOUTHWEST
	TS_WEST
	TS_NORTHWEST
)

type TerrainContext interface {
	// Used if the bounds of the diff does NOT
	// have any overlap with the bounds of the
	// previous state.
	Reset(worldterrain.MapStateSlice)

	// Shift should expect A direction the canvas
	// should be shifted to maintian the tiles that
	// have already been drawn to it.
	// The magnitude is the number of tiles it should
	// be shifted for each direction.
	Shift(TerrainShift, TerrainShiftMagnitudes)

	// DrawTile should expect a terrain type for
	// the tile and a cell with the absolute coord
	// where the cell should be drawn.
	DrawTile(worldterrain.Type, coord.Cell)
}

func abs(a int) int {
	if a < 0 {
		a = -a
	}

	return a
}

// Taking a previous world state and a diff to the next
// state, this function will call the terrain context
// with the necessary steps to apply the diff and update
// the painting of the canvas.
func ApplyTerrainDiff(c TerrainContext, prevBounds coord.Bounds, diff *worldterrain.MapStateSlices) error {
	if diff == nil {
		return nil
	}

	if len(diff.Slices) == 0 {
		return nil
	}

	if !prevBounds.Overlaps(diff.Bounds) {
		if len(diff.Slices) != 1 {
			return fmt.Errorf("unexpected number of terrain state slices {%#v}", diff.Slices)
		}

		c.Reset(diff.Slices[0])
		return nil
	}

	pb, nb := prevBounds, diff.Bounds
	switch {
	case pb.Contains(nb.BotL()) && pb.Contains(nb.BotR):
		c.Shift(TS_SOUTH, TerrainShiftMagnitudes{
			coord.South: abs(nb.TopL.Y - pb.TopL.Y),
		})

	case pb.Contains(nb.TopL) && pb.Contains(nb.BotL()):
		c.Shift(TS_WEST, TerrainShiftMagnitudes{
			coord.West: abs(nb.BotR.X - pb.BotR.X),
		})

	case pb.Contains(nb.TopL) && pb.Contains(nb.TopR()):
		c.Shift(TS_NORTH, TerrainShiftMagnitudes{
			coord.North: abs(nb.BotR.Y - pb.BotR.Y),
		})

	case pb.Contains(nb.TopR()) && pb.Contains(nb.BotR):
		c.Shift(TS_EAST, TerrainShiftMagnitudes{
			coord.East: abs(nb.TopL.X - pb.TopL.X),
		})

	case pb.Contains(nb.BotL()):
		c.Shift(TS_SOUTHWEST, TerrainShiftMagnitudes{
			coord.South: abs(nb.TopL.Y - pb.TopL.Y),
			coord.West:  abs(nb.BotR.X - pb.BotR.X),
		})

	case pb.Contains(nb.TopL):
		c.Shift(TS_NORTHWEST, TerrainShiftMagnitudes{
			coord.North: abs(nb.BotR.Y - pb.BotR.Y),
			coord.West:  abs(nb.BotR.X - pb.BotR.X),
		})

	case pb.Contains(nb.TopR()):
		c.Shift(TS_NORTHEAST, TerrainShiftMagnitudes{
			coord.North: abs(nb.BotR.Y - pb.BotR.Y),
			coord.East:  abs(nb.TopL.X - pb.TopL.X),
		})

	case pb.Contains(nb.BotR):
		c.Shift(TS_SOUTHEAST, TerrainShiftMagnitudes{
			coord.South: abs(nb.TopL.Y - pb.TopL.Y),
			coord.East:  abs(nb.TopL.X - pb.TopL.X),
		})

	default:
		return errors.New("invalid terrain diff")
	}

	for _, m := range diff.Slices {
		types, err := worldterrain.NewType2dArray(m.Bounds, m.Terrain)
		if err != nil {
			return err
		}

		for y, row := range types {
			for x, tt := range row {
				c.DrawTile(tt, m.Bounds.TopL.Add(x, -y))
			}
		}
	}

	return nil
}
