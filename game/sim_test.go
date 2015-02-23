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

	phase := newNarrowPhase(index)
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

	phase := newNarrowPhase(index)
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

// NOTE This type is only suitable for a situation
//      where the actor ids are dependent like
//
//          0 -> 1 -> 2, and id 2 is standing still
//
//      It only creates collisions for [0,1] & [1,2]
//
//      The following will NOT work as expected
//
//          0 -> 1 <- 2 id 1 is standing still
//
//      because it won't create the right number
//      of collisions. This may require that in the
//      future a more generic version of all these
//      spec structures is created that programaticly
//      generates the collisions that engine would have
//      generated based on the bounding boxes.
type spec_2move_1stand struct {
	spec string

	// entity 0-1
	paths [2]coord.PathAction

	// entity 2
	cell   coord.Cell
	facing coord.Direction

	expectations func(spec_2move_1stand, actorIndex, gospec.Context)
}

func (t spec_2move_1stand) runSpec(c gospec.Context) {
	index := actorIndex{
		0: &actor{
			actorEntity: actorEntity{
				id:     0,
				cell:   t.paths[0].Orig,
				facing: t.paths[0].Direction(),
			},
		},

		1: &actor{
			actorEntity: actorEntity{
				id:     1,
				cell:   t.paths[1].Orig,
				facing: t.paths[1].Direction(),
			},
		},

		2: &actor{
			actorEntity: actorEntity{
				id:     2,
				cell:   t.cell,
				facing: t.facing,
			},
		},
	}

	index[0].applyPathAction(&t.paths[0])
	index[1].applyPathAction(&t.paths[1])

	var A, B, C int64 = 0, 1, 2

	ABBC := quad.CollisionGroup{}
	ABBC = ABBC.AddCollision(quad.Collision{
		index[A].Entity(),
		index[B].Entity(),
	})
	ABBC = ABBC.AddCollision(quad.Collision{
		index[B].Entity(),
		index[C].Entity(),
	})

	ABCB := quad.CollisionGroup{}
	ABCB = ABCB.AddCollision(quad.Collision{
		index[A].Entity(),
		index[B].Entity(),
	})
	ABCB = ABCB.AddCollision(quad.Collision{
		index[C].Entity(),
		index[B].Entity(),
	})

	CBAB := quad.CollisionGroup{}
	CBAB = CBAB.AddCollision(quad.Collision{
		index[C].Entity(),
		index[B].Entity(),
	})
	CBAB = CBAB.AddCollision(quad.Collision{
		index[A].Entity(),
		index[B].Entity(),
	})

	CBBA := quad.CollisionGroup{}
	CBBA = CBAB.AddCollision(quad.Collision{
		index[C].Entity(),
		index[B].Entity(),
	})
	CBBA = CBAB.AddCollision(quad.Collision{
		index[B].Entity(),
		index[A].Entity(),
	})

	phase := newNarrowPhase(index)
	testCases := []struct {
		spec string
		cgrp quad.CollisionGroup
	}{{
		"[0,1],[1,2]", ABBC,
	}, {
		"[0,1],[2,1]", ABCB,
	}, {
		"[2,1],[0,1]", CBAB,
	}, {
		"[2,1],[1,0]", CBBA,
	}}

	c.Specify(t.spec, func() {
		for _, testCase := range testCases {
			c.Specify(testCase.spec, func() {
				stillExisting, removed := phase.ResolveCollisions(&testCase.cgrp, 0)
				c.Assume(len(stillExisting), Equals, 3)
				c.Assume(len(removed), Equals, 0)

				t.expectations(t, index, c)
			})
		}
	})
}

type spec_3move struct {
	spec string

	// entity 0-2
	paths [3]coord.PathAction

	expectations func(spec_3move, actorIndex, gospec.Context)
}

func (t spec_3move) runSpec(c gospec.Context) {
	index := actorIndex{
		0: &actor{
			actorEntity: actorEntity{
				id:     0,
				cell:   t.paths[0].Orig,
				facing: t.paths[0].Direction(),
			},
		},

		1: &actor{
			actorEntity: actorEntity{
				id:     1,
				cell:   t.paths[1].Orig,
				facing: t.paths[1].Direction(),
			},
		},

		2: &actor{
			actorEntity: actorEntity{
				id:     2,
				cell:   t.paths[2].Orig,
				facing: t.paths[2].Direction(),
			},
		},
	}

	index[0].applyPathAction(&t.paths[0])
	index[1].applyPathAction(&t.paths[1])
	index[2].applyPathAction(&t.paths[2])

	var A, B, C int64 = 0, 1, 2

	ABBC := quad.CollisionGroup{}
	ABBC = ABBC.AddCollision(quad.Collision{
		index[A].Entity(),
		index[B].Entity(),
	})
	ABBC = ABBC.AddCollision(quad.Collision{
		index[B].Entity(),
		index[C].Entity(),
	})

	ABCB := quad.CollisionGroup{}
	ABCB = ABCB.AddCollision(quad.Collision{
		index[A].Entity(),
		index[B].Entity(),
	})
	ABCB = ABCB.AddCollision(quad.Collision{
		index[C].Entity(),
		index[B].Entity(),
	})

	CBAB := quad.CollisionGroup{}
	CBAB = CBAB.AddCollision(quad.Collision{
		index[C].Entity(),
		index[B].Entity(),
	})
	CBAB = CBAB.AddCollision(quad.Collision{
		index[A].Entity(),
		index[B].Entity(),
	})

	CBBA := quad.CollisionGroup{}
	CBBA = CBAB.AddCollision(quad.Collision{
		index[C].Entity(),
		index[B].Entity(),
	})
	CBBA = CBAB.AddCollision(quad.Collision{
		index[B].Entity(),
		index[A].Entity(),
	})

	phase := newNarrowPhase(index)
	testCases := []struct {
		spec string
		cgrp quad.CollisionGroup
	}{{
		"[0,1],[1,2]", ABBC,
	}, {
		"[0,1],[2,1]", ABCB,
	}, {
		"[2,1],[0,1]", CBAB,
	}, {
		"[2,1],[1,0]", CBBA,
	}}

	c.Specify(t.spec, func() {
		for _, testCase := range testCases {
			c.Specify(testCase.spec, func() {
				stillExisting, removed := phase.ResolveCollisions(&testCase.cgrp, 0)
				c.Assume(len(stillExisting), Equals, 3)
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

		c.Specify("3 actors", func() {
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
	})
}
