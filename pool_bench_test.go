package xpool_test

import (
	"sync"
	"testing"

	"github.com/peczenyj/xpool"
	"github.com/peczenyj/xpool/monadic"
)

// counter is a trivial niladic Resetter used to keep the benchmarks allocation-free
// once the pool is warm, so the numbers reflect pool overhead rather than ctor cost.
type counter struct{ n int }

func (c *counter) Reset() { c.n = 0 }

// stateful is a trivial monadic Resetter[int].
type stateful struct{ n int }

func (s *stateful) Reset(state int) { s.n = state }

// BenchmarkSyncPoolBaseline is the hand-rolled sync.Pool + type assertion that
// xpool.New abstracts over. It is the reference point for the no-reset path.
func BenchmarkSyncPoolBaseline(b *testing.B) {
	pool := &sync.Pool{New: func() any { return new(counter) }}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c, _ := pool.Get().(*counter)
			pool.Put(c)
		}
	})
}

func BenchmarkXPoolNew(b *testing.B) {
	pool := xpool.New(func() *counter { return new(counter) })

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := pool.Get()
			pool.Put(c)
		}
	})
}

// BenchmarkSyncPoolBaselineWithReset is the reference point for the reset path:
// hand-rolled sync.Pool calling Reset() before Put.
func BenchmarkSyncPoolBaselineWithReset(b *testing.B) {
	pool := &sync.Pool{New: func() any { return new(counter) }}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c, _ := pool.Get().(*counter)
			c.Reset()
			pool.Put(c)
		}
	})
}

func BenchmarkXPoolNewWithResetter(b *testing.B) {
	pool := xpool.NewWithResetter(func() *counter { return new(counter) })

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := pool.Get()
			pool.Put(c)
		}
	})
}

func BenchmarkXPoolNewWithCustomResetter(b *testing.B) {
	pool := xpool.NewWithCustomResetter(
		func() *counter { return new(counter) },
		func(c *counter) { c.Reset() },
	)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := pool.Get()
			pool.Put(c)
		}
	})
}

func BenchmarkMonadicNew(b *testing.B) {
	pool := monadic.New[int](func() *stateful { return new(stateful) })

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s := pool.Get(42)
			pool.Put(s)
		}
	})
}
