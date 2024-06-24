package xpool

import "sync"

// Pool is a type-safe object pool build on top of sync.Pool
type Pool[T any] struct {
	ctor func() T
	pool *sync.Pool
}

// New is the constructor of an *xpool.Pool.
// Receives the constructor of the type T.
func New[T any](ctor func() T) *Pool[T] {
	return &Pool[T]{
		ctor: ctor,
		pool: new(sync.Pool),
	}
}

// Get fetch one item from object pool
// If needed, will create another object.
func (p *Pool[T]) Get() T {
	obj, ok := p.pool.Get().(T)
	if !ok {
		obj = p.ctor()
	}

	return obj
}

// Put return the object to the pull
func (p *Pool[T]) Put(obj T) {
	p.pool.Put(obj)
}
