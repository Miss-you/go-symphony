# T04 Todo / Residual Notes

## Open Items

- Final compare against the Elixir implementation confirms that `T04` now preserves the intended defaults, `LINEAR_*` env fallbacks, `workspace.root` path handling, startup fail-fast behavior, and last-known-good reload semantics while exposing a provider-neutral typed config contract.
- Final review should keep the legacy `tracker.*` to `Settings.Provider` mapping one-way so later packages do not drift back to raw workflow config reads.
- Verification evidence recorded on 2026-04-10:
  - `go test ./internal/config/...`
  - `go test ./...`
  - `make build`
  - `make lint`
  - `make test-e2e`
- `make test-e2e` passed for `T04`, but it currently acts as a repository command-contract / compile gate because the broader runtime e2e surfaces do not exist yet. The real behavior proof for `T04` remains the focused `internal/config` tests that cover typed defaults, env/path resolution, startup fail-fast behavior, and atomic raw-plus-typed reload fallback.

## Scope Guard

- `T04` may add typed settings normalization, validation, and typed store access under `internal/config`.
- `T04` must not change raw `WORKFLOW.md` parsing semantics from `T03`.
- `T04` must not add prompt rendering, workflow selection, orchestrator wiring, or provider-specific runtime behavior outside compatibility parsing.
