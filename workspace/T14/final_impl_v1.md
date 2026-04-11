# T14 Final Implementation Plan v1

## 1. Goal

Connect the already-built runtime pieces into an end-to-end Symphony run loop without changing their ownership boundaries.

T14 should make the Go port able to:

- drive `tracker.TrackerReader` candidates through the orchestrator
- create and reuse workspaces through `internal/workspace`
- start Codex app-server sessions through `internal/codex`
- inject workflow-selected dynamic tools into Codex without moving provider writes into core
- report worker facts back through `domain.RunEvent`
- preserve continuation retry, failure retry, stall recovery, and terminal cleanup
- prove both a no-network memory path and a Linear-backed path through behavior tests

The deliverable is the missing composition layer. It is not a new generic runtime framework.

## 2. Non-Goals

T14 does not:

- add a universal tracker write API
- add a universal workpad abstraction
- add a provider-agnostic default workflow
- add HTTP API, terminal dashboard, or web dashboard behavior
- put Linear GraphQL, workflow selection, or provider writes inside `internal/orchestrator`
- widen `internal/domain` with provider-specific fields
- redesign workspace, runner, tracker, or Codex package contracts
- implement full CLI option parity beyond the minimal executable bootstrap needed to start and stop the runtime

## 3. Symphony Parity Decisions

The implementation should preserve these Elixir runtime behaviors:

- startup terminal cleanup runs before the first poll can dispatch work
- orchestrator remains the only owner of mutable scheduling state
- candidate dispatch uses tracker listing plus refresh-by-ID before launch
- dispatch still respects active states, terminal states, blockers, routability, global concurrency, and per-state concurrency
- workspace creation keeps path safety, reuse, hook ordering, and terminal cleanup semantics
- `before_run` is hard-failing; `after_run` and `before_remove` stay best-effort
- Codex session flow stays `initialize` -> `thread/start` -> `turn/start` -> streamed events -> close
- `linear_graphql` remains the only Linear dynamic tool in this slice
- normal worker exit schedules continuation retry while the item may still be active
- failure and stall exits schedule failure retry with existing exponential backoff
- terminal-state reconciliation removes workspaces, while non-terminal invalidation only stops the run

Two boundaries are explicit:

- Linear production runs use `workflow.Select`, which selects `compat_linear_default` and injects `linear_graphql`.
- Memory runs are no-network closure tests. They use an explicitly injected runtime bundle with a prompt renderer, no dynamic tools, and an unsupported-tool handler. They must not call `workflow.CompatLinearDefaultBundle`, must not create a real Linear HTTP client, and must not advertise `linear_graphql`.

This keeps memory useful for local/test closure without inventing a provider-agnostic workflow family.

## 4. Proposed Go Shape and Package Boundaries

Use `internal/cli` as the process assembly boundary and keep `cmd/symphony/main.go` thin.

`cmd/symphony`:

- delegates to an `internal/cli` entrypoint
- does not own runtime policy
- has a narrow test seam proving main calls the entrypoint and propagates exit status

`internal/cli`:

- owns `config.Store` lifecycle, preserving last-known-good reload behavior
- reads the current workflow/settings snapshot before startup cleanup and before creating each new worker
- chooses the tracker reader
- selects the Linear workflow bundle for Linear settings
- accepts an explicit injected no-network bundle for memory tests/local closure
- creates the workspace manager
- creates and closes the orchestrator service
- owns worker handles, cancellation, Codex session close, and workspace cleanup calls

`internal/orchestrator`:

- keeps scheduling, retries, stall recovery, reconcile, and snapshots
- exports only the minimal service contract needed by `internal/cli`
- does not import `internal/workflow`, `internal/toolbridge/linear`, or provider-specific tracker packages

The exported orchestrator surface should be small:

- `Start(settings, deps)` or equivalent constructor
- `Snapshot() domain.Snapshot`
- `RequestRefresh()`
- `ApplyRunEvent(domain.RunEvent)`
- `Close()`

The dependency surface should remain callback-oriented around the already-proven private shape: list candidates, refresh by IDs, start run, stop run. Do not expose private maps, timers, retry entries, or process handles.

## 5. Runtime Flow

Startup order:

1. Create `config.Store` from the selected `WORKFLOW.md` path.
2. Read the current last-known-good workflow and settings from the store.
3. Build the tracker reader:
   - Linear settings create `trackers/linear.Reader`.
   - Memory closure uses `trackers/memory.Reader` through explicit dependency injection in tests/local harnesses.
4. Build the runtime bundle:
   - Linear settings call `workflow.Select`.
   - Memory closure receives a no-network bundle from the caller/test.
5. Create `workspace.Manager`.
6. Run startup terminal cleanup:
   - call `reader.ListByStates(ctx, settings.Provider.TerminalStates)`
   - call `workspace.RemoveIssueWorkspaces(identifier, "")` for each returned item
   - finish this sweep before constructing or starting the orchestrator timers
7. Start the orchestrator with tracker callbacks and `startRun` / `stopRun` closures.
8. Block until context cancellation, then close the orchestrator and all active worker handles idempotently.

`startRun` behavior:

- return quickly to the orchestrator with an opaque handle
- start a worker goroutine owned by `internal/cli`
- emit `RunEventRunnerHostSelected`
- create/reuse the workspace and emit `RunEventWorkspaceCreated` and `RunEventWorkspacePathDiscovered`
- run workspace hooks through `workspace.RunWithHooks`
- start a Codex session with the current settings and selected bundle
- run Codex turns until the post-turn refresh says the item is inactive/missing, a failure occurs, or `agent.max_turns` is reached
- emit only `domain.RunEvent` facts back to orchestrator

`stopRun` behavior:

- idempotently cancel the worker context
- close the active Codex session if it has started
- remove the workspace only when `CleanupWorkspace=true`
- leave workspace contents intact for non-terminal invalidation and failure retry

The runtime handle may track cancellation, session pointer, worker host, workspace path, and a done channel. It must not own scheduling truth.

## 6. Turn Loop and Max-Turn Semantics

Each worker run uses one Codex session and up to `settings.Agent.MaxTurns` turns.

Turn loop:

1. Render the first turn from the current workflow prompt template and current `WorkItem`.
2. Run `codex.Session.RunTurn`.
3. If the turn fails, is cancelled, times out, or returns an error, emit `RunEventRunFailed`.
4. If the turn completes, emit `RunEventTurnCompleted` with the returned usage totals.
5. Refresh the item by ID through `TrackerReader.RefreshByIDs`.
6. If the item is missing or no longer active, emit `RunEventRunCompleted` and exit normally. The orchestrator continuation retry will revalidate and release or clean up according to its existing rules.
7. If the item is still active and the turn count is below `agent.max_turns`, run the next turn with Symphony's continuation prompt.
8. If the item is still active and `agent.max_turns` has been reached, emit `RunEventRunCompleted` and exit normally. This deliberately keeps the claim and lets the orchestrator schedule continuation retry attempt `1`, matching the Elixir "return control to orchestrator" behavior.

The max-turn cap is not a terminal item state, not a failure, and not a claim release. It is normal completion followed by orchestrator-owned continuation retry.

## 7. Prompt, Codex, and Tool Wiring

Prompt rendering:

- use the loaded workflow prompt template for the first turn
- preserve the existing `issue.*` template variable names as a compatibility detail at the assembly edge
- support the default template's description conditional and common scalar fields needed by existing workflow prompts
- use the hardcoded continuation prompt for turns `2..max_turns`

Codex session config:

- base config comes from `codex.ConfigFromSettings(currentSettings)`
- `DynamicTools` comes from the selected runtime bundle
- `ToolHandler` comes from the selected runtime bundle
- `NonInteractive=true`
- `TurnSandboxPolicy` and approval policy come from current settings
- `TransportFactory` is injectable for tests and defaults to `codex.StartProcessTransport`

Tool behavior:

- Linear runtime bundle passes `linear_graphql` specs and handler straight through from `internal/workflow` / `internal/toolbridge/linear`
- memory runtime bundle passes no dynamic tools and a handler that returns `codex.ErrUnsupportedTool`
- runtime glue never reinterprets GraphQL arguments or provider write payloads

## 8. Event Normalization Matrix

The worker should emit these `domain.RunEvent` values:

| Source | Domain event | Notes |
| --- | --- | --- |
| worker admitted to local/SSH host | `RunEventRunnerHostSelected` | Carries `WorkerHost`. |
| workspace created or reused | `RunEventWorkspaceCreated` | Carries `WorkspacePath`; message may distinguish created vs reused. |
| workspace path known | `RunEventWorkspacePathDiscovered` | Carries `WorkspacePath`. |
| `codex.EventSessionStarted` | `RunEventCodexEventReceived` | Carries `SessionID=ThreadID`, message `session_started`. |
| `codex.EventApprovalAnswered` | `RunEventCodexEventReceived` | Message includes `approval_answered` and method. |
| `codex.EventToolInputAnswered` | `RunEventCodexEventReceived` | Message `tool_input_answered`. |
| `codex.EventToolCallCompleted` | `RunEventCodexEventReceived` | Message `tool_call_completed:<method-or-tool>`. |
| `codex.EventToolCallFailed` | `RunEventCodexEventReceived` | Message `tool_call_failed:<method-or-tool>`. |
| `codex.EventUnsupportedToolCall` | `RunEventCodexEventReceived` | Message `unsupported_tool_call`. |
| `codex.EventMalformedMessage` | `RunEventCodexEventReceived` | Message `malformed_message`; does not fail the run by itself. |
| `codex.EventUnknownMessage` | `RunEventCodexEventReceived` | Message `unknown_message:<method>`. |
| successful `RunTurn` return | `RunEventTurnCompleted` | Exactly one per completed turn; carries cumulative turn usage. |
| failed/cancelled/timeout turn | `RunEventRunFailed` | Carries stable error message. |
| session bootstrap, workspace hook, prompt render, or refresh error | `RunEventRunFailed` | Carries stable error message and context. |
| normal worker exit after inactive/missing item or max-turn cap | `RunEventRunCompleted` | Lets orchestrator schedule continuation retry. |

Do not emit both `RunEventTurnCompleted` from the Codex event sink and from the `RunTurn` result. The `RunTurn` result is the single source for turn count and token totals to avoid double counting.

Rate-limit updates remain attached to `RunEventCodexEventReceived` when present in Codex payloads. Token usage remains attached to `RunEventTurnCompleted`, because `codex.RunTurn` returns normalized usage.

## 9. Retry and Cleanup Integration

Retry remains orchestrator-owned:

- normal run completion schedules continuation retry attempt `1`
- failure run completion schedules failure retry using existing backoff
- stale retry deliveries remain ignored by scheduler nonce
- workers may emit metadata events, but they never mutate retry maps

Cleanup remains orchestrator-requested and workspace-executed:

- startup cleanup removes terminal workspaces before dispatch can happen
- terminal reconcile calls `stopRun(... CleanupWorkspace=true)`
- retry revalidation for terminal items also requests cleanup
- non-terminal invalidation calls `stopRun(... CleanupWorkspace=false)`
- repeated close/cleanup calls are idempotent

The runtime worker should always close the Codex session and run `after_run` once the run body exits. It should remove the workspace only through `stopRun` cleanup intent or startup cleanup, not after every normal turn loop.

## 10. TDD Implementation Plan

Write tests first in this order:

1. Orchestrator exported service seam
   - failing test proves `internal/cli` can construct, refresh, snapshot, apply events, and close through exported APIs only
   - no private state access from `internal/cli`

2. Startup cleanup before dispatch
   - seed terminal items through a reader
   - create matching workspaces
   - prove cleanup completes before the first candidate dispatch can start

3. Memory no-network end-to-end run
   - use `trackers/memory.Reader`
   - inject a no-network bundle with no dynamic tools
   - use temp workspace root and scripted Codex transport
   - prove candidate -> workspace -> Codex session -> turn completion -> continuation retry
   - assert no Linear HTTP client or `linear_graphql` tool is used

4. Post-turn refresh and max-turn behavior
   - prove refresh-by-ID happens after each completed turn
   - active item below max turns gets a continuation prompt in the same session
   - active item at max turns exits as normal `RunEventRunCompleted`
   - inactive or missing item exits as normal `RunEventRunCompleted` and lets orchestrator release/cleanup through retry revalidation

5. Linear-backed end-to-end run
   - use `trackers/linear.Reader` with fake GraphQL client
   - use `workflow.Select` for `compat_linear_default`
   - use scripted Codex transport
   - prove `thread/start` advertises `linear_graphql`
   - prove a full turn reaches `RunEventRunCompleted` without provider writes entering core

6. Event normalization regression
   - feed session start, approval, tool call success/failure, unsupported tool, malformed, unknown, completed, failed, and cancelled events
   - assert exact `domain.RunEventKind`, message, session ID, totals, and rate-limit projection behavior

7. Retry and cleanup regression
   - continuation retry redispatches with retained worker host/workspace metadata
   - failure retry preserves last error and attempt lineage
   - terminal cleanup removes workspace
   - non-terminal invalidation closes session but preserves workspace

8. Bootstrap and shutdown
   - prove `cmd/symphony/main.go` delegates to `internal/cli`
   - prove runtime close is idempotent and cancels active handles
   - prove `config.Store` is closed

## 11. Verification Plan

Primary targeted gate:

```bash
go test ./internal/orchestrator/... ./internal/cli/... ./cmd/symphony/...
```

Broader integration gate:

```bash
go test ./internal/...
go test ./...
make build
make lint
make test-e2e
openspec validate --type change end-to-end-run-integration
```

The acceptance evidence must include behavior assertions for:

- memory no-network run
- Linear bundle/tool advertisement
- post-turn refresh
- max-turn normal completion
- event normalization
- continuation retry
- terminal cleanup
- startup cleanup before dispatch
- idempotent shutdown

If `make test-e2e` remains low-signal or unavailable, record the exact reason in `workspace/T14/todo.md` and keep the package-level behavior tests as the blocking proof.

## 12. Risks and Deferred Work

Risks:

- runtime glue can accidentally become a second scheduler if handle state starts making retry decisions
- prompt rendering can drift from Elixir/Solid behavior if it grows beyond the template variables actually needed now
- hot reload can be partially preserved but still not update every scheduler field until later CLI/full parity work
- Codex protocol changes may require later event mapping additions

Deferred work:

- HTTP API compatibility
- terminal dashboard compatibility
- web dashboard
- full CLI flags, acknowledgement text, and shutdown rendering
- richer observability projection packaging
- any non-Linear workflow bundle

T14 is done when the runtime glue exists, memory and Linear paths both run end to end, and retry/cleanup behavior is proven by fresh tests without violating core/provider boundaries.
