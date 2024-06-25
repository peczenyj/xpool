package xpool

import "sync"

// Pool is a type-safe object pool interface.
type Pool[T any] interface {
	// Get fetch one item from object pool
	// If needed, will create another object.
	Get() T

	// Put return the object to the pull.
	// It may reset the object before put it back to sync pool.
	Put(object T)
}

// Resetter interface.
type Resetter interface {
	// Reset may return the object to his initial state.
	Reset()
}

// New is the constructor of an xpool.Pool.
// Receives the constructor of the type T.
func New[T any](ctor func() T) Pool[T] {
	return &simplePool[T]{
		ctor: ctor,
		pool: new(sync.Pool),
	}
}

// NewWithDefaultResetter is an alternative constructor of an xpool.Pool.
// We will call the resetter callback before put the object back to the pool.
// Be careful, the custom resetter must be thread safe.
func NewWithResetter[T any](ctor func() T, resetter func(T)) Pool[T] {
	return &resettablePool[T]{
		pool:     New(ctor),
		resetter: resetter,
	}
}

// NewWithDefaultResetter is an alternative constructor of an xpool.Pool.
// If T is a Resetter, before put the object back to object pool we will call Reset().
func NewWithDefaultResetter[T any](ctor func() T) Pool[T] {
	return NewWithResetter(ctor, func(t T) {
		if defaultResetter, ok := any(t).(Resetter); ok {
			defaultResetter.Reset()
		}
	})
}

type simplePool[T any] struct {
	ctor func() T
	pool *sync.Pool
}

func (p *simplePool[T]) Get() T {
	obj, ok := p.pool.Get().(T)
	if !ok {
		obj = p.ctor()
	}

	return obj
}

func (p *simplePool[T]) Put(obj T) {
	p.pool.Put(obj)
}

type resettablePool[T any] struct {
	pool     Pool[T]
	resetter func(T)
}

func (p *resettablePool[T]) Get() T {
	return p.pool.Get()
}

func (p *resettablePool[T]) Put(obj T) {
	p.resetter(obj)

	p.pool.Put(obj)
}
