package xpool

import "sync"

// Raw is the most basic type safe pool.
type Raw[T any] struct {
	pool sync.Pool
}

// Get try to return an object of type T from object pool.
// If it can't, will return a zero value of T and ok fill be false.
// It is caller responsibility to create an object T them.
func (r *Raw[T]) Get() (object T, ok bool) {
	object, ok = r.pool.Get().(T)

	return object, ok
}

// Put will add the object back to the pool.
func (r *Raw[T]) Put(object T) {
	r.pool.Put(object)
}
