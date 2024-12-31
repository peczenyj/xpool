// The intent of stateful is to support stateful objects.
//
// Different than [xpool.Pool], the monadic [Pool] handle two different generic types: S and T
//   - T is the type of the object returned from the pool
//   - S is the state, where we set before return an object, and reset it back to zero value of S when put back to the pool.
//
// In other words, instead having to do:
//
//	pool := xpool.New(func() *bytes.Reader { // you must use a type or interface that exposes a Reset() method
//	  return bytes.NewReader(nil)
//	})
//
//	br := pool.Get()
//	defer func() { br.Reset(nil) ; pool.Put(br) }()
//
//	br.Reset(payload)
//	// use the byte reader here
//
// We can use [New] to create a monadic [Pool] that manage the state of [bytes.Reader] via Reset method implicity:
//
//	pool := stateful.New[[]byte](func() io.Reader { // you can use any interface that you want
//	  return bytes.NewReader(nil)
//	}
//
//	br := pool.Get(payload) // implicit Reset(payload) -- Get(state) is monadic, instead the niladic version on xpool.Pool
//	defer pool.Put(br)      // implicit Reset(nil)     -- the zero value of state S, []byte on this case
//
//	// use byte reader here as io.Reader
package stateful

import (
	"github.com/peczenyj/xpool"
)

// Pool monadic is a type-safe object pool interface.
// This interface is parameterized on two generic types:
//   - T is reserved for the type of the object that will be stored on the pool.
//   - S is reserved for the status of the object to be setted before return the object from the pool.
type Pool[S, T any] interface {
	// Get fetch one item from object pool. If needed, will create another object.
	// The state S will be used in the resetter.
	Get(state S) T

	// Put return the object to the pull.
	// A zero value of T will be used in the resetter.
	Put(object T)
}

// Resetter monadic interface.
type Resetter[S any] interface {
	Reset(state S)
}

// New is the constructor of an [Pool] for a given set of generic types S and T.
// Receives the constructor of the type T.
// It sets a trivial resetter, T must be a [Resetter]
// will call Reset(state S) before return the object on Get(state S)
// will call Reset(zero value of S) before push back to the pool.
func New[S any, T Resetter[S]](
	ctor func() T,
) Pool[S, T] {
	return newWithResetters[S, T](
		ctor,
		func(object T, state S) {
			object.Reset(state)
		},
		func(object T) {
			var zero S

			object.Reset(zero)
		},
	)
}

// NewWithCustomResetter is the constructor of an [Pool] for a given set of generic types S and T.
// Receives the constructor of the type T as a callback.
// We can specify a special resetter, to be called with a zero value of S before
// return the object from the pool.
// Be careful, the custom resetter must be thread safe.
func NewWithCustomResetter[S, T any](
	ctor func() T,
	customResetter func(object T, state S),
) Pool[S, T] {
	return newWithResetters[S, T](
		ctor,
		customResetter,
		func(object T) {
			var zero S

			customResetter(object, zero)
		},
	)
}

func newWithResetters[S, T any](
	ctor func() T,
	onGetResetter func(object T, state S),
	onPutResetter func(object T),
) Pool[S, T] {
	pool := xpool.NewWithCustomResetter[T](ctor, onPutResetter)

	return &resettableMonadicPool[S, T]{
		pool:     pool,
		resetter: onGetResetter,
	}
}

type resettableMonadicPool[S, T any] struct {
	pool     xpool.Pool[T]
	resetter func(object T, state S)
}

func (p *resettableMonadicPool[S, T]) Get(state S) T {
	object := p.pool.Get()

	p.resetter(object, state)

	return object
}

func (p *resettableMonadicPool[_, T]) Put(object T) {
	p.pool.Put(object) // will call Reset with zero value
}
