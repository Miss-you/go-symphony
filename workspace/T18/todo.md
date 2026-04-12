# T18 Residual Notes

## Live Provider E2E

- Real Linear/Codex live e2e was not run during implementation because this environment does not provide an explicit live-test opt-in or dedicated Linear/Codex test credentials.
- The repository-level `make test-e2e` gate is still required. For T18 it includes a no-network e2e-tagged smoke test that starts the memory runtime, binds an ephemeral dashboard/API listener, verifies `/` and `/api/v1/state`, and shuts down cleanly.
- A future live-provider e2e should be environment-gated and should skip explicitly when credentials are absent, matching the original Symphony behavior.

## Accepted Limits

- The Go log helper implements the parity-relevant `--logs-root` path behavior with test-safe restoration, but it does not implement Elixir's rotating disk log handler. Full rotation is outside V1 unless a later task expands logging requirements.
- The CLI uses the existing `dashboard.RenderOffline()` frame for shutdown parity. It does not add a separate terminal dashboard state owner.
