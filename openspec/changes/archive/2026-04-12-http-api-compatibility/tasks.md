## 1. Handler Contracts And Tests

- [x] 1.1 Add `internal/httpapi` package tests for fixed route precedence, method handling, unknown routes, and shared JSON error envelopes.
- [x] 1.2 Add state payload tests for counts, running/retrying arrays, token totals, rate limits, timestamp nullability, and snapshot timeout/unavailable envelopes.
- [x] 1.3 Add issue detail tests for running-only, retry-only, both-present, missing issue, workspace path fallback, attempts, logs, tracked, and recent-event inference.
- [x] 1.4 Add refresh tests for accepted refresh payloads, `requested_at` stamping, operations, nil refresh function, and refresh-unavailable errors.

## 2. HTTP API Implementation

- [x] 2.1 Implement `SnapshotFunc`, `RefreshFunc`, `RefreshResult`, sentinel errors, options, and `NewHandler`.
- [x] 2.2 Implement route handling using only `net/http` and `encoding/json`.
- [x] 2.3 Implement state DTO projection from `domain.Snapshot`.
- [x] 2.4 Implement issue detail DTO projection and exact identifier lookup from snapshot entries.
- [x] 2.5 Implement refresh DTO projection and error mapping using `errors.Is`.

## 3. Verification And Documentation

- [x] 3.1 Run `go test ./internal/httpapi/...` and fix any package-level failures.
- [x] 3.2 Run closure checks required by `workspace/T15/test_strategy.md`.
- [x] 3.3 Record residual non-goals or deferred integration work in `workspace/T15/todo.md`.
