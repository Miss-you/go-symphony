# T01 Todo / Residual Notes

## Open Items

- Low-risk note: the content-presence gate in `test_strategy.md` is heuristic (`rg`-based), not semantic validation. For this documentation-only contract task, that is acceptable and not blocking.

## Explicitly Non-Applicable Verification Gates

- `lint`: not applicable because `T01` introduces no source files.
- `build/compile`: not applicable because `T01` must not add or modify repo-skeleton/code paths such as `go.mod`, `cmd/`, or `internal/`.
- `unit tests`: not applicable because `T01` adds no runtime behavior.
- `e2e tests`: not applicable because `T01` adds no executable service path.

These are scope-based exclusions for `T01`, not deferred follow-up work.
