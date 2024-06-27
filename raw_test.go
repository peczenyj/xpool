package xpool_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/peczenyj/xpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRaw(t *testing.T) {
	t.Parallel()

	var pool xpool.Raw[int]

	n, ok := pool.Get()
	require.False(t, ok)

	if !ok {
		n = 1
	}

	pool.Put(n)

	n2, ok := pool.Get()
	require.True(t, ok)
	assert.Equal(t, 1, n2)
}

func ExampleRaw() {
	var pool xpool.Raw[*bytes.Reader]

	r, ok := pool.Get()
	if !ok {
		// need to explicit create
		r = bytes.NewReader(nil)
	}
	// explicit set the state
	r.Reset([]byte(`payload`))

	defer func() {
		// explicit reset
		r.Reset(nil)

		// add it back to the pool
		pool.Put(r)
	}()

	_, _ = io.Copy(os.Stdout, r)
	// Output:
	// payload
}
