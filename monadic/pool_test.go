//nolint:staticcheck //not needed for examples.
package monadic_test

import (
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"io"
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
			pool: monadic.NewWithCustomResetter(func() *bytes.Reader {
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
				reader := pool.Get(b)
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

type box struct{ n int }

var errReset = errors.New("reset failed")

func TestFallibleMonadicHappyPath(t *testing.T) {
	t.Parallel()

	pool := monadic.NewWithFallibleResetter(
		func() *bytes.Buffer { return new(bytes.Buffer) },
		func(b *bytes.Buffer, state string) error {
			b.Reset()
			b.WriteString(state)

			return nil
		},
		nil,
	)

	b := pool.Get("hello")
	assert.Equal(t, "hello", b.String())

	pool.Put(b) // zero-value reset on Put succeeds
	assert.Empty(t, b.String())
}

func TestFallibleMonadicGetFailsThenRebuilds(t *testing.T) {
	t.Parallel()

	var errs []error

	attempt := 0

	pool := monadic.NewWithFallibleResetter(
		func() *box { return &box{n: -1} },
		func(b *box, state int) error {
			attempt++
			if attempt == 1 {
				return errReset // fail the first Get reset
			}

			b.n = state

			return nil
		},
		func(err error, _ *box) { errs = append(errs, err) },
	)

	b := pool.Get(7)

	assert.Equal(t, 7, b.n, "Get must return an object reset from a fresh ctor")
	require.Len(t, errs, 1)
	require.ErrorIs(t, errs[0], errReset)
}

func TestFallibleMonadicGetFailsTwiceReturnsBestEffort(t *testing.T) {
	t.Parallel()

	count := 0

	pool := monadic.NewWithFallibleResetter(
		func() *box { return &box{n: -1} },
		func(*box, int) error { return errReset }, // always fails
		func(error, *box) { count++ },
	)

	b := pool.Get(7)

	assert.Equal(t, -1, b.n, "a never-reset object is returned best effort")
	assert.Equal(t, 2, count, "onError fires for both the pooled object and the rebuilt one")
}

func TestFallibleMonadicNilOnErrorDoesNotPanic(t *testing.T) {
	t.Parallel()

	pool := monadic.NewWithFallibleResetter(
		func() *box { return new(box) },
		func(*box, int) error { return errReset },
		nil,
	)

	require.NotPanics(t, func() { _ = pool.Get(1) })
}

func TestFallibleMonadicPutZeroResetFailureDrops(t *testing.T) {
	t.Parallel()

	var putErr error

	pool := monadic.NewWithFallibleResetter(
		func() *box { return new(box) },
		func(b *box, state int) error {
			if state == 0 {
				return errReset // fail only the zero-value reset on Put
			}

			b.n = state

			return nil
		},
		func(err error, _ *box) { putErr = err },
	)

	b := pool.Get(5)
	assert.Equal(t, 5, b.n)

	pool.Put(b) // zero reset fails: object dropped and reported

	require.ErrorIs(t, putErr, errReset)
}

func TestFallibleMonadicPanicsOnNilResetter(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		monadic.NewWithFallibleResetter[int](func() *box { return new(box) }, nil, nil)
	},
		"must panic",
	)
}

func ExampleNewWithFallibleResetter() {
	poolWriter := monadic.New[io.Writer, *flate.Writer](func() *flate.Writer {
		zw, _ := flate.NewWriter(nil, flate.DefaultCompression)
		return zw
	})

	// flate.Resetter.Reset returns an error; surface it instead of discarding it.
	poolReader := monadic.NewWithFallibleResetter[io.Reader, io.ReadCloser](
		func() io.ReadCloser {
			return flate.NewReader(nil)
		},
		func(reader io.ReadCloser, state io.Reader) error {
			resetter, ok := reader.(flate.Resetter)
			if !ok {
				return nil
			}

			return resetter.Reset(state, nil)
		},
		func(err error, _ io.ReadCloser) {
			fmt.Println("reset failed:", err)
		},
	)

	var b bytes.Buffer

	r := strings.NewReader("hello, world!\n")

	zwc := poolWriter.Get(&b)
	defer poolWriter.Put(zwc)

	if _, err := io.Copy(zwc, r); err != nil {
		panic(err)
	}

	if err := zwc.Close(); err != nil {
		panic(err)
	}

	zrc := poolReader.Get(&b)
	defer poolReader.Put(zrc)

	_, _ = io.Copy(os.Stdout, zrc)

	// Output:
	// hello, world!
}

func ExampleNew() {
	var pool monadic.Pool[[]byte, *bytes.Reader] = monadic.New[[]byte, *bytes.Reader](func() *bytes.Reader {
		return bytes.NewReader(nil)
	})

	reader := pool.Get([]byte(`payload`))
	defer pool.Put(reader)

	_, _ = io.Copy(os.Stdout, reader)
	// Output: payload
}

func ExampleNewWithCustomResetter() {
	poolWriter := monadic.New[io.Writer, *flate.Writer](func() *flate.Writer {
		zw, _ := flate.NewWriter(nil, flate.DefaultCompression)
		return zw
	})

	poolReader := monadic.NewWithCustomResetter[io.Reader, io.ReadCloser](func() io.ReadCloser {
		return flate.NewReader(nil)
	}, func(reader io.ReadCloser, state io.Reader) {
		reseter, _ := reader.(flate.Resetter)
		_ = reseter.Reset(state, nil)
	})

	var b bytes.Buffer

	r := strings.NewReader("hello, world!\n")

	zwc := poolWriter.Get(&b)
	defer poolWriter.Put(zwc)

	if _, err := io.Copy(zwc, r); err != nil {
		panic(err)
	}

	if err := zwc.Close(); err != nil {
		panic(err)
	}

	zrc := poolReader.Get(&b)
	defer poolReader.Put(zrc)

	_, _ = io.Copy(os.Stdout, zrc)

	// Output:
	// hello, world!
}
