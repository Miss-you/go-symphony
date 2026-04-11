# T14 Final Comparison

## Result

T14 now connects the Go runtime pieces into a complete internal run loop while preserving the V1 package boundaries from the design.

The implementation matches the relevant original Symphony behavior for this task:

- startup loads a workflow/config snapshot and performs terminal workspace cleanup before dispatch
- polling, dispatch, claims, retries, stale retry protection, reconcile, and snapshots remain owned by `internal/orchestrator`
- worker goroutines report `domain.RunEvent` facts instead of mutating scheduler state directly
- workspace creation, run hooks, cleanup intent, and runner-backed local/SSH behavior stay behind the workspace/runner packages
- Codex app-server sessions run first-turn prompts and continuation turns through the existing session boundary
- successful turns refresh the current item before continuing
- `agent.max_turns` exits as normal completion for still-active work
- failed and cancelled turns normalize as run failures
- memory-backed runs use no Linear workflow bundle, no Linear dynamic tools, and no provider network requirement
- Linear-backed runs use workflow-selected `linear_graphql` dynamic tool injection
- shutdown is idempotent across orchestrator, workers, Codex sessions, and owned config store resources

## Compatibility Notes

The Go implementation intentionally keeps provider-specific wiring out of `internal/orchestrator`. Linear selection and tool injection happen through `internal/cli` and `internal/workflow`; memory runs use an explicit no-network bundle.

Prompt rendering remains intentionally minimal. It supports the V1 issue placeholders and the default description conditional needed by the current workflow loader. A broader Solid-compatible template engine is deferred because T14 is the runtime assembly task, not the full workflow parity task.

Full user-facing CLI parity, HTTP API compatibility, terminal dashboard compatibility, and web dashboard compatibility remain scheduled as T15-T18.

## Verification Evidence

Fresh verification after review fixes:

```bash
go test -count=1 ./internal/orchestrator/... ./internal/cli/... ./cmd/symphony/...
go test -count=1 ./internal/...
go test -count=1 ./...
make build
make lint
make test-e2e
openspec validate --type change end-to-end-run-integration
git diff --check
```

All commands completed successfully.

Final validation after spec sync and archive:

```bash
make verify
openspec validate --specs
git diff --check
```

All commands completed successfully.
