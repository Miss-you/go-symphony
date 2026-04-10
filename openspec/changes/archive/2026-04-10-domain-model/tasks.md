## 1. Core Domain Types

- [x] 1.1 Add the exported provider-neutral domain types for work items, blockers, active runs, retry entries, polling state, snapshots, Codex totals, and rate limits under `internal/domain`.
- [x] 1.2 Freeze the initial `RunEventKind` set and the tagged `RunEvent` envelope using the approved worker-reporting vocabulary and Go-native time fields.

## 2. Contract-Locking Tests

- [x] 2.1 Add package-scoped tests that assert the exported `internal/domain` contract shape and boundary rules, including prompt-relevant `WorkItem` fields and the absence of provider-specific leakage.
- [x] 2.2 Run `go test ./internal/domain/...` and fix the domain package until the contract suite passes.

## 3. Verification And Task Sync

- [x] 3.1 Run `go test ./...`, `make build`, and `make lint` after the domain package lands.
- [x] 3.2 Run `make test-e2e` or record its exact applicability limits in `workspace/T05/todo.md`, then sync fresh verification evidence back into the task board and workspace artifacts.
