## 1. Runner Contract

- [x] 1.1 Add runner tests for local command execution, timeout handling, output/status normalization, and non-zero exit behavior.
- [x] 1.2 Add runner tests for SSH argv construction, `SYMPHONY_SSH_CONFIG`, host/port parsing, user-prefixed hosts, bracketed IPv6, and remote `bash -lc` wrapping.
- [x] 1.3 Add runner tests for stateless host selection with local admission, preferred host, least-loaded fallback, tie-breaking, unknown preferred host fallback, and all-hosts-full rejection.
- [x] 1.4 Implement the runner command request/result types, executor interface, local executor, SSH executor, process-start seam, and host selector.

## 2. Workspace Boundary Refactor

- [x] 2.1 Refactor `internal/workspace` so lifecycle hooks execute through a runner executor while preserving existing fatal/best-effort hook semantics.
- [x] 2.2 Refactor host-addressed remote workspace create/remove paths so workspace keeps lifecycle command policy and delegates execution to runner.
- [x] 2.3 Update workspace tests to prove lifecycle behavior is unchanged and workspace no longer owns local shell launch or SSH command construction.

## 3. Orchestrator Admission Wiring

- [x] 3.1 Add orchestrator host-load projection from private running entries.
- [x] 3.2 Wire the existing admission path through `runner.HostSelection` while keeping total concurrency, state limits, claims, retries, and cleanup intent orchestrator-owned.
- [x] 3.3 Update orchestrator tests for runner-backed host selection, capacity rejection, preferred host lineage, and unchanged retry/polling ownership.

## 4. Verification

- [x] 4.1 Run `go test ./internal/runner/...`.
- [x] 4.2 Run `go test ./internal/workspace/... ./internal/orchestrator/...`.
- [x] 4.3 Run `go test ./...`, `make build`, `make lint`, and `make test-e2e`, recording any current e2e applicability limitation in `workspace/T08/todo.md`.
