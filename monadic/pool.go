// Package monadic supports objects whose state is set when they leave the pool
// and cleared when they return to it.
//
// Different than [xpool.Pool], the monadic [Pool] handles two different generic types: S and T
//   - T is the type of the object returned from the pool
//   - S is the state, set before an object is returned, and reset back to the zero value of S when put back to the pool.
//
// Note that a pooled object is reset twice per cycle: with the supplied state on Get, and with the
// zero value of S on Put. The Put-side reset is what keeps idle objects from pinning your state in
// memory (for example, a [bytes.Reader] holding a large slice), so it is intentional rather than redundant.
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
// We can use [New] to create a monadic [Pool] that manage the state of [bytes.Reader] via Reset method implicitly:
//
//	pool := monadic.New[[]byte](func() io.Reader { // you can use any interface that you want
//	  return bytes.NewReader(nil)
//	}
//
//	br := pool.Get(payload) // implicit Reset(payload) -- Get(state) is monadic, instead the niladic version on xpool.Pool
//	defer pool.Put(br)      // implicit Reset(nil)     -- the zero value of state S, []byte on this case
//
//	// use byte reader here as io.Reader
package monadic

import (
	"github.com/peczenyj/xpool"
)

// Pool monadic is a type-safe object pool interface.
// This interface is parameterized on two generic types:
//   - T is reserved for the type of the object that will be stored on the pool.
//   - S is reserved for the state of the object to be set before the object is returned from the pool.
type Pool[S, T any] interface {
	// Get fetch one item from object pool. If needed, will create another object.
	// The state S will be used in the resetter.
	Get(state S) T

	// Put returns the object to the pool.
	// A zero value of S will be used in the resetter.
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
//
// A monadic pool always resets; New is the [xpool.NewWithResetter] equivalent for monadic objects.
// Use [NewWithCustomResetter] when T does not implement [Resetter] or the reset needs extra logic.
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
// We can specify a special resetter: it is called with the supplied state on Get (before the object
// is returned to the caller) and with the zero value of S on Put (before the object is returned to the pool).
// The resetter may run concurrently from multiple goroutines, so any state it shares beyond the
// object being reset must be synchronized.
func NewWithCustomResetter[S, T any](
	ctor func() T,
	customResetter func(object T, state S),
) Pool[S, T] {
	return newWithResetters[S, T](
		ctor,
		customResetter,
		wrapResetToZeroValue(customResetter),
	)
}

// NewWithFallibleResetter is the constructor of a monadic [Pool] for objects whose reset can fail,
// such as a type implementing [flate.Resetter] (whose Reset returns an error).
//
// The resetter is called with the supplied state on Get and with the zero value of S on Put:
//   - If it fails on Get, the object is discarded, a fresh one is built via ctor and reset once more.
//     onError is invoked for each failure. Get always returns an object (best effort), so a reset
//     that keeps failing yields an object that was never successfully reset.
//   - If it fails on Put, the object is dropped instead of being pooled, and onError is invoked.
//
// onError is optional: if nil, failures are handled silently. The dropped/failed object is passed
// to onError so it can be inspected or closed.
// The resetter may run concurrently from multiple goroutines, so any state it shares beyond the
// object being reset must be synchronized.
// Will panic if resetter is nil.
func NewWithFallibleResetter[S, T any](
	ctor func() T,
	resetter func(object T, state S) error,
	onError func(err error, object T),
) Pool[S, T] {
	if resetter == nil {
		panic("callback 'resetter' must not be nil")
	}

	base := xpool.NewWithFallibleResetter[T](
		ctor,
		func(object T) error {
			var zero S

			return resetter(object, zero)
		},
		onError,
	)

	return &fallibleMonadicPool[S, T]{
		pool:    base,
		ctor:    ctor,
		onGet:   resetter,
		onError: onError,
	}
}

type fallibleMonadicPool[S, T any] struct {
	pool    xpool.Pool[T]
	ctor    func() T
	onGet   func(object T, state S) error
	onError func(err error, object T)
}

func (p *fallibleMonadicPool[S, T]) Get(state S) T {
	object := p.pool.Get()

	if err := p.onGet(object, state); err != nil {
		p.reportError(err, object)

		object = p.ctor() // the pooled object is suspect; start from a fresh one.

		if err := p.onGet(object, state); err != nil {
			p.reportError(err, object)
		}
	}

	return object
}

func (p *fallibleMonadicPool[_, T]) Put(object T) {
	p.pool.Put(object) // the underlying pool resets with the zero value and drops on failure.
}

func (p *fallibleMonadicPool[_, T]) reportError(err error, object T) {
	if p.onError != nil {
		p.onError(err, object)
	}
}

func wrapResetToZeroValue[S, T any](customResetter func(object T, state S)) func(object T) {
	return func(object T) {
		var zero S

		customResetter(object, zero)
	}
}

func newWithResetters[S, T any](
	ctor func() T,
	onGetResetter func(object T, state S),
	onPutResetter func(object T),
) Pool[S, T] {
	pool := xpool.NewWithCustomResetter[T](ctor, onPutResetter)

	return &resettableMonadicPool[S, T]{
		pool:          pool,
		onGetResetter: onGetResetter,
	}
}

type resettableMonadicPool[S, T any] struct {
	pool          xpool.Pool[T]
	onGetResetter func(object T, state S)
}

func (p *resettableMonadicPool[S, T]) Get(state S) T {
	object := p.pool.Get()

	p.onGetResetter(object, state)

	return object
}

func (p *resettableMonadicPool[_, T]) Put(object T) {
	p.pool.Put(object) // will call Reset with zero value
}
