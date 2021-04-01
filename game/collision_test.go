package game

import (
	"fmt"

	"github.com/ghthor/filu/rpg2d"
	"github.com/ghthor/filu/rpg2d/coord"
	"github.com/ghthor/filu/rpg2d/entity"
	"github.com/ghthor/filu/rpg2d/quad"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

type spec_2moving struct {
	spec         string
	paths        []coord.PathAction
	expectations func(spec_2moving, ActorIndex, gospec.Context)
}

func (t spec_2moving) runSpec(c gospec.Context) {
	pa0 := t.paths[0]
	pa1 := t.paths[1]

	index := ActorIndex{
		0: &actor{
			id: 0,
			actorEntity: actorEntity{
				id:      0,
				actorId: 0,

				cell:   pa0.Orig,
				facing: pa0.Direction(),
			},
		},

		1: &actor{
			id: 1,
			actorEntity: actorEntity{
				id:      1,
				actorId: 1,

				cell:   pa1.Orig,
				facing: pa1.Direction(),
			},
		},
	}

	index[0].applyPathAction(&pa0)
	index[1].applyPathAction(&pa1)

	cg := func(a, b rpg2d.ActorId) *quad.CollisionGroup {
		cg := quad.NewCollisionGroup(1)
		cg.AddCollision(index[a].Entity(), index[b].Entity())
		c.Assume(len(cg.Entities), Equals, 2)
		c.Assume(len(cg.CollisionsById), Equals, 1)
		return cg
	}

	phase := newNarrowPhase(index)
	testCases := []struct {
		spec string
		cgrp *quad.CollisionGroup
	}{
		{"AB", cg(0, 1)},
		{"BA", cg(1, 0)},
	}

	for _, testCase := range testCases {
		c.Specify(testCase.spec, func() {
			result := phase.ResolveCollisions([]*quad.CollisionGroup{testCase.cgrp}, 0)
			c.Assume(len(result.Updated()), Equals, 2)

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

	expectations func(spec_1move_1stand, ActorIndex, gospec.Context)
}

func (t spec_1move_1stand) runSpec(c gospec.Context) {
	index := ActorIndex{
		0: &actor{
			id: 0,
			actorEntity: actorEntity{
				id:      0,
				actorId: 0,
				cell:    t.path.Orig,
				facing:  t.path.Direction(),
			},
		},

		1: &actor{
			id: 1,
			actorEntity: actorEntity{
				id:      1,
				actorId: 1,
				cell:    t.cell,
				facing:  t.facing,
			},
		},
	}

	index[0].applyPathAction(&t.path)

	cg := func(a, b rpg2d.ActorId) *quad.CollisionGroup {
		cg := quad.NewCollisionGroup(1)
		cg.AddCollision(index[a].Entity(), index[b].Entity())
		return cg
	}

	phase := newNarrowPhase(index)
	testCases := []struct {
		spec string
		cgrp *quad.CollisionGroup
	}{
		{"AB", cg(0, 1)},
		{"BA", cg(1, 0)},
	}

	c.Specify(t.spec, func() {
		for _, testCase := range testCases {
			c.Specify(testCase.spec, func() {
				result := phase.ResolveCollisions([]*quad.CollisionGroup{testCase.cgrp}, 0)
				c.Assume(len(result.Updated()), Equals, 2)

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

	expectations func(spec_2move_1stand, ActorIndex, gospec.Context)
}

func (t spec_2move_1stand) runSpec(c gospec.Context) {
	index := ActorIndex{
		0: &actor{
			id: 0,
			actorEntity: actorEntity{
				id:      0,
				actorId: 0,
				cell:    t.paths[0].Orig,
				facing:  t.paths[0].Direction(),
			},
		},

		1: &actor{
			id: 1,
			actorEntity: actorEntity{
				id:      1,
				actorId: 1,
				cell:    t.paths[1].Orig,
				facing:  t.paths[1].Direction(),
			},
		},

		2: &actor{
			id: 2,
			actorEntity: actorEntity{
				id:      2,
				actorId: 2,
				cell:    t.cell,
				facing:  t.facing,
			},
		},
	}

	index[0].applyPathAction(&t.paths[0])
	index[1].applyPathAction(&t.paths[1])

	checkSetup := func(cg *quad.CollisionGroup) {
		c.Assume(len(cg.Entities), Equals, 3)
		c.Assume(len(cg.CollisionsById), Equals, 2)
	}

	var A, B, C rpg2d.ActorId = 0, 1, 2

	ABBC := quad.NewCollisionGroup(2)
	ABBC.AddCollision(
		index[A].Entity(),
		index[B].Entity(),
	)
	ABBC.AddCollision(
		index[B].Entity(),
		index[C].Entity(),
	)
	checkSetup(ABBC)

	ABCB := quad.NewCollisionGroup(2)
	ABCB.AddCollision(
		index[A].Entity(),
		index[B].Entity(),
	)
	ABCB.AddCollision(
		index[C].Entity(),
		index[B].Entity(),
	)
	checkSetup(ABCB)

	CBAB := quad.NewCollisionGroup(2)
	CBAB.AddCollision(
		index[C].Entity(),
		index[B].Entity(),
	)
	CBAB.AddCollision(
		index[A].Entity(),
		index[B].Entity(),
	)
	checkSetup(CBAB)

	CBBA := quad.NewCollisionGroup(2)
	CBBA.AddCollision(
		index[C].Entity(),
		index[B].Entity(),
	)
	CBBA.AddCollision(
		index[B].Entity(),
		index[A].Entity(),
	)
	checkSetup(CBBA)

	phase := newNarrowPhase(index)
	testCases := []struct {
		spec string
		cgrp *quad.CollisionGroup
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
				result := phase.ResolveCollisions([]*quad.CollisionGroup{testCase.cgrp}, 0)
				c.Assume(len(result.Updated()), Equals, 3)

				t.expectations(t, index, c)
			})
		}
	})
}

type spec_3move struct {
	spec string

	// entity 0-2
	paths [3]coord.PathAction

	expectations func(spec_3move, ActorIndex, gospec.Context)
}

func (t spec_3move) runSpec(c gospec.Context) {
	index := ActorIndex{
		0: &actor{
			id: 0,
			actorEntity: actorEntity{
				id:      0,
				actorId: 0,
				cell:    t.paths[0].Orig,
				facing:  t.paths[0].Direction(),
			},
		},

		1: &actor{
			id: 1,
			actorEntity: actorEntity{
				id:      1,
				actorId: 1,
				cell:    t.paths[1].Orig,
				facing:  t.paths[1].Direction(),
			},
		},

		2: &actor{
			id: 2,
			actorEntity: actorEntity{
				id:      2,
				actorId: 2,
				cell:    t.paths[2].Orig,
				facing:  t.paths[2].Direction(),
			},
		},
	}

	index[0].applyPathAction(&t.paths[0])
	index[1].applyPathAction(&t.paths[1])
	index[2].applyPathAction(&t.paths[2])

	var A, B, C rpg2d.ActorId = 0, 1, 2

	checkSetup := func(cg *quad.CollisionGroup) {
		c.Assume(len(cg.Entities), Equals, 3)
		c.Assume(len(cg.CollisionsById), Equals, 2)
	}

	ABBC := quad.NewCollisionGroup(2)
	ABBC.AddCollision(
		index[A].Entity(),
		index[B].Entity(),
	)
	ABBC.AddCollision(
		index[B].Entity(),
		index[C].Entity(),
	)
	checkSetup(ABBC)

	ABCB := quad.NewCollisionGroup(2)
	ABCB.AddCollision(
		index[A].Entity(),
		index[B].Entity(),
	)
	ABCB.AddCollision(
		index[C].Entity(),
		index[B].Entity(),
	)
	checkSetup(ABCB)

	CBAB := quad.NewCollisionGroup(2)
	CBAB.AddCollision(
		index[C].Entity(),
		index[B].Entity(),
	)
	CBAB.AddCollision(
		index[A].Entity(),
		index[B].Entity(),
	)
	checkSetup(CBAB)

	CBBA := quad.NewCollisionGroup(2)
	CBBA.AddCollision(
		index[C].Entity(),
		index[B].Entity(),
	)
	CBBA.AddCollision(
		index[B].Entity(),
		index[A].Entity(),
	)
	checkSetup(CBBA)

	phase := newNarrowPhase(index)
	testCases := []struct {
		spec string
		cgrp *quad.CollisionGroup
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
				result := phase.ResolveCollisions([]*quad.CollisionGroup{testCase.cgrp}, 0)
				c.Assume(len(result.Updated()), Equals, 3)

				t.expectations(t, index, c)
			})
		}
	})
}

type spec_allMoving struct {
	spec string

	paths []coord.PathAction

	expectations func(spec_allMoving, ActorIndex, gospec.Context)
}

type testCase struct {
	spec string
	cgrp *quad.CollisionGroup
}

func findCollisions(index ActorIndex) []quad.Collision {
	var collisions []quad.Collision

	for id1 := 0; id1 < len(index); id1++ {
		a1 := index[rpg2d.ActorId(id1)]
		for id2 := 0; id2 < len(index); id2++ {
			a2 := index[rpg2d.ActorId(id2)]

			if a1.Id() == a2.Id() {
				continue
			}

			if a1.Bounds().Overlaps(a2.Bounds()) {
				collisions = append(collisions, quad.NewCollision(a1.Entity(), a2.Entity()))
			}
		}
	}

	return collisions
}

// NOTE Only generates a single case
func generateCases(index ActorIndex) []testCase {
	testCases := make([]testCase, 0, len(index))

	// Assume that each index is only colliding with the
	// one in front of it and behind it.
	var spec string
	collisions := findCollisions(index)
	cg := quad.NewCollisionGroup(len(collisions))

	for _, c := range collisions {
		cg.AddCollision(c.A, c.B)
	}

	i := 0
	for _, c := range cg.CollisionsById {
		if i == 0 {
			spec = fmt.Sprintf("[%d, %d]", c.A.Id(), c.B.Id())
		} else {
			spec = fmt.Sprintf("%s,[%d, %d]", spec, c.A.Id(), c.B.Id())
		}
		i++
	}

	testCases = append(testCases, testCase{spec, cg})

	return testCases
}

func (t spec_allMoving) runSpec(c gospec.Context) {
	index := make(ActorIndex, len(t.paths))
	for i, p := range t.paths {
		index[rpg2d.ActorId(i)] = &actor{
			id: rpg2d.ActorId(i),

			actorEntity: actorEntity{
				id:      entity.Id(i),
				actorId: rpg2d.ActorId(i),
				cell:    p.Orig,
				facing:  p.Direction(),
			},
		}

		index[rpg2d.ActorId(i)].applyPathAction(&t.paths[i])
	}

	phase := newNarrowPhase(index)
	testCases := generateCases(index)

	c.Specify(t.spec, func() {
		for _, testCase := range testCases {
			c.Specify(testCase.spec, func() {
				result := phase.ResolveCollisions([]*quad.CollisionGroup{testCase.cgrp}, 0)
				c.Assume(len(result.Updated()), Equals, len(t.paths))

				t.expectations(t, index, c)
			})
		}
	})
}
