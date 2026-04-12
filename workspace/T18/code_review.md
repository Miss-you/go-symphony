# T18 Code Review

Review verdict: accepted after follow-up fixes.

Findings:

1. Medium: the e2e smoke test did not exercise the scripted Codex transport path and did not prove the HTTP listener stopped after `Close()`.
   - Resolution: updated `internal/cli/runtime_e2e_test.go` to seed an active memory item, inject a scripted transport, wait for `turn/start`, and verify requests fail after runtime close.

2. Low: normal shutdown test asserted only substrings from the offline frame.
   - Resolution: updated `internal/cli/main_test.go` to compare stdout exactly with `dashboard.RenderOffline() + "\n"` and assert the offline marker appears once.

Additional local fix before review:

- Added parser coverage and implementation for `--logs-root --port 0` so another flag is not accepted as a logs-root value.

Follow-up verification:

- `go test -count=1 ./internal/cli/... ./cmd/symphony/...` passed.
- `go test -count=1 -tags=e2e ./internal/cli/...` passed.
