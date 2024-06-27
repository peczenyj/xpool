package monadic

import (
	"github.com/peczenyj/xpool"
)

// Pool monadic is a type-safe object pool interface.
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

// OnResetCallback type.
// Will be called on each Reset with two parameters:
//   - called: will be true if T isa Resetter[S]
//   - onGet: will be true if the Reset is on Get(state S), else it is on Put(object T)
type OnResetCallback func(called, onGet bool)

// New is the constructor of an *xpool.Pool.
// Receives the constructor of the type T.
// It sets a trivial resetter, that try to convert T to Resetter[S]
// will call Reset(state S) before return the object on Get(state S)
// will call Reset(zero value of S) before push back to the pool.
// It accepts optional callbacks to be executed after a Reset.
func New[S, T any](
	ctor func() T,
	onResets ...OnResetCallback,
) Pool[S, T] {
	onGetResetter := buildDefaultResetter[S, T](true, onResets...)
	onPutResetter := buildDefaultResetter[S, T](false, onResets...)

	return newWithResetters[S, T](
		ctor,
		onGetResetter,
		func(object T) {
			var zero S
			onPutResetter(object, zero)
		},
	)
}

func buildDefaultResetter[S, T any](
	onGet bool,
	onResets ...OnResetCallback,
) func(object T, state S) {
	return func(object T, state S) {
		defaultResetter, ok := any(object).(Resetter[S])
		if ok {
			defaultResetter.Reset(state)
		}

		for _, onReset := range onResets {
			onReset(ok, onGet)
		}
	}
}

// New is the constructor of an *xpool.Pool.
// Receives the constructor of the type T.
// It allow to specify a special resetter, to be called before return the object from the pool.
// The resetter will be called with zero value of S before push back to the pool.
// Be careful, the custom resetter must be thread safe.
func NewWithResetter[S, T any](
	ctor func() T,
	resetter func(object T, state S),
) Pool[S, T] {
	onPutResetter := func(object T) {
		var zero S

		resetter(object, zero)
	}

	return newWithResetters(
		ctor,
		resetter,
		onPutResetter,
	)
}

func newWithResetters[S, T any](
	ctor func() T,
	onGetResetter func(object T, state S),
	onPutResetter func(object T),
) Pool[S, T] {
	pool := xpool.NewWithResetter[T](ctor, onPutResetter)

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
