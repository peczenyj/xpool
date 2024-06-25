package xpool

import "sync"

// Pool is a type-safe object pool interface.
type Pool[T any] interface {
	// Get fetch one item from object pool
	// If needed, will create another object.
	Get() T

	// Put return the object to the pull.
	Put(object T)
}

// Resetter interface.
type Resetter interface {
	// Reset may return the object to his initial state.
	Reset()
}

type niladicPool[T any] struct {
	ctor     func() T
	pool     *sync.Pool
	resetter func(T)
}

// New is the constructor of an xpool.Pool.
// Receives the constructor of the type T.
func New[T any](ctor func() T) Pool[T] {
	return NewWithResetter(ctor, nil)
}

// NewWithDefaultResetter is an alternative constructor of an xpool.Pool.
// We will call the resetter callback before put the object back to the pool.
func NewWithResetter[T any](ctor func() T, resetter func(T)) Pool[T] {
	return &niladicPool[T]{
		ctor:     ctor,
		pool:     new(sync.Pool),
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

func (p *niladicPool[T]) Get() T {
	obj, ok := p.pool.Get().(T)
	if !ok {
		obj = p.ctor()
	}

	return obj
}

func (p *niladicPool[T]) Put(obj T) {
	if p.resetter != nil {
		p.resetter(obj)
	}

	p.pool.Put(obj)
}
