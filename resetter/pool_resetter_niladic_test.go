package resetter_test

import (
	"bytes"
	"crypto/sha256"
	"hash"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/require"

	"github.com/peczenyj/xpool/resetter"
)

func TestResetterNiladic(t *testing.T) {
	t.Parallel()

	pool := resetter.NewPool(func() hash.Hash {
		return sha256.New()
	})

	f := func(p []byte) bool {
		hasher := pool.Get()
		defer pool.Put(hasher)

		_, _ = hasher.Write(p)

		reference := sha256.New()
		reference.Write(p)

		return bytes.Equal(reference.Sum(nil), hasher.Sum(nil))
	}

	err := quick.Check(f, nil)
	require.NoError(t, err)
}
