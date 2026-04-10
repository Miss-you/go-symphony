# T02 Test Strategy

## Goal

`T02 Repo Skeleton` must prove that the repository has crossed from planning-only state into a valid Go module with the approved top-level package layout and a working build/test/lint entrypoint.

## What Must Be Proven

1. The Go module initializes cleanly.
2. The `cmd/symphony` entrypoint exists and compiles.
3. The approved `internal/...` package layout exists without leaking provider-specific behavior into the core packages.
4. The canonical `make` targets now run successfully instead of stopping at the missing-`go.mod` guard.

## Verification Matrix

### 1. TDD red-green check for the entrypoint

Commands:

```bash
go test ./cmd/symphony
```

Why this matters:

- The first run must fail before `main.go` exists.
- The second run must pass after the minimal entrypoint is added.

### 2. Repository-wide Go test gate

Command:

```bash
go test ./...
```

Why this matters:

- Proves the module and all placeholder packages compile together.
- Matches the task board gate for `T02`.

### 3. Canonical Make targets

Commands:

```bash
make build
make lint
make test
make test-e2e
make verify
```

Why this matters:

- Proves the root development commands now execute against a real Go module.
- Confirms the bootstrap did not only make ad hoc commands work.

## Acceptance Threshold

`T02` is done only if:

- `go test ./...` passes
- the Make targets above pass
- the directory layout matches the approved V1 package layout

If any of those fail, `T02` is not complete.
