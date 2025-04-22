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

	// pool can infer T from constructor
	pool := xpool.New(func() io.ReadWriter {
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

	// pool can infer T from constructor
	pool := xpool.New(func() io.ReadWriter {
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

func TestResetter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		label string
		pool  xpool.Pool[hash.Hash]
	}{
		{
			label: "NewWithCustomResetter + explicit call to Reset() via callback",
			pool: xpool.NewWithCustomResetter(sha256.New, func(h hash.Hash) {
				h.Reset()
			}),
		},
		{
			label: "test NewWithDefaultResetter + implicit call to Reset",
			pool:  xpool.NewWithResetter(sha256.New),
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

func TestNewWithCustomResetter(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		xpool.NewWithCustomResetter(sha256.New, nil)
	}, "must panic")
}

func ExampleNew() {
	// pool can infer T from constructor
	pool := xpool.New(func() io.ReadWriter {
		return new(bytes.Buffer)
	})

	rw := pool.Get()
	defer pool.Put(rw)

	// your favorite usage of rw

	_, _ = fmt.Fprint(rw, "example")

	_, _ = io.Copy(os.Stdout, rw)
	// Output:
	// example
}

func ExampleNewWithCustomResetter() {
	// pool can infer T from constructor
	var pool xpool.Pool[hash.Hash] = xpool.NewWithCustomResetter(sha256.New,
		func(h hash.Hash) {
			h.Reset()
			fmt.Println("hash resetted with success")
		},
	)

	var hasher hash.Hash = pool.Get() // get a new hash.Hash interface

	_, _ = hasher.Write([]byte(`payload`))

	fmt.Printf("%x\n", hasher.Sum(nil))

	pool.Put(hasher) // reset it before put back to sync pool.
	// Output:
	// 239f59ed55e737c77147cf55ad0c1b030b6d7ee748a7426952f9b852d5a935e5
	// hash resetted with success
}

func ExampleNewWithResetter() {
	// pool can infer T from constructor
	var pool xpool.Pool[hash.Hash] = xpool.NewWithResetter(sha256.New)

	var hasher hash.Hash = pool.Get() // get a new hash.Hash interface
	defer pool.Put(hasher)            // reset it before put back to sync pool.

	_, _ = hasher.Write([]byte(`payload`))

	fmt.Printf("%x\n", hasher.Sum(nil))

	// Output:
	// 239f59ed55e737c77147cf55ad0c1b030b6d7ee748a7426952f9b852d5a935e5
}
