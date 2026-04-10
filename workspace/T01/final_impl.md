# T01 Final Implementation

## Acceptance Summary

- Reviewers: 3 independent rubric reviews
- Average score: 91/100
- High-severity issues: none
- Accepted direction: `T01` is a contract-capture task only

## Final Decision

`T01 Compatibility Contract` will freeze the approved design's normative compatibility rules into durable OpenSpec artifacts. It will not create Go code, repo skeleton, or package layout.

## Scope

This task will:

- create one OpenSpec change: `freeze-compatibility-contract`
- define one capability: `compatibility-contract`
- promote the approved design's parity checklist, terminology mapping, non-goals, and core-versus-compatibility-shell boundary rules into stable spec language
- make the synced main spec at `openspec/specs/compatibility-contract/spec.md` the durable contract for downstream tasks

This task will not:

- introduce `cmd/`, `internal/`, `go.mod`, or other code-bearing bootstrap files
- add runtime implementations
- perform `T02 Repo Skeleton` work
- rewrite `README.md` beyond a tiny pointer if one becomes strictly necessary

## Contract Content To Freeze

### Parity surfaces

- `WORKFLOW.md` loading semantics
- default unattended Linear workflow behavior, including handoff semantics
- `WORKFLOW.md` hot reload and last-known-good behavior
- prompt rendering behavior
- tracker polling and reconciliation behavior
- workspace hook lifecycle
- local and SSH worker execution behavior
- Codex app-server protocol handling
- `linear_graphql` dynamic tool behavior
- Linear reader semantics and normalization
- HTTP API routes and payload semantics
- terminal dashboard rendering behavior
- web dashboard at `/`
- CLI bootstrap and acknowledgement behavior
- shutdown and offline dashboard rendering

### Terminology mapping

The spec must preserve the design's Symphony/Elixir-to-Go mapping as normative reference, including at minimum:

- `issue` -> `WorkItem`
- `tracker.kind` -> `provider`
- `linear_graphql` -> Linear ToolBridge capability

Compatibility surfaces may remain issue-centric even when core internals use provider-neutral terms.

### Boundary rules

- the orchestrator is the single mutable runtime state owner
- the core may depend on tracker read behavior, not generic tracker writes
- ticket writes may exist in workflow/runtime tooling, consistent with the root Symphony spec, but must not expand the Go core tracker interface
- provider-specific write behavior belongs in compatibility-shell tooling
- observability is projection-only and must not become a second runtime truth source

### Explicit non-goals

- no universal tracker write interface in core
- no universal workpad abstraction in core
- no fake provider-agnostic default workflow
- no second observability state machine
- no oversized provider-specific core model leakage
- no deep package fragmentation before behavior stabilizes
- no Lark task support in the first implementation phase

## Artifact Plan

Create and validate:

- `openspec/changes/archive/2026-04-10-freeze-compatibility-contract/proposal.md`
- `openspec/changes/archive/2026-04-10-freeze-compatibility-contract/design.md`
- `openspec/changes/archive/2026-04-10-freeze-compatibility-contract/tasks.md`
- `openspec/changes/archive/2026-04-10-freeze-compatibility-contract/specs/compatibility-contract/spec.md`
- `workspace/T01/test_strategy.md`

After sync/archive, the durable landing point must be:

- `openspec/specs/compatibility-contract/spec.md`

## Verification Gates

`T01` is complete only when all of the following are true:

1. `openspec validate compatibility-contract` passes in the landed repo state.
   The change-level validation `openspec validate freeze-compatibility-contract` was run pre-archive during task execution.
2. The change delta captures parity scope, terminology mapping, boundary rules, and explicit non-goals.
3. `openspec/specs/compatibility-contract/spec.md` exists after sync and matches the change contract.
4. The task board, workspace artifacts, and OpenSpec change name agree.
5. No `T02` code or skeleton files were introduced.

## Process Rule For Later Tasks

If any later task changes parity scope, terminology mapping, or core/shell boundaries, that same change must update `openspec/specs/compatibility-contract/spec.md`.
