# xpool

[![tag](https://img.shields.io/github/tag/peczenyj/xpool.svg)](https://github.com/peczenyj/xpool/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.18-%23007d9c)
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

Inspired by [xpool](https://pkg.go.dev/go.unistack.org/micro/v3/util/xpool)

## Definition

This package defines an interface `Pool[T any]`

```go
// Pool is a type-safe object pool interface.
// for convenience, *sync.Pool is a Pool[any]
type Pool[T any] interface {
    // Get fetch one item from object pool
    // If needed, will create another object.
    Get() T

    // Put return the object to the pull.
    // It may reset the object before put it back to sync pool.
    Put(object T)
}
```

In such way that `*sync.Pool` is a `Pool[any]`

## Usage

Imagine you need a pool of [io.ReadWrite](https://pkg.go.dev/io#ReadWriter) interfaces implemented by [bytes.Buffer](https://pkg.go.dev/bytes#Buffer). You don't need to cast from `interface{}` `any`more, just do:

```go
    pool := xpool.New(func() io.ReadWriter {
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

Object pools are perfect for that are simple to create, like the ones that have a constructor with no parameters. If we need to specify parameters to create one object, then each combination of parameters may create a different object and they are not easy to use from an object pool.

There are two possible approaches:

* map all possible parameters and create one object pool for combination.
* create stateful object that can be easily created and a particular state can be set via some methods.

The second approach we call "Resettable Objects".

## Dealing with Stateful Resettable Objects

Object pools are perfect for stateless objects, however when dealing with stateful objects we need to be extra careful with the object state. Fortunately, we have some objects that we can easily reset the state before reuse.

Some classes of objects like [hash.Hash](https://pkg.go.dev/hash#Hash) and [bytes.Buffer](https://pkg.go.dev/bytes#Buffer) we can call a method `Reset()` to return the object to his initial state. Others such [bytes.Reader](https://pkg.go.dev/bytes#Reader) and [gzip.Writer](https://pkg.go.dev/compress/gzip#Writer) have a special meaning for a `Reset(state T)` to be possible reuse the same object instead create a new one.

We define two forms of Reset:

The Niladic interface, where `Reset()` receives no arguments (for instance, the `hash.Hash` case) to be executed before put the object back to the pool.

```go
// Resetter interface.
type Resetter interface {
    Reset()
}
```

And the Monadic interface, where `Reset(S)` receives one single argument (for instance, the `gzip.Writer` case) to be executed when we fetch an object from the pool and initialize with a value of type S, and will be resetted back to a zero value of S before put the object back to the pool.

```go
// Resetter interface.
type Resetter[S any] interface {
    Reset(state S)
}
```

Monadic resetters are handling by package [xpool/monadicpool](https://pkg.go.dev/github.com/peczenyj/xpool/monadicpool).

Important: you may not want to expose objects with a `Reset` method, the xpool will not ensure that the type `T` is a `Resetter[S]` unless you define it like this.

### Examples

Calling `Reset()` before put it back to the pool of objects, on [xpool](https://pkg.go.dev/github.com/peczenyj/xpool) package:

```go
    var pool xpool.Pool[hash.Hash] = xpool.NewWithDefaultResetter(func() hash.Hash {
        return sha256.New()
    })

    hasher := pool.Get()   // get a new hash.Hash interface
    defer pool.Put(hasher) // reset it with nil before put back to sync pool.

    _, _ = hasher.Write(p)

    value := hasher.Sum(nil)
```

Calling `Reset(v)` with some value when acquire the instance and `Reset( <zero value> )` before put it back to the pool of objects, on [xpool/monadicpool](https://pkg.go.dev/github.com/peczenyj/xpool/monadicpool) package:

```go
    // this constructor can't infer type S, so you should be explicit!
    var pool monadicpool.Pool[[]byte,*bytes.Reader] = monadicpool.New[[]byte](
        func() *bytes.Reader {
            return bytes.NewReader(nil)
        },
    )

    reader := pool.Get([]byte(`payload`)) // reset the bytes.Reader with payload
    defer pool.Put(reader)                // reset the bytes.Reader with nil

    content, err := io.ReadAll(reader)
```

### Custom Resetters

It is possible set a custom thread-safe Resetter, instead just call `Reset()` or `Reset(v)`, via a custom resette, instead use the default one.

on [xpool](https://pkg.go.dev/github.com/peczenyj/xpool) package:

```go
    //besides the log, both calls are equivalent

    pool:= xpool.NewWithResetter(sha256.New, 
        func(h hash.Hash) {
            h.Reset()

            log.Println("just reset the hash.Hash")
        },
    ),

    // the default resetter try to call `Reset()` method.
    pool:=  xpool.NewWithDefaultResetter(sha256.New),
```

on [xpool/monadicpool](https://pkg.go.dev/github.com/peczenyj/xpool/monadicpool) package:

```go
    // besides the log, both calls are equivalent
    
    // the monadicpool pool will try to call `Reset([]byte)` method by default.
    pool:= monadicpool.New[[]byte](func() *bytes.Reader {
        return bytes.NewReader(nil)
    })

    // the monadicpool pool will try to call the specific resetter callback.
    pool:= monadicpool.NewWithResetter(func() *bytes.Reader {
        return bytes.NewReader(nil)
    }, func(object *bytes.Reader, state []byte) {
        object.Reset(state)

        log.Println("just reset the *bytes.Buffer")
    })
```

You can use custom resetters to handle more complex types of Reset. For instance, the [flate.NewReader](https://pkg.go.dev/compress/flate#NewReader) returns an [io.ReadCloser](https://pkg.go.dev/io#ReadCloser) that also implements [flate.Resetter](https://pkg.go.dev/compress/flate#Resetter) that supports a different kind of `Reset()` that expect two arguments and also returns an error.

If we can discard the error and set the second parameter a constant value like nil, we can:

```go
    // can infer types from resetter
    poolReader := monadicpool.NewWithResetter(func() io.ReadCloser {
        return flate.NewReader(nil)
    }, func(object io.ReadCloser, state io.Reader) {
        if resetter, ok := any(object).(flate.Resetter); ok {
            _ = resetter.Reset(state, nil)
        }
    })
```

An alternative can be create an object to hold different arguments like in the example below:

```go
    type flateResetterArgs struct {
        r    io.Reader
        dict []byte
    }
    // can infer type S from resetter
    poolReader := monadicpool.NewWithResetter(func() io.ReadCloser {
        return flate.NewReader(nil)
    }, func(object io.ReadCloser, state *flateResetterArgs) {
        if resetter, ok := any(oobject).(flate.Resetter); ok {
            _ = resetter.Reset(state.r, state.dict)
        }
    })
```

Custom resetters can do more than just set the status of the object, they can be used to log, trace and extract metrics.

## Important

On [xpool](https://pkg.go.dev/github.com/peczenyj/xpool) the resetter is optional, while on [xpool/monadicpool](https://pkg.go.dev/github.com/peczenyj/xpool/monadicpool) this is mandatory. If you don't want to have resetters on a monadicpool xpool, please create a regular `xpool.Pool`.

## Raw pool

If you want a minimal implementation of a type safe object pool, you can use the `xpool.Pool[T]`.

```go
    var pool xpool.Raw[*bytes.Reader]

    r, ok := pool.Get()
    if !ok {
        // need to explicit create
        r = bytes.NewReader(nil)
    }
    // explicit set the state
    r.Reset([]byte(`payload`))

    defer func() {
        // explicit reset
        r.Reset(nil)

        // add it back to the pool
        pool.Put(r)
    }()

    // use the bytes reader r 
```
