package datastore

import (
	"errors"
	"sync"

	"github.com/ghthor/engine/rpg2d/coord"
)

type Actor struct {
	Name, password string

	Id uint

	Loc    coord.Cell
	Facing coord.Direction
}

func (a Actor) Authenticate(name, password string) bool {
	return a.Name == name && a.password == password
}

type actorPool struct {
	nextId uint
	store  map[string]Actor
	lock   sync.Mutex
}

func newActorPool(size int) actorPool {
	return actorPool{
		store: make(map[string]Actor, size),
	}
}

func (p actorPool) ActorExists(name string) (Actor, bool) {
	p.lock.Lock()
	defer p.lock.Unlock()
	a, exists := p.store[name]
	return a, exists
}

var ErrActorExists = errors.New("actor already exists")

var defaultSpawn = coord.Cell{0, 0}

func (p actorPool) AddActor(name, password string) (Actor, error) {
	actor, actorExists := p.ActorExists(name)
	if actorExists {
		return actor, ErrActorExists
	}

	actor = Actor{
		Name:     name,
		password: password,

		Id: p.nextId,

		Loc:    defaultSpawn,
		Facing: coord.South,
	}

	p.lock.Lock()
	p.store[name] = actor
	p.lock.Unlock()
	return actor, nil
}

// Datastore interface
type Datastore interface {
	ActorExists(name string) (Actor, bool)
	AddActor(name, password string) (Actor, error)
}

type memDb struct {
	actorPool
}

func NewMemDatastore() Datastore {
	return &memDb{
		actorPool: newActorPool(10),
	}
}
