package game

import (
	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

func DescribeActorState(c gospec.Context) {
	c.Specify("an actor's state", func() {
		c.Specify("is different if its", func() {
			c.Specify("name has changed", func() {
				actor := ActorEntityState{
					Name: "actor name",
				}

				c.Expect(actor.IsDifferentFrom(actor), IsFalse)
				c.Expect(actor.IsDifferentFrom(ActorEntityState{
					Name: "actor name changed",
				}), IsTrue)
			})

			c.Specify("position has changed", func() {
				actor := ActorEntityState{
					Cell: coord.Cell{0, 0},
				}

				c.Expect(actor.IsDifferentFrom(actor), IsFalse)
				c.Expect(actor.IsDifferentFrom(ActorEntityState{
					Cell: coord.Cell{1, 0},
				}), IsTrue)
			})

			c.Specify("facing has changed", func() {
				actor := ActorEntityState{
					Facing: coord.North,
				}

				c.Expect(actor.IsDifferentFrom(actor), IsFalse)
				c.Expect(actor.IsDifferentFrom(ActorEntityState{
					Facing: coord.South,
				}), IsTrue)
			})

			c.Specify("health has changed", func() {
				actor := ActorEntityState{
					Hp:    100,
					HpMax: 100,
				}

				c.Expect(actor.IsDifferentFrom(actor), IsFalse)

				c.Expect(actor.IsDifferentFrom(ActorEntityState{
					Hp:    90,
					HpMax: 100,
				}), IsTrue)

				c.Expect(actor.IsDifferentFrom(ActorEntityState{
					Hp:    100,
					HpMax: 110,
				}), IsTrue)
			})

			c.Specify("mana has changed", func() {
				actor := ActorEntityState{
					Mp:    100,
					MpMax: 100,
				}

				c.Expect(actor.IsDifferentFrom(actor), IsFalse)

				c.Expect(actor.IsDifferentFrom(ActorEntityState{
					Mp:    90,
					MpMax: 100,
				}), IsTrue)

				c.Expect(actor.IsDifferentFrom(ActorEntityState{
					Mp:    100,
					MpMax: 110,
				}), IsTrue)
			})
		})
	})
}
