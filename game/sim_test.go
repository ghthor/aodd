package game

import (
	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/sim/stime"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

var cell = func(x, y int) coord.Cell { return coord.Cell{x, y} }
var pa = func(start, speed int64, origin, dest coord.Cell) coord.PathAction {
	return coord.PathAction{
		Span: stime.NewSpan(stime.Time(start), stime.Time(start+speed)),
		Orig: origin,
		Dest: dest,
	}
}

func Describe2Actors(c gospec.Context) {
	c.Specify("a collision between 2 actors", func() {
		c.Specify("that are both moving", func() {
			testCases := []spec_2moving{{
				spec: "and attempting to swap positions",
				paths: []coord.PathAction{
					pa(0, 10, cell(0, 0), cell(0, 1)),
					pa(0, 10, cell(0, 1), cell(0, 0)),
				},
				expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
					c.Expect(index[0].pathAction, IsNil)
					c.Expect(index[1].pathAction, IsNil)
				},
			}}

			for _, testCase := range testCases {
				c.Specify(testCase.spec, func() {
					testCase.runSpec(c)
				})
			}

			c.Specify("in the same directions", func() {
				testCases := []spec_2moving{{
					spec: "allows the leader to move",
					paths: []coord.PathAction{
						pa(0, 10, cell(0, 0), cell(0, 1)),
						pa(0, 9, cell(0, -1), cell(0, 0)),
					},
					expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
						c.Assume(testCase.paths[0].Direction(), Equals, coord.North)
						c.Assume(testCase.paths[1].Direction(), Equals, coord.North)
						c.Expect(*index[0].pathAction, Equals, testCase.paths[0])
						c.Expect(index[1].pathAction, IsNil)
					},
				}, {
					spec: "allows both to move",
					paths: []coord.PathAction{
						pa(0, 10, cell(0, 0), cell(0, 1)),
						pa(0, 10, cell(0, -1), cell(0, 0)),
					},
					expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
						c.Assume(testCase.paths[0].Direction(), Equals, coord.North)
						c.Assume(testCase.paths[1].Direction(), Equals, coord.North)
						c.Expect(*index[0].pathAction, Equals, testCase.paths[0])
						c.Expect(*index[1].pathAction, Equals, testCase.paths[1])
					},
				}}

				for _, testCase := range testCases {
					c.Specify(testCase.spec, func() {
						testCase.runSpec(c)
					})
				}

			})
			c.Specify("in perpendicular directions", func() {
				testCases := []spec_2moving{{
					spec: "allows the leader to move",
					paths: []coord.PathAction{
						pa(0, 10, cell(0, 0), cell(1, 0)),
						pa(0, 9, cell(0, -1), cell(0, 0)),
					},
					expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
						c.Assume(testCase.paths[0].Direction(), Equals, coord.East)
						c.Assume(testCase.paths[1].Direction(), Equals, coord.North)
						c.Expect(*index[0].pathAction, Equals, testCase.paths[0])
						c.Expect(index[1].pathAction, IsNil)
					},
				}, {
					spec: "allows both to move",
					paths: []coord.PathAction{
						pa(0, 10, cell(0, 0), cell(1, 0)),
						pa(0, 10, cell(0, -1), cell(0, 0)),
					},
					expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
						c.Assume(testCase.paths[0].Direction(), Equals, coord.East)
						c.Assume(testCase.paths[1].Direction(), Equals, coord.North)
						c.Expect(*index[0].pathAction, Equals, testCase.paths[0])
						c.Expect(*index[1].pathAction, Equals, testCase.paths[1])
					},
				}, {
					spec: "allows both to move",
					paths: []coord.PathAction{
						pa(0, 10, cell(0, 0), cell(1, 0)),
						pa(0, 11, cell(0, -1), cell(0, 0)),
					},
					expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
						c.Assume(testCase.paths[0].Direction(), Equals, coord.East)
						c.Assume(testCase.paths[1].Direction(), Equals, coord.North)
						c.Expect(*index[0].pathAction, Equals, testCase.paths[0])
						c.Expect(*index[1].pathAction, Equals, testCase.paths[1])
					},
				}}

				for _, testCase := range testCases {
					c.Specify(testCase.spec, func() {
						testCase.runSpec(c)
					})
				}
			})

			c.Specify("and contesting the same location", func() {
				c.Specify("from the side", func() {

					testCases := []spec_2moving{{
						spec: "moving east loses to moving north",
						paths: []coord.PathAction{
							pa(0, 10, cell(0, 0), cell(0, 1)),
							pa(0, 10, cell(-1, 1), cell(0, 1)),
						},
						expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
							c.Assume(testCase.paths[0].Direction(), Equals, coord.North)
							c.Assume(testCase.paths[1].Direction(), Equals, coord.East)
							c.Expect(*index[0].pathAction, Equals, testCase.paths[0])
							c.Expect(index[1].pathAction, IsNil)
						},
					}, {
						spec: "moving south loses to moving east",
						paths: []coord.PathAction{
							pa(0, 10, cell(0, 0), cell(1, 0)),
							pa(0, 10, cell(1, 1), cell(1, 0)),
						},
						expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
							c.Assume(testCase.paths[0].Direction(), Equals, coord.East)
							c.Assume(testCase.paths[1].Direction(), Equals, coord.South)
							c.Expect(*index[0].pathAction, Equals, testCase.paths[0])
							c.Expect(index[1].pathAction, IsNil)
						},
					}, {
						spec: "moving west loses to moving south",
						paths: []coord.PathAction{
							pa(0, 10, cell(0, 0), cell(0, -1)),
							pa(0, 10, cell(1, -1), cell(0, -1)),
						},
						expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
							c.Assume(testCase.paths[0].Direction(), Equals, coord.South)
							c.Assume(testCase.paths[1].Direction(), Equals, coord.West)
							c.Expect(*index[0].pathAction, Equals, testCase.paths[0])
							c.Expect(index[1].pathAction, IsNil)
						},
					}, {
						spec: "moving slower loses to moving faster",
						paths: []coord.PathAction{
							pa(0, 9, cell(0, 0), cell(-1, 0)),
							pa(0, 10, cell(-1, -1), cell(-1, 0)),
						},
						expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
							c.Assume(testCase.paths[0].Direction(), Equals, coord.West)
							c.Assume(testCase.paths[1].Direction(), Equals, coord.North)
							c.Expect(*index[0].pathAction, Equals, testCase.paths[0])
							c.Expect(index[1].pathAction, IsNil)
						},
					}, {
						spec: "moving first wins",
						paths: []coord.PathAction{
							pa(0, 10, cell(0, 0), cell(-1, 0)),
							pa(1, 9, cell(-1, -1), cell(-1, 0)),
						},
						expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
							c.Assume(testCase.paths[0].Direction(), Equals, coord.West)
							c.Assume(testCase.paths[1].Direction(), Equals, coord.North)
							c.Expect(*index[0].pathAction, Equals, testCase.paths[0])
							c.Expect(index[1].pathAction, IsNil)
						},
					}}

					for _, testCase := range testCases {
						c.Specify(testCase.spec, func() {
							testCase.runSpec(c)
						})
					}
				})

				c.Specify("head to head", func() {
					testCases := []spec_2moving{{
						spec: "moving south loses to moving north",
						paths: []coord.PathAction{
							pa(0, 10, cell(0, 0), cell(0, 1)),
							pa(0, 10, cell(0, 2), cell(0, 1)),
						},
						expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
							c.Assume(testCase.paths[0].Direction(), Equals, coord.North)
							c.Assume(testCase.paths[1].Direction(), Equals, coord.South)
							c.Expect(*index[0].pathAction, Equals, testCase.paths[0])
							c.Expect(index[1].pathAction, IsNil)
						},
					}, {
						spec: "moving west loses to moving east",
						paths: []coord.PathAction{
							pa(0, 10, cell(0, 0), cell(1, 0)),
							pa(0, 10, cell(2, 0), cell(1, 0)),
						},
						expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
							c.Assume(testCase.paths[0].Direction(), Equals, coord.East)
							c.Assume(testCase.paths[1].Direction(), Equals, coord.West)
							c.Expect(*index[0].pathAction, Equals, testCase.paths[0])
							c.Expect(index[1].pathAction, IsNil)
						},
					}, {
						spec: "moving slower loses to moving faster",
						paths: []coord.PathAction{
							pa(0, 9, cell(0, 0), cell(-1, 0)),
							pa(0, 10, cell(-2, 0), cell(-1, 0)),
						},
						expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
							c.Assume(testCase.paths[0].Direction(), Equals, coord.West)
							c.Assume(testCase.paths[1].Direction(), Equals, coord.East)
							c.Expect(*index[0].pathAction, Equals, testCase.paths[0])
							c.Expect(index[1].pathAction, IsNil)
						},
					}, {
						spec: "moving first wins",
						paths: []coord.PathAction{
							pa(0, 10, cell(0, 0), cell(-1, 0)),
							pa(1, 9, cell(-2, 0), cell(-1, 0)),
						},
						expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
							c.Assume(testCase.paths[0].Direction(), Equals, coord.West)
							c.Assume(testCase.paths[1].Direction(), Equals, coord.East)
							c.Expect(*index[0].pathAction, Equals, testCase.paths[0])
							c.Expect(index[1].pathAction, IsNil)
						},
					}}

					for _, testCase := range testCases {
						c.Specify(testCase.spec, func() {
							testCase.runSpec(c)
						})
					}
				})
			})
		})

		c.Specify("where 1 is moving and 1 is standing still,", func() {
			testCases := []spec_1move_1stand{{
				"moving into stationary from the side",

				// entity 0
				pa(0, 10, cell(0, 0), cell(1, 0)),

				// entity 1
				cell(1, 0),
				coord.South,

				func(t spec_1move_1stand, index actorIndex, c gospec.Context) {
					c.Expect(index[0].pathAction, IsNil)
				},
			}, {
				"moving into stationary from behind",

				// entity 0
				pa(0, 10, cell(0, 0), cell(1, 0)),

				// entity 1
				cell(1, 0),
				coord.East,

				func(t spec_1move_1stand, index actorIndex, c gospec.Context) {
					c.Expect(index[0].pathAction, IsNil)
				},
			}, {
				"moving into stationary from in front",

				// entity 0
				pa(0, 10, cell(0, 0), cell(1, 0)),

				// entity 1
				cell(1, 0),
				coord.West,

				func(t spec_1move_1stand, index actorIndex, c gospec.Context) {
					c.Expect(index[0].pathAction, IsNil)
				},
			}}

			for _, testCase := range testCases {
				testCase.runSpec(c)
			}
		})
	})
}

func Describe3Actors(c gospec.Context) {
	c.Specify("a collision between 3 actors", func() {
		c.Specify("where 2 are moving and 1 is standing still", func() {
			testCases := []spec_2move_1stand{{
				spec: "to the west",

				paths: [...]coord.PathAction{
					pa(0, 10, cell(0, 0), cell(-1, 0)),
					pa(0, 10, cell(-1, 0), cell(-2, 0)),
				},

				cell:   cell(-2, 0),
				facing: coord.West,

				expectations: func(t spec_2move_1stand, index actorIndex, c gospec.Context) {
					pa0 := t.paths[0]
					pa1 := t.paths[1]

					a0 := index[0]
					a1 := index[1]
					a2 := index[2]

					c.Assume(pa0.Direction(), Equals, coord.West)
					c.Assume(pa1.Direction(), Equals, coord.West)
					c.Assume(a2.pathAction, IsNil)

					c.Expect(a0.pathAction, IsNil)
					c.Expect(a1.pathAction, IsNil)
				},
			}, {
				spec: "to the south and to the west",

				paths: [...]coord.PathAction{
					pa(0, 10, cell(1, 1), cell(1, 0)),
					pa(0, 10, cell(1, 0), cell(0, 0)),
				},

				cell:   cell(0, 0),
				facing: coord.West,

				expectations: func(t spec_2move_1stand, index actorIndex, c gospec.Context) {
					pa0 := t.paths[0]
					pa1 := t.paths[1]

					a0 := index[0]
					a1 := index[1]
					a2 := index[2]

					c.Assume(pa0.Direction(), Equals, coord.South)
					c.Assume(pa1.Direction(), Equals, coord.West)
					c.Assume(a2.pathAction, IsNil)

					collision := coord.NewPathCollision(pa0, pa1)
					c.Assume(collision.Type(), Equals, coord.CT_A_INTO_B_FROM_SIDE)

					c.Expect(a0.pathAction, IsNil)
					c.Expect(a1.pathAction, IsNil)
				},
			}}

			for _, testCase := range testCases {
				testCase.runSpec(c)
			}
		})

		c.Specify("where all are moving", func() {
			testCases := []spec_3move{{
				spec: "in a line to the west",

				paths: [...]coord.PathAction{
					pa(0, 10, cell(0, 0), cell(-1, 0)),
					pa(0, 10, cell(-1, 0), cell(-2, 0)),
					pa(0, 10, cell(-2, 0), cell(-3, 0)),
				},

				expectations: func(t spec_3move, index actorIndex, c gospec.Context) {
					pa0 := t.paths[0]
					pa1 := t.paths[1]
					pa2 := t.paths[2]

					a0 := index[0]
					a1 := index[1]
					a2 := index[2]

					c.Assume(pa0.Direction(), Equals, coord.West)
					c.Assume(pa1.Direction(), Equals, coord.West)
					c.Assume(pa2.Direction(), Equals, coord.West)

					c.Expect(*a0.pathAction, Equals, pa0)
					c.Expect(*a1.pathAction, Equals, pa1)
					c.Expect(*a2.pathAction, Equals, pa2)
				},
			}}

			for _, testCase := range testCases {
				testCase.runSpec(c)
			}
		})
	})
}

func DescribeSomeActors(c gospec.Context) {
	c.Specify("a collision between some actors", func() {
		c.Specify("that cycles", func() {
			testCases := []spec_allMoving{{
				spec: "with 4 actors in a square",

				paths: []coord.PathAction{
					pa(0, 10, cell(0, 0), cell(1, 0)),
					pa(0, 10, cell(1, 0), cell(1, 1)),
					pa(0, 10, cell(1, 1), cell(0, 1)),
					pa(0, 10, cell(0, 1), cell(0, 0)),
				},

				expectations: func(t spec_allMoving, index actorIndex, c gospec.Context) {
					c.Assume(t.paths[0].Direction(), Equals, coord.East)
					c.Assume(t.paths[1].Direction(), Equals, coord.North)
					c.Assume(t.paths[2].Direction(), Equals, coord.West)
					c.Assume(t.paths[3].Direction(), Equals, coord.South)

					c.Assume(coord.NewPathCollision(t.paths[0], t.paths[1]).Type(), Equals, coord.CT_A_INTO_B_FROM_SIDE)
					c.Assume(coord.NewPathCollision(t.paths[1], t.paths[2]).Type(), Equals, coord.CT_A_INTO_B_FROM_SIDE)
					c.Assume(coord.NewPathCollision(t.paths[2], t.paths[3]).Type(), Equals, coord.CT_A_INTO_B_FROM_SIDE)
					c.Assume(coord.NewPathCollision(t.paths[3], t.paths[0]).Type(), Equals, coord.CT_A_INTO_B_FROM_SIDE)

					c.Expect(index[0].pathAction, IsNil)
					c.Expect(index[1].pathAction, IsNil)
					c.Expect(index[2].pathAction, IsNil)
					c.Expect(index[3].pathAction, IsNil)
				},
			}, {
				spec: "8 actors in a square with an empty center cell",

				paths: []coord.PathAction{
					pa(0, 10, cell(1, 0), cell(2, 0)),
					pa(0, 10, cell(2, 0), cell(2, 1)),
					pa(0, 10, cell(2, 1), cell(2, 2)),
					pa(0, 10, cell(2, 2), cell(1, 2)),
					pa(0, 10, cell(1, 2), cell(0, 2)),
					pa(0, 10, cell(0, 2), cell(0, 1)),
					pa(0, 10, cell(0, 1), cell(0, 0)),
					pa(0, 10, cell(0, 0), cell(1, 0)),
				},

				expectations: func(t spec_allMoving, index actorIndex, c gospec.Context) {
					c.Assume(coord.NewPathCollision(t.paths[7], t.paths[0]).Type(), Equals, coord.CT_NONE)

					for i, _ := range t.paths {
						c.Expect(index[int64(i)].pathAction, IsNil)
					}
				},
			}, {
				spec: "with 4 actors in a square w/ one trailing outside of the square",

				paths: []coord.PathAction{
					pa(0, 10, cell(-2, 0), cell(-1, 0)),
					pa(0, 10, cell(0, 0), cell(1, 0)),
					pa(0, 10, cell(-1, 0), cell(0, 0)),
					pa(0, 10, cell(1, 0), cell(1, 1)),
					pa(0, 10, cell(1, 1), cell(0, 1)),
					pa(0, 10, cell(0, 1), cell(0, 0)),
				},

				expectations: func(t spec_allMoving, index actorIndex, c gospec.Context) {
					for i, _ := range t.paths {
						c.Expect(index[int64(i)].pathAction, IsNil)
					}
				},
			}}

			for _, testCase := range testCases {
				testCase.runSpec(c)
			}
		})
	})
}
