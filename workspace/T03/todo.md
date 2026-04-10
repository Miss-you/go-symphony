# T03 Todo / Residual Notes

## Open Items

- Final compare against the Elixir implementation confirms that T03 preserves the intended path-precedence, front-matter parsing, blank-prompt fallback, and last-known-good reload semantics.
- Verification evidence recorded on 2026-04-10:
  - `go test ./internal/config/...`
  - `go test ./...`
  - `make build`
  - `make lint`
  - `make test-e2e`
- `make test-e2e` passed for `T03`, but it currently acts as a repository command-contract / compile gate because the broader runtime e2e surfaces do not exist yet. This task's real behavior proof still comes from the focused `internal/config` tests plus the broader compile/lint sweep.
- The current implementation exports `EffectivePromptTemplate()`. If later prompt-building tasks want a narrower API surface, they can revisit whether that helper should remain exported or become package-local.

## Scope Guard

- `T03` may add raw workflow-loading and reload-cache behavior under `internal/config`.
- `T03` must not introduce typed provider-neutral config normalization; that belongs to `T04`.
- `T03` must not implement prompt rendering, workflow-bundle selection, orchestrator integration, or tracker-specific validation.
