package monadicpool_test

import (
	"bytes"
	"compress/flate"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/peczenyj/xpool/monadicpool"
)

func TestResetterMonadic(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		label string
		pool  monadicpool.Pool[[]byte, *bytes.Reader]
	}{
		{
			label: "monadic New + implicit default Reset",
			pool: monadicpool.New[[]byte](func() *bytes.Reader {
				return bytes.NewReader(nil)
			}),
		},
		{
			label: "monadic NewWithResetter + explicit custom Reset",
			pool: monadicpool.NewWithResetter(func() *bytes.Reader {
				return bytes.NewReader(nil)
			}, func(r *bytes.Reader, b []byte) {
				r.Reset(b)
			}),
		},
	}

	for _, testCase := range testCases {
		pool := testCase.pool

		t.Run(testCase.label, func(t *testing.T) {
			t.Parallel()

			f := func(b []byte) bool {
				var reader *bytes.Reader = pool.Get(b)
				defer pool.Put(reader)

				readed, err := io.ReadAll(reader)
				if err != nil {
					return false
				}

				return bytes.Equal(b, readed)
			}

			err := quick.Check(f, nil)
			require.NoError(t, err)
		})
	}
}

func TestOnResetCallbacks(t *testing.T) {
	t.Parallel()

	t.Run("negative case / has no Reset method", func(t *testing.T) {
		t.Parallel()

		var (
			onGetCalls []bool
			onPutCalls []bool
		)

		pool := monadicpool.New[int](
			func() string { return "" },
			monadicpool.WithOnGetResetCallback(func(called bool) {
				onGetCalls = append(onGetCalls, called)
			}),
			monadicpool.WithOnPutResetCallback(func(called bool) {
				onPutCalls = append(onPutCalls, called)
			}),
		)

		instance := pool.Get(1)
		pool.Put(instance)

		require.Len(t, onGetCalls, 1)
		assert.False(t, onGetCalls[0], "should not call Reset(int) on Get on a string")

		require.Len(t, onPutCalls, 1)
		assert.False(t, onPutCalls[0], "should not call Reset(int) on Put on a string")
	})

	t.Run("negative case / has Reset method but the argument type must match", func(t *testing.T) {
		t.Parallel()

		var (
			onGetCalls []bool
			onPutCalls []bool
		)

		pool := monadicpool.New[int](
			func() io.Reader { return bytes.NewReader(nil) },
			monadicpool.WithOnGetResetCallback(func(called bool) {
				onGetCalls = append(onGetCalls, called)
			}),
			monadicpool.WithOnPutResetCallback(func(called bool) {
				onPutCalls = append(onPutCalls, called)
			}),
		)

		instance := pool.Get(1)
		pool.Put(instance)

		require.Len(t, onGetCalls, 1)
		assert.False(t, onGetCalls[0], "should not call Reset(int) on Get on a *bytes.Reader")

		require.Len(t, onPutCalls, 1)
		assert.False(t, onPutCalls[0], "should not call Reset(int) on Put on a *bytes.Reader")
	})

	t.Run("positive case", func(t *testing.T) {
		t.Parallel()

		var (
			onGetCalls []bool
			onPutCalls []bool
		)

		pool := monadicpool.New[[]byte](
			func() io.Reader { return bytes.NewReader(nil) },
			monadicpool.WithOnGetResetCallback(func(called bool) {
				onGetCalls = append(onGetCalls, called)
			}),
			monadicpool.WithOnPutResetCallback(func(called bool) {
				onPutCalls = append(onPutCalls, called)
			}),
		)

		instance := pool.Get([]byte(`something`))
		pool.Put(instance)

		require.Len(t, onGetCalls, 1)
		assert.True(t, onGetCalls[0], "should not call Reset([]byte) on Get on a *bytes.Reader")

		require.Len(t, onPutCalls, 1)
		assert.True(t, onPutCalls[0], "should not call Reset([]byte) on Put on a *bytes.Reader")
	})
}

func ExampleNew() {
	// can't infer type V, must be explicit
	var pool monadicpool.Pool[[]byte, io.Reader] = monadicpool.New[[]byte](func() io.Reader {
		return bytes.NewReader(nil)
	})

	var reader io.Reader = pool.Get([]byte(`payload`))
	defer pool.Put(reader)

	_, _ = io.Copy(os.Stdout, reader)
	// Output: payload
}

func ExampleNewWithResetter() {
	// can't infer type V, must be explicit
	poolWriter := monadicpool.New[io.Writer](
		func() io.WriteCloser {
			zw, _ := flate.NewWriter(nil, flate.DefaultCompression)
			return zw
		},
	)

	// can infer type V from resetter
	poolReader := monadicpool.NewWithResetter(func() io.ReadCloser {
		return flate.NewReader(nil)
	}, func(t io.ReadCloser, v io.Reader) {
		if resetter, ok := any(t).(flate.Resetter); ok {
			_ = resetter.Reset(v, nil)
		}
	})

	var b bytes.Buffer

	r := strings.NewReader("hello, world!\n")

	zwc := poolWriter.Get(&b)
	defer poolWriter.Put(zwc)

	if _, err := io.Copy(zwc, r); err != nil {
		log.Fatal(err)
	}

	if err := zwc.Close(); err != nil {
		log.Fatal(err)
	}

	zrc := poolReader.Get(&b)
	defer poolReader.Put(zrc)

	_, _ = io.Copy(os.Stdout, zrc)

	// Output:
	// hello, world!
}
