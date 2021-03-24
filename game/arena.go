package game

import (
	"github.com/ghthor/filu/rpg2d/coord"
	"github.com/ghthor/filu/rpg2d/entity"
	"github.com/ghthor/filu/rpg2d/quad"
)

type wallEntity struct {
	id    entity.Id
	cell  coord.Cell
	flags entity.Flag
}

type WallEntityState struct {
	Type string     `json:"type"`
	Id   entity.Id  `json:"id"`
	Cell coord.Cell `jsonZ:"cell"`
}

func (w wallEntity) Id() entity.Id        { return w.id }
func (w wallEntity) Cell() coord.Cell     { return w.cell }
func (w wallEntity) Bounds() coord.Bounds { return coord.Bounds{w.cell, w.cell} }
func (w wallEntity) Flags() entity.Flag   { return w.flags }

func (w wallEntity) ToState() entity.State {
	return WallEntityState{
		Type: "wall",
		Id:   w.id,
		Cell: w.cell,
	}
}

func (w WallEntityState) EntityId() entity.Id  { return w.Id }
func (w WallEntityState) Bounds() coord.Bounds { return coord.Bounds{w.Cell, w.Cell} }
func (w WallEntityState) IsDifferentFrom(entity.State) bool {
	return false
}

func addWalls(quad quad.Quad, nextId func() entity.Id) quad.Quad {
	c := func(x, y int) coord.Cell { return coord.Cell{x, y} }
	newWall := func(c coord.Cell) {
		quad = quad.Insert(wallEntity{
			id:   nextId(),
			cell: c,
		})
	}

	// Outer Wall
	// {c(30, -30), c(99, -30)},
	for x := 30; x < 100; x++ {
		newWall(c(x, -30))
	}

	// {c(100, -30), c(100, -99)},
	for y := -30; y > -100; y-- {
		newWall(c(100, y))
	}

	// {c(31, -100), c(100, -100)},
	for x := 26; x <= 100; x++ {
		newWall(c(x, -100))
	}

	// {c(30, -31), c(30, -100)},
	for y := -31; y >= -100; y-- {
		newWall(c(30, y))
	}

	// Inner Wall
	// {c(45, -45), c(84, -45)}
	for x := 45; x < 85; x++ {
		newWall(c(x, -45))
	}

	// {c(85, -45), c(85, -84)}
	for y := -45; y > -85; y-- {
		newWall(c(85, y))
	}

	// {c(46, -85), c(85, -85)}
	for x := 46; x <= 85; x++ {
		newWall(c(x, -85))
	}

	// {c(45, -46), c(45, -85)}
	for y := -46; y >= -85; y-- {
		newWall(c(45, y))
	}

	return quad
}
