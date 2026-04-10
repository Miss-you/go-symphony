# T10 Residuals And Verification

## Fresh Verification Evidence

- `go test ./internal/tracker/... ./internal/trackers/memory/...`
- `go test ./...`
- `make build`
- `make lint`
- `make test-e2e`

All of the above passed fresh during `T10`.

## Residual Low-Risk Gaps

- `ListByStates` is intentionally frozen in `TrackerReader`, but Go runtime adoption of that method is deferred to later tasks that own workspace cleanup and runtime assembly. Later artifacts must keep that deferral explicit.
- `memory.Reader` is intended to be constructed via `NewReader`. Nil-receiver guarding is not implemented because current usage stays on the constructor path; if later code bypasses the constructor, that risk should be handled in the owning task.

## Final Compare Notes

- The Go core now freezes the same three read operations the Symphony spec and Elixir runtime actually depend on: candidate listing, state-based listing, and refresh by ID.
- The Go task intentionally does not copy Elixir's tracker write surface into the core. That matches the approved Go architecture boundary.
- The memory adapter remains local/test focused and deterministic, with explicit deep-copy behavior so callers cannot mutate adapter-owned seed data.
