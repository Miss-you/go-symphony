## 1. Orchestrator State And Service Skeleton

- [x] 1.1 Add the `internal/orchestrator` state model, normalized scheduler settings, and package-private collaborator seams for candidate refresh, run start/stop, host admission hints, and clock/timer control.
- [x] 1.2 Add the orchestrator service shell and snapshot projection helpers so private running/retry/poll state can project into deterministic `domain.Snapshot` values.

## 2. Scheduler Behavior

- [x] 2.1 Implement poll cadence, immediate startup poll, refresh coalescing, and stale tick / retry-delivery guards.
- [x] 2.2 Implement deterministic candidate ordering, dispatch gating, and revalidation-before-dispatch behavior from refreshed `domain.WorkItem` state.
- [x] 2.3 Implement worker event application, continuation versus failure retry lineage, running-item reconcile, and stall recovery semantics.

## 3. Contract Tests And Verification

- [x] 3.1 Add package-scoped orchestrator tests that lock startup checking/refresh coalescing, sorting, gating, retry progression, claim retention, terminal-versus-nonterminal cleanup intent, reconcile, stall, aggregate totals, rate-limit, `retry_scheduled` metadata-only handling, and snapshot projection behavior.
- [x] 3.2 Run `go test ./internal/orchestrator/...`, then broader verification (`go test ./...`, `make build`, `make lint`, and `make test-e2e` or an explicit applicability note) and record the evidence in `workspace/T06/`.
