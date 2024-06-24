package resetter_test

import (
	"bytes"
	"io"
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
