package resetter_test

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/require"

	"github.com/peczenyj/xpool/resetter"
)

func TestResetterMonadic(t *testing.T) {
	t.Parallel()

	pool := resetter.NewPoolMonadic(func() *bytes.Reader {
		return bytes.NewReader(nil)
	})

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
}

func ExampleResetterMonadic() {
	pool := resetter.NewPoolMonadic(func() *bytes.Reader {
		return bytes.NewReader(nil)
	})

	reader := pool.Get([]byte(`payload`))
	defer pool.Put(reader)

	_, _ = io.Copy(os.Stdout, reader)
	// Output: payload
}

func ExampleResetterMonadic_example() {
	pool := resetter.NewPoolMonadic(func() *gzip.Writer {
		return gzip.NewWriter(nil)
	})

	f, err := os.Open("notes.txt")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	gzipWriter := pool.Get(f)
	defer pool.Put(gzipWriter)

	fmt.Fprintln(gzipWriter, "this message will be compressed with gzip format")
}
