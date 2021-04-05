package game_test

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/ghthor/aodd/game"
	"github.com/ghthor/filu/rpg2d"
	"github.com/ghthor/filu/rpg2d/entity"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

func DescribeGobConn(c gospec.Context) {
	c.Specify("a gob conn", func() {
		buf := bytes.NewBuffer(make([]byte, 0, 1024))
		gobconn := game.NewGobConn(buf)

		c.Specify("can send", func() {
			c.Specify("world states w/ entities", func() {
				worldState := rpg2d.WorldState{
					Entities: entity.StateSlice{
						game.ActorEntityState{Id: 2},
						game.SayEntityState{Id: 3},
						game.AssailEntityState{Id: 4},
					},
				}

				c.Expect(gobconn.EncodeAndSend(game.ET_WORLD_STATE, worldState), IsNil)

				eType, err := gobconn.ReadNextType()
				c.Assume(err, IsNil)
				c.Expect(eType, Equals, game.ET_WORLD_STATE)

				c.Specify("and can recv", func() {
					var decodedState rpg2d.WorldState
					c.Expect(gobconn.Decode(&decodedState), IsNil)
					c.Expect(decodedState.Entities[0], Equals, game.ActorEntityState{Id: 2})
					c.Expect(decodedState.Entities[1], Equals, game.SayEntityState{Id: 3})
					c.Expect(decodedState.Entities[2], Equals, game.AssailEntityState{Id: 4})
				})
			})

			c.Specify("world state diffs w/ entities", func() {
			})
		})
	})
}

func TestGobDecodingSlices(t *testing.T) {
	ids := make([]entity.Id, 0, 20)
	ids = append(ids, 0, 1, 4)

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(ids)
	if err != nil {
		t.Fatalf("gob encoding error: %v", err)
	}

	ids = make([]entity.Id, 0, 20)

	dec := gob.NewDecoder(&buf)
	err = dec.Decode(&ids)
	if err != nil {
		t.Fatalf("gob decoding error: %v", err)
	}

	if len(ids) != 3 {
		t.Errorf("len(ids) != 3, %#v", ids)
	}

	err = enc.Encode(ids)
	if err != nil {
		t.Fatalf("gob encoding error: %v", err)
	}

	ids = nil
	err = dec.Decode(&ids)
	if err != nil {
		t.Fatalf("gob decoding error: %v", err)
	}

	if len(ids) != 3 {
		t.Errorf("len(ids) != 3, %#v", ids)
	}

	if ids[2] != 4 {
		t.Fatal(ids)
	}
}
