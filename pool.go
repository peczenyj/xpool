package xpool

import "sync"

var _ Pool[any] = (*sync.Pool)(nil)

// Pool is a type-safe object pool interface.
// for convenience, *sync.Pool is a Pool[any]
type Pool[T any] interface {
	// Get fetch one item from object pool
	// If needed, will create another object.
	Get() T

	// Put return the object to the pull.
	// It may reset the object before put it back to sync pool.
	Put(object T)
}

// New is the constructor of an xpool.Pool[T].
// Receives the constructor of the type T.
func New[T any](
	ctor func() T,
) Pool[T] {
	return &simplePool[T]{
		ctor: ctor,
		pool: new(Raw[T]),
	}
}

// Resetter interface.
type Resetter interface {
	// Reset may return the object to his initial state.
	Reset()
}

// OnResetCallback type.
// Will be called with a true value if the value T isa Resetter and was called with success.
type OnResetCallback func(called bool)

type pollConfig struct {
	onPutResets []OnResetCallback
}

// Option type.
type Option func(*pollConfig)

// WithOnPutResetCallback is a functional option.
// Includes one or more callbacks to be executed on object reset on Put method.
func WithOnPutResetCallback(onPutResets ...OnResetCallback) Option {
	return func(o *pollConfig) {
		o.onPutResets = append(o.onPutResets, onPutResets...)
	}
}

// NewWithDefaultResetter is an alternative constructor of an xpool.Pool[T].
// We will call the resetter callback before put the object back to the pool.
// Be careful, the custom resetter must be thread safe.
func NewWithResetter[T any](
	ctor func() T,
	onPutResetter func(T),
) Pool[T] {
	return &resettablePool[T]{
		onPutResetter: onPutResetter,
		pool:          New(ctor),
	}
}

// NewWithDefaultResetter is an alternative constructor of an xpool.Pool[T].
// If T is a Resetter, before put the object back to object pool we will call Reset().
func NewWithDefaultResetter[T any](
	ctor func() T,
	opts ...Option,
) Pool[T] {
	var c pollConfig

	for _, opt := range opts {
		opt(&c)
	}

	return NewWithResetter(ctor, func(object T) {
		defaultResetter, ok := any(object).(Resetter)
		if ok {
			defaultResetter.Reset()
		}

		for _, onReset := range c.onPutResets {
			onReset(ok)
		}
	})
}

type simplePool[T any] struct {
	ctor func() T
	pool *Raw[T]
}

func (p *simplePool[T]) Get() T {
	object, ok := p.pool.Get()
	if !ok {
		object = p.ctor()
	}

	return object
}

func (p *simplePool[T]) Put(object T) {
	p.pool.Put(object)
}

type resettablePool[T any] struct {
	onPutResetter func(T)
	pool          Pool[T]
}

func (p *resettablePool[T]) Get() T {
	return p.pool.Get()
}

func (p *resettablePool[T]) Put(object T) {
	p.onPutResetter(object)

	p.pool.Put(object)
}
