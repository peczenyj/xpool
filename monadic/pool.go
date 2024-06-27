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
// Will be called with a true value if the value T isa Resetter and was called with success.
type OnResetCallback func(called bool)

type pollConfig struct {
	onGetResets []OnResetCallback
	onPutResets []OnResetCallback
}

// Option type.
type Option func(*pollConfig)

// WithOnGetResetCallback is a functional option.
// Includes one or more callbacks to be executed on object reset on Get method.
func WithOnGetResetCallback(onGetResets ...OnResetCallback) Option {
	return func(o *pollConfig) {
		o.onGetResets = append(o.onGetResets, onGetResets...)
	}
}

// WithOnPutResetCallback is a functional option.
// Includes one or more callbacks to be executed on object reset on Put method.
func WithOnPutResetCallback(onPutResets ...OnResetCallback) Option {
	return func(o *pollConfig) {
		o.onPutResets = append(o.onPutResets, onPutResets...)
	}
}

// New is the constructor of an *xpool.Pool.
// Receives the constructor of the type T.
// It sets a trivial resetter, that try to convert T to Resetter[S]
// will call Reset(state S) before return the object on Get(state S)
// will call Reset(zero value of S) before push back to the pool.
func New[S, T any](
	ctor func() T,
	opts ...Option,
) Pool[S, T] {
	var c pollConfig

	for _, opt := range opts {
		opt(&c)
	}

	onGetResetter := buildDefaultResetter[S, T](c.onGetResets...)
	onPutResetter := buildZeroResetter[S, T](c.onPutResets...)

	return newWithResetters[S, T](
		ctor,
		onGetResetter,
		onPutResetter,
	)
}

func buildDefaultResetter[S, T any](
	onResets ...OnResetCallback,
) func(object T, state S) {
	return func(object T, state S) {
		defaultResetter, ok := any(object).(Resetter[S])
		if ok {
			defaultResetter.Reset(state)
		}

		for _, onReset := range onResets {
			onReset(ok)
		}
	}
}

func buildZeroResetter[S, T any](
	onResets ...OnResetCallback,
) func(object T) {
	defaultResetter := buildDefaultResetter[S, T](onResets...)
	return func(object T) {
		var zero S

		defaultResetter(object, zero)
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
