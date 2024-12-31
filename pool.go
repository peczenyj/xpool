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

	// Put return the object to the pull.
	// It may reset the object before put it back to sync pool.
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
	return NewWithCustomResetter[T](ctor, nil)
}

// NewWithDefaultResetter is an alternative constructor of an [Pool] for a given generic type T.
// We can specify a special resetter, to be called before return the object from the pool.
// Be careful, the custom resetter must be thread safe.
func NewWithCustomResetter[T any](
	ctor func() T,
	onPutResetter func(T),
) Pool[T] {
	return &simplePool[T]{
		pool:          new(sync.Pool),
		ctor:          ctor,
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

type simplePool[T any] struct {
	pool          Pool[any]
	ctor          func() T
	onPutResetter func(T)
}

func (p *simplePool[T]) Get() T {
	object, ok := p.pool.Get().(T)
	if !ok {
		object = p.ctor()
	}

	return object
}

func (p *simplePool[T]) Put(object T) {
	if p.onPutResetter != nil {
		p.onPutResetter(object)
	}

	p.pool.Put(object)
}
