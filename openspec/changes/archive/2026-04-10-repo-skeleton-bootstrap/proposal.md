## Why

The repository already has an approved design and task board, but it is still not a valid Go module and does not yet contain the package skeleton that downstream implementation tasks depend on. Until `T02` lands, the canonical build/test/lint commands cannot execute successfully and later tasks have no stable package homes.

## What Changes

- initialize the root Go module
- add the approved `cmd/symphony` entrypoint
- add the approved flat `internal/...` package layout as placeholder packages
- make the canonical `make` targets run against a real Go module

## Capabilities

### Modified Capabilities

- None. This is an internal bootstrap change under the already approved design.

## Impact

- `go.mod`
- `cmd/symphony/`
- approved `internal/...` package directories
- `workspace/T02/`
