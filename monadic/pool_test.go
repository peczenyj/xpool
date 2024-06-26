package monadic_test

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

	"github.com/peczenyj/xpool/monadic"
)

func TestResetterMonadic(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		label string
		pool  monadic.Pool[[]byte, *bytes.Reader]
	}{
		{
			label: "monadic New + implicit default Reset",
			pool: monadic.New[[]byte](func() *bytes.Reader {
				return bytes.NewReader(nil)
			}),
		},
		{
			label: "monadic NewWithResetter + explicit custom Reset",
			pool: monadic.NewWithResetter(func() *bytes.Reader {
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

	type callbackArgs struct {
		called bool
		onGet  bool
	}

	t.Run("negative case / has no Reset method", func(t *testing.T) {
		t.Parallel()

		var callbackCalls []callbackArgs

		pool := monadic.New[int](
			func() string { return "" },
			func(called, onGet bool) {
				callbackCalls = append(callbackCalls, callbackArgs{
					called: called,
					onGet:  onGet,
				})
			},
		)

		instance := pool.Get(1)
		pool.Put(instance)

		require.Len(t, callbackCalls, 2)

		assert.False(t, callbackCalls[0].called, "should not call Reset on a string")
		assert.True(t, callbackCalls[0].onGet, "first call should be on Get")
		assert.False(t, callbackCalls[1].called, "should not call Reset on a string")
		assert.False(t, callbackCalls[1].onGet, "second call should be on Put")
	})

	t.Run("negative case / has Reset method but the argument type must match", func(t *testing.T) {
		t.Parallel()

		var callbackCalls []callbackArgs

		pool := monadic.New[int](
			func() io.Reader { return bytes.NewReader(nil) },
			func(called, onGet bool) {
				callbackCalls = append(callbackCalls, callbackArgs{
					called: called,
					onGet:  onGet,
				})
			},
		)

		instance := pool.Get(1)
		pool.Put(instance)

		require.Len(t, callbackCalls, 2)

		assert.False(t, callbackCalls[0].called, "should not call Reset(int) on a *bytes.Reader")
		assert.True(t, callbackCalls[0].onGet, "first call should be on Get")
		assert.False(t, callbackCalls[1].called, "should not call Reset(int) on a *bytes.Reader")
		assert.False(t, callbackCalls[1].onGet, "second call should be on Put")
	})

	t.Run("positive case", func(t *testing.T) {
		t.Parallel()

		var callbackCalls []callbackArgs

		pool := monadic.New[[]byte](
			func() io.Reader { return bytes.NewReader(nil) },
			func(called, onGet bool) {
				callbackCalls = append(callbackCalls, callbackArgs{
					called: called,
					onGet:  onGet,
				})
			},
		)

		instance := pool.Get([]byte(`something`))
		pool.Put(instance)

		require.Len(t, callbackCalls, 2)

		assert.True(t, callbackCalls[0].called, "should call Reset([]byte) on a *bytes.Reader")
		assert.True(t, callbackCalls[0].onGet, "first call should be on Get")
		assert.True(t, callbackCalls[1].called, "should call Reset([]byte) on a *bytes.Reader")
		assert.False(t, callbackCalls[1].onGet, "second call should be on Put")
	})
}

func ExampleNew() {
	// can't infer type V, must be explicit
	var pool monadic.Pool[[]byte, io.Reader] = monadic.New[[]byte](func() io.Reader {
		return bytes.NewReader(nil)
	})

	var reader io.Reader = pool.Get([]byte(`payload`))
	defer pool.Put(reader)

	_, _ = io.Copy(os.Stdout, reader)
	// Output: payload
}

func ExampleNewWithResetter() {
	// can't infer type V, must be explicit
	poolWriter := monadic.New[io.Writer](
		func() io.WriteCloser {
			zw, _ := flate.NewWriter(nil, flate.DefaultCompression)
			return zw
		},
	)

	// can infer type V from resetter
	poolReader := monadic.NewWithResetter(func() io.ReadCloser {
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
