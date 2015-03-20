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

func TestCanBeConnected(t *testing.T) {
	pool := newActorPool(1)
	pool.AddActor("testing", "testingpasswd")

	actor, exists := pool.ActorExists("testing")
	if !exists {
		t.Fail()
	}

	if !actor.CanBeConnected() {
		t.Fail()
	}

	<-actor.IsConnected
	actor.IsConnected <- true

	actor, exists = pool.ActorExists("testing")
	if !exists {
		t.Fail()
	}

	if actor.CanBeConnected() {
		t.Fail()
	}
}

func TestActorShouldHaveAUniqueID(t *testing.T) {
	pool := newActorPool(2)
	a1, err := pool.AddActor("actor1", "password")
	if err != nil {
		t.Fail()
	}

	a2, err := pool.AddActor("actor2", "password")
	if err != nil {
		t.Fail()
	}

	if a1.Id == a2.Id {
		t.Fail()
	}
}
