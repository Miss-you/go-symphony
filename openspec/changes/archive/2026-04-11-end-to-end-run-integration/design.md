## Context

T14 is the runtime assembly task for `go-symphony`. The repository already has the provider-neutral core packages, the compatibility shell pieces, and the workflow/config contracts, but it does not yet have a single end-to-end process boundary that proves those parts can be composed into one Symphony-compatible run loop.

The key constraint is compatibility-first wiring:

- keep provider-neutral state ownership in the core runtime
- keep provider-specific tool injection and workflow selection in the compatibility shell
- keep `cmd/symphony` thin
- treat `config.Store` as the source of the current snapshot, not as a reparsed map
- allow the memory path to run with no network traffic

## Goals / Non-Goals

**Goals:**

- assemble a complete runtime from existing packages without moving ownership boundaries
- make startup cleanup deterministic before the first dispatch cycle
- make the memory path explicitly no-network for local and test verification
- make Linear tool injection follow the selected workflow
- define post-turn refresh, max-turn completion, retry lineage, and event normalization precisely enough to test
- define idempotent shutdown and config-store lifecycle semantics

**Non-Goals:**

- no new provider-agnostic tracker write API
- no new workpad abstraction
- no HTTP API, terminal dashboard, or web dashboard behavior
- no Lark-specific runtime behavior
- no moving provider-specific behavior into `internal/orchestrator`
- no broader CLI parity beyond the minimal bootstrap and teardown needed to run the loop

## Decisions

1. Use `internal/cli` as the process assembly boundary.
   `cmd/symphony/main.go` should only parse process-level concerns and hand off to `internal/cli`. This keeps startup, shutdown, and runtime dependency injection in one place.
   Alternatives considered: pushing assembly into `cmd/symphony` or dispersing it across runtime packages. Those options make the lifecycle harder to test and blur ownership.

2. Keep `internal/orchestrator` as the mutable state owner and keep workers event-driven.
   Workers report `domain.RunEvent` facts and the orchestrator remains the sole owner of claims, retries, and scheduling truth.
   Alternatives considered: letting the worker decide retry state or reusing shared mutable runtime maps. That would duplicate scheduling authority and make the retry path harder to reason about.

3. Treat memory runs as an explicit no-network bundle.
   The compatibility shell should inject a bundle with no dynamic tools and an unsupported-tool handler so local verification cannot accidentally fall back to real Linear HTTP behavior.
   Alternatives considered: reusing the normal provider bundle with a mock transport layer. That is more fragile because a missed branch can still reach the network.

4. Make Linear wiring workflow-selected.
   The runtime should ask `internal/workflow` which compatibility tools and dynamic capabilities apply, then inject the Linear bridge only when that workflow selects it.
   Alternatives considered: hard-wiring Linear tool names in the orchestrator or runtime core. That would leak provider concerns into the wrong layer.

5. Define post-turn refresh as the authoritative continuation check.
   After each successful turn, the worker refreshes the current work item before deciding whether to continue. That keeps max-turn completion, missing-item exits, and active continuation behavior testable.
   Alternatives considered: checking continuation only from the pre-turn snapshot. That can miss a terminal transition that happened during the turn.

6. Normalize Codex events at the worker boundary.
   The worker should convert Codex app-server events into runtime events in one place so the tests can assert stable event categories without depending on transport detail.
   Alternatives considered: exposing raw Codex transport events to the rest of the runtime. That would make the observable contract brittle.

7. Close resources idempotently and in reverse-lifecycle order.
   Shutdown must tolerate duplicate calls and close the active session, worker activity, orchestrator, and config store without double-free behavior.
   Alternatives considered: making shutdown callers coordinate exact one-time semantics. That is error-prone in signal and cancellation paths.

## Risks / Trade-offs

- [Risk] A thin assembly layer can become a second scheduler if it starts taking retry decisions. → Keep scheduling truth in `internal/orchestrator` and limit `internal/cli` to wiring and lifecycle management.
- [Risk] Memory no-network behavior can drift if the bundle shape changes later. → Keep the no-network path explicit and test it with a fixture that fails on any unsupported tool call.
- [Risk] Hot reload and current snapshot handling can diverge if the runtime re-reads raw config in multiple places. → Pass typed current snapshots through the store boundary and avoid re-parsing in worker construction.
- [Risk] Event normalization can lose detail if it is too coarse. → Preserve enough normalized metadata for test assertions while keeping the runtime contract stable.
- [Risk] Shutdown cleanup can accidentally remove workspaces that should be retained after non-terminal exits. → Gate workspace removal on the orchestrator's cleanup intent and the terminal-cleanup rules already expressed in the spec.

## Migration Plan

This change is internal and does not require a user-facing migration.

1. Add the OpenSpec contract for end-to-end runtime assembly.
2. Implement the runtime wiring under the existing package boundaries.
3. Verify the no-network memory path and the Linear path separately.
4. Keep the task board and workspace artifacts aligned with the OpenSpec change while implementation lands.

## Open Questions

None that block this change. Later CLI parity work can extend the same assembly boundary, but it does not change the T14 contract.
