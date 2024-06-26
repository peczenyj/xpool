# xpool/monadic

## Definition

The intent of this package is to support stateful objects in the [xpool](http://pkg.go.dev/github.com/peczenyj/xpool). By default is uses a `Resetter[S any]` monadic interface `Reset(state S)`.

We define a different Pool interface, where we set a new state `S` on the object when we get it from the pool, and we reset the state by setting a zero value of state `S` before put it back to the object pool.

```go
// Pool monadic is a type-safe object pool interface.
type Pool[S, T any] interface {
    // Get fetch one item from object pool. If needed, will create another object.
    // The state S will be used in the resetter.
    Get(state S) T

    // Put return the object to the pull.
    // A zero value of T will be used in the resetter.
    Put(object T)
}
```

## Usage

We offer two constructors:

```go
    // besides the log, both calls are equivalent

    // the monadic pool will try to call `Reset([]byte)` method by default.
    pool:= monadic.New[[]byte](func() *bytes.Reader {
        return bytes.NewReader(nil)
    })

    // the monadic pool will try to call the specific resetter callback.
    pool:= monadic.NewWithResetter(func() *bytes.Reader {
        return bytes.NewReader(nil)
    }, func(r *bytes.Reader, b []byte) {
        r.Reset(b)

        log.Println("just reset the *bytes.Buffer")
    })
```

using the second constructor, you can build more complex resetters, like:

```go
    // can infer types from resetter
    poolReader := monadic.NewWithResetter(func() io.ReadCloser {
        return flate.NewReader(nil)
    }, func(object io.ReadCloser, state io.Reader) {
        if resetter, ok := any(object).(flate.Resetter); ok {
            _ = resetter.Reset(state, nil)
        }
    })
```
