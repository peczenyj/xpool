package monadic

import (
	"github.com/peczenyj/xpool"
)

// Pool monadic is a type-safe object pool interface.
type Pool[V, T any] interface {
	// Get fetch one item from object pool. If needed, will create another object.
	// The value V will be used in the resetter.
	Get(value V) T

	// Put return the object to the pull.
	// A zero value of T will be used in the resetter.
	Put(object T)
}

// Resetter interface.
type Resetter[V any] interface {
	Reset(value V)
}

type monadicPool[V, T any] struct {
	pool     xpool.Pool[T]
	resetter func(v V, t T)
}

// New is the constructor of an *xpool.Pool.
// Receives the constructor of the type T.
// It sets a trivial resetter, that try to convert T to Resetter[V]
// and call Reset(V) before return the object on Get(V)
// and call Reset(zero value of V) before push back to the pool.
func New[V, T any](ctor func() T) Pool[V, T] {
	return NewWithResetter(ctor, func(v V, t T) {
		if defaultResetter, ok := any(t).(Resetter[V]); ok {
			defaultResetter.Reset(v)
		}
	})
}

// New is the constructor of an *xpool.Pool.
// Receives the constructor of the type T.
// It allow to specify a special resetter, to be called before return the object from the pool.
// The resetter will be called with zero value of V before push back to the pool.
func NewWithResetter[V, T any](ctor func() T, resetter func(v V, t T)) Pool[V, T] {
	return &monadicPool[V, T]{
		pool:     xpool.New[T](ctor),
		resetter: resetter,
	}
}

func (p *monadicPool[V, T]) Get(value V) T {
	object := p.pool.Get()

	p.resetter(value, object)

	return object
}

func (p *monadicPool[V, T]) Put(object T) {
	var zero V

	p.resetter(zero, object)

	p.pool.Put(object)
}
