package resetter

import "github.com/peczenyj/xpool"

// Resetter interface.
type Resetter interface {
	Reset()
}

// PoolResetter pool specialized type.
// Will call Reset on each object before send it back to object pool.
type PoolResetter[R Resetter] struct {
	pool *xpool.Pool[R]
}

// NewPool is the constructor of an *resetter.Pool.
// Receives the constructor of the type R that implements Resetter interface.
func NewPool[R Resetter](ctor func() R) *PoolResetter[R] {
	return &PoolResetter[R]{
		pool: xpool.New(ctor),
	}
}

// Get fetch one item from object pool
// If needed, will create another object.
func (p *PoolResetter[R]) Get() R {
	return p.pool.Get()
}

// Put return the object to the pull
// Will call Reset on the object before send back to object pool.
func (p *PoolResetter[R]) Put(resetter R) {
	resetter.Reset()

	p.pool.Put(resetter)
}
