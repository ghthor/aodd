package datastore

import (
	"testing"
)

func TestAddActorShouldAddActorToActorPool(t *testing.T) {
	pool := newActorPool(1)
	pool.AddActor("testing", "testingpasswd")

	if len(pool.store) != 1 {
		t.Fail()
	}
}

func TestAddActorShouldFailIfActorExists(t *testing.T) {
	pool := newActorPool(1)
	pool.AddActor("testing", "testingpasswd")

	_, err := pool.AddActor("testing", "something")
	if err != ErrActorExists {
		t.Fail()
	}
}
