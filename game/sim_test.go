package game

import (
	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/rpg2d/quad"
	"github.com/ghthor/engine/sim/stime"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

type spec_2moving struct {
	spec         string
	paths        []coord.PathAction
	expectations func(spec_2moving, actorIndex, gospec.Context)
}

func (t spec_2moving) runSpec(c gospec.Context) {
	pa0 := t.paths[0]
	pa1 := t.paths[1]

	index := actorIndex{
		0: &actor{
			actorEntity: actorEntity{
				id: 0,

				cell:   pa0.Orig,
				facing: pa0.Direction(),
			},
		},

		1: &actor{
			actorEntity: actorEntity{
				id: 1,

				cell:   pa1.Orig,
				facing: pa1.Direction(),
			},
		},
	}

	index[0].applyPathAction(&pa0)
	index[1].applyPathAction(&pa1)

	phase := narrowPhase{index}
	testCases := []struct {
		spec string
		cgrp quad.CollisionGroup
	}{{
		"AB", quad.CollisionGroup{}.AddCollision(quad.Collision{
			index[0].Entity(),
			index[1].Entity(),
		}),
	}, {
		"BA", quad.CollisionGroup{}.AddCollision(quad.Collision{
			index[1].Entity(),
			index[0].Entity(),
		}),
	}}

	for _, testCase := range testCases {
		c.Specify(testCase.spec, func() {
			stillExisting, removed := phase.ResolveCollisions(&testCase.cgrp, 0)
			c.Assume(len(stillExisting), Equals, 2)
			c.Assume(len(removed), Equals, 0)

			t.expectations(t, index, c)
		})
	}
}

type spec_1move_1stand struct {
	spec string

	// entity 0
	path coord.PathAction

	// entity 1
	cell   coord.Cell
	facing coord.Direction

	expectations func(spec_1move_1stand, actorIndex, gospec.Context)
}

func (t spec_1move_1stand) runSpec(c gospec.Context) {
	index := actorIndex{
		0: &actor{
			actorEntity: actorEntity{
				id:     0,
				cell:   t.path.Orig,
				facing: t.path.Direction(),
			},
		},

		1: &actor{
			actorEntity: actorEntity{
				id:     1,
				cell:   t.cell,
				facing: t.facing,
			},
		},
	}

	index[0].applyPathAction(&t.path)

	phase := narrowPhase{index}
	testCases := []struct {
		spec string
		cgrp quad.CollisionGroup
	}{{
		"AB", quad.CollisionGroup{}.AddCollision(quad.Collision{
			index[0].Entity(),
			index[1].Entity(),
		}),
	}, {
		"BA", quad.CollisionGroup{}.AddCollision(quad.Collision{
			index[1].Entity(),
			index[0].Entity(),
		}),
	}}

	c.Specify(t.spec, func() {
		for _, testCase := range testCases {
			c.Specify(testCase.spec, func() {
				stillExisting, removed := phase.ResolveCollisions(&testCase.cgrp, 0)
				c.Assume(len(stillExisting), Equals, 2)
				c.Assume(len(removed), Equals, 0)

				t.expectations(t, index, c)
			})
		}
	})
}

func DescribeCollision(c gospec.Context) {
	cell := func(x, y int) coord.Cell { return coord.Cell{x, y} }
	pa := func(start, speed int64, origin, dest coord.Cell) coord.PathAction {
		return coord.PathAction{
			Span: stime.NewSpan(stime.Time(start), stime.Time(start+speed)),
			Orig: origin,
			Dest: dest,
		}
	}

	c.Specify("a collision between", func() {
		c.Specify("2 actors", func() {

			c.Specify("that are both moving", func() {
				testCases := []spec_2moving{{
					spec: "in the same direction",
					paths: []coord.PathAction{
						pa(0, 5, cell(0, 0), cell(0, 1)),
						pa(0, 10, cell(0, 1), cell(0, 2)),
					},
					expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
						c.Expect(index[0].pathAction, IsNil)
						c.Expect(*index[1].pathAction, Equals, testCase.paths[1])
					},
				}, {
					spec: "head to head",
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

				c.Specify("and contesting the same location", func() {
					testCases := []spec_2moving{{
						spec: "priority goes to moving north",
						paths: []coord.PathAction{
							pa(0, 10, cell(0, 0), cell(0, 1)),
							pa(0, 10, cell(-1, 1), cell(0, 1)),
						},
						expectations: func(testCase spec_2moving, index actorIndex, c gospec.Context) {
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
	})
}
