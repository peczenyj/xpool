# Agent Context: xpool

This document provides an architectural overview, development workflows, and coding conventions for the `xpool` project, for any coding agent or contributor.

## Project Overview

`xpool` is a tiny, type-safe generic wrapper around the standard library's
[`sync.Pool`](https://pkg.go.dev/sync#Pool). It removes the `interface{}` /
`any` casting boilerplate of `sync.Pool` and adds first-class support for
resettable ("monadic") objects. It targets **Go 1.18+** (generics) and has no
production dependencies beyond `testify` (used only in tests).

The library is organized in two layers: a niladic **core** (`xpool`, repo root)
and a **monadic** layer (`monadic`) that composes it for objects whose reset
takes an argument.

### Core Package (`xpool`, repo root)

The foundation, in `pool.go`. It deals only with a generic type `T` and never
needs reflection.

- **`Pool[T]`:** the central interface — `Get() T` / `Put(T)`. For convenience,
  a `*sync.Pool` already satisfies `Pool[any]` (asserted at `pool.go:22`).
- **`New`:** plain pool with no reset behavior.
- **`NewWithResetter`:** constrains `T` to the `Resetter` interface (`Reset()`)
  and calls it before each `Put`.
- **`NewWithCustomResetter`:** takes an arbitrary `func(T)` reset callback (must
  be thread-safe; panics if nil). Useful for logging, tracing, or non-`Resetter`
  types.
- **Internals:** `simplePool[T]` wraps a `*sync.Pool`, storing values as `any`
  and type-asserting on `Get` with a `ctor()` fallback when the pool is empty.
  `resettablePool[T]` decorates another `Pool[T]`, running the reset callback
  before delegating `Put`.

### Monadic Package (`monadic`)

For objects whose reset takes a value — e.g. `bytes.Reader.Reset(b)` or
`gzip.Writer.Reset(w)`. In `monadic/pool.go`.

- **`Pool[S, T]`:** `Get(state S) T` / `Put(T)`. `Get` resets the object **with**
  the state; `Put` resets it back to the **zero value of S** (the unit of
  delivery returns clean).
- **`New`:** constrains `T` to `Resetter[S]` (`Reset(state S)`).
- **`NewWithCustomResetter`:** takes a `func(object T, state S)` resetter,
  letting you adapt types whose reset signature differs (see the `flate.Resetter`
  example in `monadic/pool_test.go`).
- **Internals:** built on top of `xpool.NewWithCustomResetter` — the monadic pool
  supplies the on-put resetter (zero value of `S`) to the core pool and applies
  the on-get resetter itself in `resettableMonadicPool.Get`.

Key design point: a resetter is **optional** in `xpool` but **mandatory** in
`monadic`. If you don't want a resetter, use a plain `xpool.Pool`.

## Building and Running

The project uses [Task](https://taskfile.dev/) for workflow automation. A
`Makefile` is provided as a thin proxy to `task`.

| Command | Description |
| :--- | :--- |
| `task test` | Runs all tests using `gotestsum`. |
| `task test:race` | Runs tests with the race detector. |
| `task test:coverage` | Runs tests with race + atomic coverage into `coverage.txt`. |
| `task lint` | Executes `golangci-lint` with the project's configuration. |
| `task lint:fix` | Runs `golangci-lint run --fix`. |
| `task format` | Formats the codebase via `golangci-lint fmt` (`gofumpt`, `goimports`, `gci`). |
| `task tidy` | Runs `go mod tidy`. |
| `task tidy:check` | Fails if `go.mod`/`go.sum` are not tidy. |
| `task ci` | Full pre-push gate: `tidy:check` + `lint` + build + `test`. |
| `task changelog` | Regenerates `CHANGELOG.md` using `git-cliff`. |

### Prerequisites

- Go 1.18+ (CI tests every minor version from 1.18 through 1.25, plus `stable`)
- `task` (Taskfile runner)
- `golangci-lint` v2 (CI pins `v2.12.2`)
- `gotestsum` (used by the test tasks)
- `git-cliff` (only for regenerating the changelog)

## Development Conventions

### Coding Style

- Follow standard Go idioms and effective Go practices.
- **Formatting:** Run `task format`. The project uses `gofumpt`, `goimports`,
  and `gci` for import ordering (Standard → Default → `github.com/peczenyj/xpool`).
- **Linting:** Enforced via `.golangci.yml` (golangci-lint v2 config). Always run
  `task lint` before submitting changes. Enabled linters beyond the standard set
  include `gocritic`, `revive`, `misspell`, `gocyclo`, `bodyclose`, `errorlint`,
  and `prealloc`.
- **Go version floor:** Do not use language or stdlib features newer than Go
  1.18 — the CI matrix builds against it.

### Testing Practices

- Tests live alongside the source in `*_test.go` files; runnable examples
  (`Example*`) double as documentation and are verified by their `// Output:`.
- Use `testify` for assertions and `testing/quick` for property checks (see
  `monadic/pool_test.go`).
- Coverage is expected to stay at/near 100% (`.codecov.yml` sets a 90% floor).

### Releases & Changelog

- Commit messages follow [Conventional Commits](https://www.conventionalcommits.org);
  `git-cliff` (`cliff.toml`) groups them into `CHANGELOG.md`. Run `task changelog`
  before tagging a release.
- Tagging `v*` triggers `.github/workflows/release.yml`, which publishes a source
  archive with SHA256 checksums and SLSA build provenance.
