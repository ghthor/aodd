package game

import (
	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/rpg2d/quad"
	"github.com/ghthor/engine/sim/stime"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

func DescribeCollision(c gospec.Context) {
	cell := func(x, y int) coord.Cell { return coord.Cell{x, y} }
	pa := func(start, speed int64, origin, dest coord.Cell) *coord.PathAction {
		return &coord.PathAction{
			Span: stime.NewSpan(stime.Time(start), stime.Time(start+speed)),
			Orig: origin,
			Dest: dest,
		}
	}

	c.Specify("a collision between", func() {
		c.Specify("2 actors", func() {
			c.Specify("that are both moving", func() {
				c.Specify("in the same direction", func() {
					pa0 := pa(0, 5, cell(0, 0), cell(0, 1))
					pa1 := pa(0, 10, cell(0, 1), cell(0, 2))

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

					index[0].applyPathAction(pa0)
					index[1].applyPathAction(pa1)

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

							c.Expect(index[0].pathAction, IsNil)
							c.Expect(*index[1].pathAction, Equals, *pa1)
						})
					}
				})
			})
		})
	})
}
