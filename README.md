# xpool

[![tag](https://img.shields.io/github/tag/peczenyj/xpool.svg)](https://github.com/peczenyj/xpool/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-%23007d9c)
[![GoDoc](https://pkg.go.dev/badge/github.com/peczenyj/xpool)](http://pkg.go.dev/github.com/peczenyj/xpool)
[![Go](https://github.com/peczenyj/xpool/actions/workflows/go.yml/badge.svg)](https://github.com/peczenyj/xpool/actions/workflows/go.yml)
[![Lint](https://github.com/peczenyj/xpool/actions/workflows/lint.yml/badge.svg)](https://github.com/peczenyj/xpool/actions/workflows/lint.yml)
[![codecov](https://codecov.io/gh/peczenyj/xpool/graph/badge.svg?token=9y6f3vGgpr)](https://codecov.io/gh/peczenyj/xpool)
[![Report card](https://goreportcard.com/badge/github.com/peczenyj/xpool)](https://goreportcard.com/report/github.com/peczenyj/xpool)
[![CodeQL](https://github.com/peczenyj/xpool/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/peczenyj/xpool/actions/workflows/github-code-scanning/codeql)
[![Dependency Review](https://github.com/peczenyj/xpool/actions/workflows/dependency-review.yml/badge.svg)](https://github.com/peczenyj/xpool/actions/workflows/dependency-review.yml)
[![License](https://img.shields.io/github/license/peczenyj/xpool)](./LICENSE)
[![Latest release](https://img.shields.io/github/release/peczenyj/xpool.svg)](https://github.com/peczenyj/xpool/releases/latest)
[![GitHub Release Date](https://img.shields.io/github/release-date/peczenyj/xpool.svg)](https://github.com/peczenyj/xpool/releases/latest)
[![Last commit](https://img.shields.io/github/last-commit/peczenyj/xpool.svg)](https://github.com/peczenyj/xpool/commit/HEAD)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/peczenyj/xpool/blob/main/CONTRIBUTING.md#pull-request-process)

The xpool is a user-friendly, type-safe version of [sync.Pool](https://pkg.go.dev/sync#Pool).

Inspired by [xpool](go.unistack.org/micro/v3/util/xpool)

## Usage

Imagine you need a pool of [io.ReadWrite](https://pkg.go.dev/io#ReadWriter) interfaces implemented by [bytes.Buffer](https://pkg.go.dev/bytes#Buffer). You don't need to cast from ̀`interface{}` anymore, just do:

```go
    pool := xpool.New[io.ReadWriter](func() io.ReadWriter {
        return new(bytes.Buffer)
    })

    rw := pool.Get()
    defer pool.Put(rw)

    // now you can use a new io.ReadWrite instance
```

instead using pure go

```go
    pool := &sync.Pool{
        New: func() any {
            return new(bytes.Buffer)
        },
    }

    rw, _ := pool.Get().(io.ReadWriter)
    defer pool.Put(rw)

    // now you can use a new io.ReadWrite instance
```

## Dealing with Resettable objects

Some classes of objects like [hash.Hash](https://pkg.go.dev/hash#Hash) and [bytes.Buffer](https://pkg.go.dev/bytes#Buffer) we can call a method `Reset()` to return the object to his initial state. Others such [bytes.Reader](https://pkg.go.dev/bytes#Reader) and [gzip.Writer](https://pkg.go.dev/compress/gzip#Writer) have a special meaning for a `Reset(state T)` to be possible reuse the same object instead create a new one. We can use the special package [xpool/resetter](https://pkg.go.dev/github.com/peczenyj/xpool/resetter).

We define two forms of Reset:

The Niladic interface, where `Reset()` receives no arguments (for instance, the `hash.Hash` case)

```go
// Resetter interface.
type Resetter interface {
    Reset()
}
```

And the Monadic interface, where `Reset(T)` receives one single argument (for instance, the `gzip.Writer` case)

```go
// ResetterMonadic interface.
type ResetterMonadic[T any] interface {
    Reset(t T)
}
```

### Examples

Calling `Reset()` before put it back to the pool of objects.

```go
    pool := resetter.NewPool(func() hash.Hash {
        return sha256.New()
    })

    hasher := pool.Get()   // get a new hash.Hash interface
    defer pool.Put(hasher) // reset it before put back to sync pool.

    _, _ = hasher.Write(p)

    value := hasher.Sum(nil)
```

Calling `Reset(v)` with some value when acquire the instance and `Reset( <zero value> )` before put it back to the pool of objects.

```go
    pool := resetter.NewPoolMonadic(func() *bytes.Reader {
        return bytes.NewReader(nil)
    })

    reader := pool.Get([]byte(`payload`)) // dummy example
    defer pool.Put(reader)                // reset the bytes.Reader

    content, err := io.ReadAll(reader)
```

A more interesting example: acquire a `*gzip.Writer` from pool and automatic reset it to use an `io.Writer`.

```go
    pool := resetter.NewPoolMonadic(func() *gzip.Writer {
        return gzip.NewWriter(nil)
    })

    f, err := os.Open("notes.txt.gz")
    if err != nil {
        log.Fatal(err)
    }

    defer f.Close()

    gzipWriter := pool.Get(f)
    defer pool.Put(gzipWriter)

    fmt.Fprintln(gzipWriter, "this message will be compressed with gzip format")
```
