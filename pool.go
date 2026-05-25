// Package xpool is a type-safe object pool built on top of [sync.Pool].
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
// For monadic objects, when we need to reset the object to an initial state before put it back to the pool,
// there are two alternative constructors:
//   - [NewWithResetter] verify if the type T is a [Resetter] and call Reset() method.
//   - [NewWithCustomResetter] allow add a generic callback func(T) to perform some more complex operations, if needed.
//
// Another alternative is to use https://github.com/peczenyj/xpool/monadic subpackage package.
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

	// Put returns the object to the pool.
	// It may reset the object before putting it back to the sync pool.
	Put(object T)
}

// Resetter interface.
type Resetter interface {
	// Reset may return the object to his initial state.
	Reset()
}

// New is the constructor of an [Pool] for a given generic type T.
// Receives the constructor of the type T.
func New[T any](
	ctor func() T,
) Pool[T] {
	return &simplePool[T]{
		pool: new(sync.Pool),
		ctor: ctor,
	}
}

// NewWithCustomResetter is an alternative constructor of an [Pool] for a given generic type T.
// We can specify a special resetter, called on Put just before the object is returned to the pool.
// The resetter may run concurrently from multiple goroutines, so any state it shares beyond the
// object being reset must be synchronized.
// Will panic if onPutResetter is nil.
func NewWithCustomResetter[T any](
	ctor func() T,
	onPutResetter func(T),
) Pool[T] {
	if onPutResetter == nil {
		panic("callback 'onPutResetter' must not be nil")
	}

	return &resettablePool[T]{
		pool:          New(ctor),
		onPutResetter: onPutResetter,
	}
}

// NewWithResetter is an alternative constructor of an [Pool] for a given generic type T.
// T must be a [Resetter], before put the object back to object pool we will call Reset().
func NewWithResetter[T Resetter](
	ctor func() T,
) Pool[T] {
	return NewWithCustomResetter(ctor, func(object T) {
		object.Reset()
	})
}

// NewWithFallibleResetter is an alternative constructor of an [Pool] for a given generic type T,
// for objects whose reset can fail (for example, a reset that returns an error).
//
// The resetter is called on Put, just before the object would be returned to the pool. If it
// returns a non-nil error, the object is dropped instead of being pooled (so the next Get builds a
// fresh one via ctor), and onError is invoked for observability. The dropped object is passed to
// onError so it can be inspected or closed.
//
// onError is optional: if nil, a failed reset silently drops the object.
// The resetter may run concurrently from multiple goroutines, so any state it shares beyond the
// object being reset must be synchronized.
// Will panic if resetter is nil.
func NewWithFallibleResetter[T any](
	ctor func() T,
	resetter func(object T) error,
	onError func(err error, object T),
) Pool[T] {
	if resetter == nil {
		panic("callback 'resetter' must not be nil")
	}

	return &fallibleResettablePool[T]{
		pool:     New(ctor),
		resetter: resetter,
		onError:  onError,
	}
}

type simplePool[T any] struct {
	pool *sync.Pool
	ctor func() T
}

func (p *simplePool[T]) Get() T {
	object, ok := p.pool.Get().(T)
	if !ok {
		object = p.ctor()
	}

	return object
}

func (p *simplePool[T]) Put(object T) {
	p.pool.Put(object)
}

type resettablePool[T any] struct {
	pool          Pool[T]
	onPutResetter func(T)
}

func (p *resettablePool[T]) Get() T {
	return p.pool.Get()
}

type fallibleResettablePool[T any] struct {
	pool     Pool[T]
	resetter func(object T) error
	onError  func(err error, object T)
}

func (p *fallibleResettablePool[T]) Get() T {
	return p.pool.Get()
}

func (p *fallibleResettablePool[T]) Put(object T) {
	if err := p.resetter(object); err != nil {
		if p.onError != nil {
			p.onError(err, object)
		}

		return // drop: a failed reset must not return the object to the pool.
	}

	p.pool.Put(object)
}

func (p *resettablePool[T]) Put(object T) {
	p.onPutResetter(object)

	p.pool.Put(object)
}
