package resetter

import "github.com/peczenyj/xpool"

// ResetterMonadic interface.
type ResetterMonadic[T any] interface {
	Reset(t T)
}

type PoolResetterMonadic[T any, R ResetterMonadic[T]] struct {
	pool *xpool.Pool[R]
}

// New is the constructor of an *resetter.Pool.
// Receives the constructor of the type R that implements ResetterMonadic[T] interface.
func NewPoolMonadic[T any, R ResetterMonadic[T]](ctor func() R) *PoolResetterMonadic[T, R] {
	return &PoolResetterMonadic[T, R]{
		pool: xpool.New(ctor),
	}
}

// New is the constructor of an *resetter.Pool.
// Receives the constructor of the type R that implements ResetterMonadic[T] interface.
// Will call Reset(T) with the given argument of type T.
func (p *PoolResetterMonadic[T, R]) Get(t T) R {
	resetter := p.pool.Get()

	resetter.Reset(t)

	return resetter
}

// Put return the object to the pull
// Will call Reset(T) with a zero value of T on the object before send back to object pool.
func (p *PoolResetterMonadic[T, R]) Put(resetter R) {
	var zero T

	resetter.Reset(zero)

	p.pool.Put(resetter)
}
