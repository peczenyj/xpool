package resetter_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
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

func ExampleResetter() {
	pool := resetter.NewPool(func() hash.Hash {
		return sha256.New()
	})

	hasher := pool.Get()   // get a new hash.Hash interface
	defer pool.Put(hasher) // reset it before put back to sync pool.

	_, _ = hasher.Write([]byte(`payload`))

	fmt.Printf("%x", hasher.Sum(nil))
	// Output: 239f59ed55e737c77147cf55ad0c1b030b6d7ee748a7426952f9b852d5a935e5
}
