// xpool is a type safe object pool build on top of [sync.Pool]
//
// It is easy to use, just give a function that can create an object of a given type T
// as argument of [New] and it will return an implementation of [Pool] interface:
//
//	pool := xpool.New(func() io.ReadWriter {
//	  return new(bytes.Buffer)
//	})
//	rw := pool.Get()
//	defer pool.Put(rw)
//
// For stateful objects, when we need to reset the object to an initial state before put it back to the pool,
// there are two alternative constructors:
//   - [NewWithDefaultResetter] verify if the type T is a [Resetter] and call Reset() method.
//   - [NewWithResetter] allow add a generic callback func(T) to perform some more complex operations, if needed.
//
// Another alternative is to use https://github.com/peczenyj/xpool/monadicpool subpackage package.
package xpool

import "sync"

var _ Pool[any] = (*sync.Pool)(nil)

// Pool is a type-safe object pool interface.
// This interface is parameterized on one generic types:
//   - T is reserved for the type of the object that will be stored on the pool.
//
// For convenience, a pointer to sync.Pool is a Pool[any]
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

// OnResetCallback type.
// Will be called with a true value if the value T isa [Resetter] and was called with success.
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

// New is the constructor of an [Pool] for a given generic type T.
// Receives the constructor of the type T.
func New[T any](
	ctor func() T,
) Pool[T] {
	return NewWithResetter[T](ctor, nil)
}

// NewWithDefaultResetter is an alternative constructor of an [Pool] for a given generic type T.
// We can specify a special resetter, to be called before return the object from the pool.
// Be careful, the custom resetter must be thread safe.
func NewWithResetter[T any](
	ctor func() T,
	resetter func(T),
) Pool[T] {
	return &simplePool[T]{
		pool:     new(sync.Pool),
		ctor:     ctor,
		resetter: resetter,
	}
}

// NewWithDefaultResetter is an alternative constructor of an [Pool] for a given generic type T.
// If T is a [Resetter], before put the object back to object pool we will call Reset().
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
	pool     Pool[any]
	ctor     func() T
	resetter func(T)
}

func (p *simplePool[T]) Get() T {
	object, ok := p.pool.Get().(T)
	if !ok {
		object = p.ctor()
	}

	return object
}

func (p *simplePool[T]) Put(object T) {
	if p.resetter != nil {
		p.resetter(object)
	}

	p.pool.Put(object)
}
