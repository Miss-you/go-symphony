## 1. Runtime Assembly

- [x] 1.1 Add the thin `internal/cli` assembly boundary that reads `config.Store`, selects the workflow, builds the tracker reader, and wires the orchestrator without provider leakage.
- [x] 1.2 Add `cmd/symphony` bootstrap coverage that delegates startup and shutdown to the CLI boundary.
- [x] 1.3 Add startup cleanup wiring so terminal issue workspaces are removed before the first dispatch cycle begins.

## 2. Runtime Paths

- [x] 2.1 Implement the memory no-network bundle path with no dynamic tools and an unsupported-tool handler.
- [x] 2.2 Wire Linear workflow-selected tool injection through the compatibility shell and session startup path.
- [x] 2.3 Implement post-turn refresh and `max_turns` normal completion handling in the worker loop.
- [x] 2.4 Normalize Codex app-server events into stable runtime events and ensure turn completion is counted once.

## 3. Scheduling And Cleanup

- [x] 3.1 Preserve retry lineage and metadata for continuation retries, failure retries, and stale retry deliveries.
- [x] 3.2 Implement terminal cleanup intent handling versus non-terminal invalidation during reconciliation.
- [x] 3.3 Make shutdown idempotent across orchestrator, worker, Codex session, workspace, and config store teardown.

## 4. Verification

- [x] 4.1 Add behavior-level tests for startup cleanup, memory no-network runs, Linear injection, post-turn refresh, max-turn completion, and event normalization.
- [x] 4.2 Add regression tests for `config.Store` current snapshot use and bootstrap/shutdown delegation.
- [x] 4.3 Run the targeted Go test gates and record any remaining e2e limitations separately if they still apply.
