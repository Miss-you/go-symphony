# T19 Test Strategy

## Purpose

T19 is about operator confidence, not new runtime semantics. The tests must prove:

- Linear probe code can be exercised without credentials or network.
- The probe does not accidentally start runtime/Codex components.
- Runtime smoke filtering can only dispatch the selected work item.
- Production `cmd/symphony` behavior remains untouched.

## Test Mapping

| Goal | Test Evidence |
| --- | --- |
| Filter one selected work item | `go test ./internal/tracker/...` covers candidate, state, and refresh filtering by ID and identifier. |
| Preserve read errors | `internal/tracker` tests cover error propagation from wrapped readers. |
| Probe argument behavior | `go test ./cmd/symphony-verify/...` covers `linear` flags, limits, refresh IDs, and provider validation. |
| Probe works without Linear | `cmd/symphony-verify` tests inject fake settings/readers and never call the real Linear HTTP client. |
| Probe stays read-only | Boundary test checks the verification package's imports do not include runtime, workspace, orchestrator, or Codex packages for the Linear probe path. |
| Runtime smoke is guarded | `cmd/symphony-verify` tests cover missing acknowledgement and missing `--only-issue` failures before runtime startup. |
| Runtime smoke can be controlled | Tests inject a fake runtime starter and verify the filtered reader is passed into `cli.StartRuntime` options. |
| Operator docs are usable | Documentation review checks for copyable commands, required environment variables, and explicit live-run risk warnings. |

## Commands

Targeted:

```bash
go test ./internal/tracker/... ./cmd/symphony-verify/...
openspec validate --type change verification-workflows
```

Broad:

```bash
go test ./...
make build
make test-e2e
make verify
openspec validate --specs
git diff --check
```

## Manual Live Smoke

Live Linear/Codex validation is optional evidence because it requires credentials and intentionally launches Codex:

```bash
export LINEAR_API_KEY=...
go run ./cmd/symphony-verify linear WORKFLOW.verify.md
go run ./cmd/symphony-verify run \
  --i-understand-that-this-will-be-running-without-the-usual-guardrails \
  --only-issue ABC-123 \
  --port 34567 \
  --timeout 10m \
  WORKFLOW.verify.md
```

If credentials or a disposable Linear issue are unavailable, record that live smoke was not run instead of treating it as passed.
