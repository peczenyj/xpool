package xpool_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
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

func TestWithDefaultResetter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		label string
		pool  xpool.Pool[hash.Hash]
	}{
		{
			label: "NewWithResetter + explicit call to Reset()",
			pool: xpool.NewWithResetter(sha256.New, func(h hash.Hash) {
				h.Reset()
			}),
		},
		{
			label: "test NewWithDefaultResetter + implicit call to Reset",
			pool:  xpool.NewWithDefaultResetter(sha256.New),
		},
	}

	for _, testCase := range testCases {
		pool := testCase.pool

		t.Run(testCase.label, func(t *testing.T) {
			t.Parallel()

			f := func(p []byte) bool {
				var hasher hash.Hash = pool.Get()
				defer pool.Put(hasher)

				_, _ = hasher.Write(p)

				reference := sha256.New()
				reference.Write(p)

				return bytes.Equal(reference.Sum(nil), hasher.Sum(nil))
			}

			err := quick.Check(f, nil)
			require.NoError(t, err)
		})
	}
}

func ExampleNew() {
	pool := xpool.New[io.ReadWriter](func() io.ReadWriter {
		return new(bytes.Buffer)
	})

	rw := pool.Get()
	defer pool.Put(rw)

	// your favorite usage of rw

	fmt.Fprint(rw, "example")

	_, _ = io.Copy(os.Stdout, rw)
	// Output: example
}

func ExampleNewWithDefaultResetter() {
	pool := xpool.NewWithDefaultResetter(sha256.New)
	var hasher hash.Hash = pool.Get() // get a new hash.Hash interface
	defer pool.Put(hasher)            // reset it before put back to sync pool.

	_, _ = hasher.Write([]byte(`payload`))

	fmt.Printf("%x", hasher.Sum(nil))
	// Output: 239f59ed55e737c77147cf55ad0c1b030b6d7ee748a7426952f9b852d5a935e5
}
