## Context

T08 follows T07. `internal/workspace` now owns path safety, hook ordering, create/reuse/remove behavior, and terminal cleanup fan-out. It still contains private command execution mechanics that are only sufficient for local hooks and placeholder remote handling.

The approved design assigns local and SSH execution behavior to `internal/runner`, not `workspace`. The current orchestrator already carries worker-host facts and a private admission seam, but host selection still needs a concrete runner-side selector that can preserve Elixir per-host capacity behavior without moving mutable runtime state out of orchestrator.

## Goals / Non-Goals

**Goals:**

- Add a real `internal/runner` boundary for local and SSH command execution.
- Preserve one command execution contract for local and SSH paths.
- Add a pure host selector that uses configured SSH hosts, optional per-host capacity, preferred host, and caller-supplied host-load data.
- Refactor workspace so it keeps lifecycle policy but delegates command execution to runner.
- Wire orchestrator admission through the runner selector while keeping running-state ownership in orchestrator.

**Non-Goals:**

- Do not implement Codex app-server session lifecycle, protocol parsing, tool calls, or approval flow.
- Do not add a long-lived runner scheduler, mutable host pool, or generic remote orchestration framework.
- Do not move workspace path derivation, hook policy, or cleanup decisions into runner.
- Do not change tracker read/write contracts or provider workflow behavior.

## Decisions

1. **Use one runner executor contract for local and SSH commands.**
   - Rationale: Workspace hooks, remote lifecycle commands, and later Codex launch code all need to run a shell command on either the local machine or a selected host. A small `Executor.RunCommand(ctx, CommandRequest)` contract avoids duplicating local/SSH transport details.
   - Alternatives considered:
     - Keep command execution private in workspace. Rejected because T08 exists to prevent SSH/local execution mechanics from leaking into workspace.
     - Add Codex-specific process APIs now. Rejected because T09 owns the app-server protocol and session lifecycle.

2. **Keep host selection pure and stateless.**
   - Rationale: The orchestrator is the only owner of mutable running state. Runner can own the deterministic selection algorithm, but it must receive host-load data from the orchestrator instead of storing active runs itself.
   - Alternatives considered:
     - A runner-managed host pool. Rejected because it would create a second runtime state owner.
     - Leaving host selection only in orchestrator. Rejected because per-host SSH selection is execution-host policy and should be reusable without exposing SSH details to orchestrator.

3. **Workspace keeps lifecycle policy and delegates only execution.**
   - Rationale: `workspace` should continue to decide what create/remove/hook operation is being performed, including local path validation and remote lifecycle shell script content. Runner only decides how to execute a command on a host.
   - Alternatives considered:
     - Move `EnsureWorkspace` and `RemoveWorkspace` into runner. Rejected because that duplicates workspace lifecycle ownership.
     - Keep private workspace transport forever. Rejected because it would keep local/SSH mechanics in the wrong package.

4. **Wire orchestrator admission through runner selection in this task.**
   - Rationale: A selector that is tested but not used would not complete T08. The existing admission seam should call runner selection with host loads derived from `running`, while total and state-specific dispatch gates remain in orchestrator.
   - Alternatives considered:
     - Defer wiring to T14. Rejected because the T08 boundary would be partially dormant.

## Risks / Trade-offs

- [Risk] SSH quoting can diverge from Elixir behavior. → Mitigation: test argv construction, config-file handling, port parsing, bracketed IPv6, and remote `bash -lc` wrapping without real SSH.
- [Risk] Workspace could drift back into transport ownership while building remote lifecycle commands. → Mitigation: keep runner responsible for local/SSH execution and test that workspace uses an injected executor for commands.
- [Risk] Runner selector could accidentally own runtime state. → Mitigation: make selection pure and require loads as inputs.
- [Risk] Orchestrator changes could widen beyond host admission. → Mitigation: keep the change to host-load projection and the existing admission seam.

## Migration Plan

1. Add runner tests that fail against the placeholder package.
2. Implement runner executor, SSH command builder, and pure host selector.
3. Refactor workspace command execution to call runner while preserving T07 behavior.
4. Wire orchestrator admission through runner selection with orchestrator-owned host loads.
5. Run runner, workspace, orchestrator, full test, build, lint, and e2e command gates.

Rollback is straightforward because this is an internal package boundary: revert the change and workspace returns to its previous private transport implementation.

## Open Questions

None that block implementation. Codex process/session details intentionally remain for T09.
