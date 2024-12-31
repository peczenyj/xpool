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

func ExampleNew() {
	var pool monadic.Pool[[]byte, *bytes.Reader] = monadic.New(func() *bytes.Reader {
		return bytes.NewReader(nil)
	})

	reader := pool.Get([]byte(`payload`))
	defer pool.Put(reader)

	_, _ = io.Copy(os.Stdout, reader)
	// Output: payload
}

func ExampleNewWithCustomResetter() {
	poolWriter := monadic.New(func() *flate.Writer {
		zw, _ := flate.NewWriter(nil, flate.DefaultCompression)
		return zw
	})

	poolReader := monadic.NewWithCustomResetter(func() io.ReadCloser {
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
