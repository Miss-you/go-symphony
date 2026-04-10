# T02 Todo / Residual Notes

## Open Items

- No blocking residuals for `T02`.
- The `test-e2e` target now executes successfully as a command contract, but real end-to-end behavior is intentionally deferred to later runtime tasks.

## Scope Guard

- `T02` may create `go.mod`, `cmd/symphony`, and the approved `internal/...` package layout.
- `T02` must not invent runtime domain types, config formats, provider adapters, or workflow behavior early.
