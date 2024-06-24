package xpool_test

import (
	"bytes"
	"io"
	"testing"
	"testing/quick"

	"github.com/peczenyj/xpool"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXPoolBasicGetPut(t *testing.T) {
	t.Parallel()

	pool := xpool.New[io.ReadWriter](func() io.ReadWriter {
		return new(bytes.Buffer)
	})

	f := func(p []byte) bool {
		rw := pool.Get()
		defer pool.Put(rw)

		_, _ = rw.Write(p)

		readed, err := io.ReadAll(rw)
		if err != nil {
			return false
		}

		return bytes.Equal(p, readed)
	}

	err := quick.Check(f, nil)
	require.NoError(t, err)
}

func TestXPoolMultipleGets(t *testing.T) {
	t.Parallel()

	pool := xpool.New[io.ReadWriter](func() io.ReadWriter {
		return new(bytes.Buffer)
	})

	rw1 := pool.Get()
	defer pool.Put(rw1)

	require.NotNil(t, rw1)

	rw2 := pool.Get()
	defer pool.Put(rw2)

	require.NotNil(t, rw2)

	assert.NotSame(t, rw1, rw2)
}
