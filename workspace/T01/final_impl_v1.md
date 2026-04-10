# T01 Final Implementation V1

## Objective

Complete `T01 Compatibility Contract` by turning the approved design's normative compatibility rules into durable OpenSpec artifacts that later tasks can implement against without reinterpreting the design prose.

## Final Decision

`T01` is a contract-capture task, not a code or repo-skeleton task.

The task should:

- preserve the existing Symphony compatibility boundary from `SPEC.md` and the Elixir reference implementation
- promote the approved Go design's parity checklist, terminology mapping, and explicit non-goals into stable spec artifacts
- create a single OpenSpec change that covers only this contract freeze

The task must not:

- introduce `cmd/` or `internal/` package scaffolding
- add implementation code
- decide details that belong to `T02` or later runtime tasks
- rewrite `README.md` beyond a tiny pointer if one becomes strictly necessary

## Artifact Plan

### Workspace artifacts

- keep `workspace/T01/original_impl.md` as the source-system research memo
- keep `workspace/T01/new_impl.md` as the current-repo gap analysis
- revise this file into `workspace/T01/final_impl.md` after rubric review passes
- create `workspace/T01/test_strategy.md` for document/spec validation gates

### OpenSpec change

- Create change: `freeze-compatibility-contract`
- Scope the change to exactly one capability: `compatibility-contract`

Expected change artifacts before archive:

- `openspec/changes/freeze-compatibility-contract/proposal.md`
- `openspec/changes/freeze-compatibility-contract/design.md`
- `openspec/changes/freeze-compatibility-contract/tasks.md`
- `openspec/changes/freeze-compatibility-contract/specs/compatibility-contract/spec.md`

Archived repo paths after archive:

- `openspec/changes/archive/2026-04-10-freeze-compatibility-contract/proposal.md`
- `openspec/changes/archive/2026-04-10-freeze-compatibility-contract/design.md`
- `openspec/changes/archive/2026-04-10-freeze-compatibility-contract/tasks.md`
- `openspec/changes/archive/2026-04-10-freeze-compatibility-contract/specs/compatibility-contract/spec.md`

### Main spec after sync/archive

After `openspec-sync-specs` and `openspec-archive-change`, the normative contract should live at:

- `openspec/specs/compatibility-contract/spec.md`

That spec should become the durable source for later task work.

## What The Spec Must Freeze

### 1. Parity surfaces

The spec should explicitly preserve the approved parity checklist:

- `WORKFLOW.md` loading semantics
- default unattended Linear workflow behavior, including its state-machine-like handoff semantics
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

### 2. Terminology mapping

The spec should explicitly freeze the design's terminology mapping so later code tasks do not reintroduce provider-specific or Elixir-specific names into the core domain.

Minimum examples to preserve:

- `issue` -> `WorkItem`
- `tracker.kind` -> `provider`
- `linear_graphql` -> Linear ToolBridge capability
- issue state vocabulary remains valid on compatibility surfaces even if core internals use provider-neutral names

### 3. Core versus compatibility-shell boundary

The spec should make these rules normative:

- the orchestrator is the single mutable runtime state owner
- the provider-neutral core may depend on tracker read behavior, not generic tracker writes
- ticket writes may still exist in workflow/runtime tooling, matching the root Symphony spec boundary, but must not expand the Go core tracker interface
- provider-specific write behavior belongs in compatibility-shell tooling such as Linear ToolBridge
- observability is projection-only and must not become a second runtime truth source

### 4. Explicit V1 non-goals

The spec should carry forward the approved non-goals:

- no universal tracker write API in core
- no universal workpad abstraction in core
- no fake provider-agnostic default workflow
- no deep package fragmentation before behavior stabilizes
- no Lark task support in the first implementation phase

## Change Discipline

`T01` should also freeze one process rule for later tasks:

- if a later task changes parity scope, terminology mapping, or core/shell boundary rules, that same change must update `openspec/specs/compatibility-contract/spec.md`

This keeps contract drift visible in repo artifacts instead of chat history.

## Why This Is The Right Scope

- It is faithful to the old system because it captures the existing contract rather than inventing a new architecture.
- It is Go-native because it accepts the design's provider-neutral core and compatibility shell split without copying Elixir module boundaries.
- It avoids overdesign because it does not create code, package scaffolding, or generalized abstractions before `T02+`.
- It is testable because completion can be proven through repo artifacts and OpenSpec validation rather than subjective agreement.

## Verification Approach

`T01` should be considered complete only if all of the following are true:

1. `openspec validate compatibility-contract` passes in the landed repo state, and `openspec validate freeze-compatibility-contract` was run pre-archive during task execution.
2. The change spec captures parity scope, terminology mapping, boundary rules, and explicit non-goals.
3. The synced main spec exists under `openspec/specs/compatibility-contract/spec.md`, and it reflects the same contract as the change delta.
4. The task board, workspace artifacts, and OpenSpec change name all agree.
5. No `T02` repo-skeleton or code work slipped into this change, especially `cmd/`, `internal/`, `go.mod`, or other code-bearing bootstrap files.

## Explicit Out Of Scope

- `cmd/` or `internal/` package creation
- Go module initialization
- build/testable code scaffolding
- runtime implementations for workflow loading, config, domain, orchestrator, workspace, runner, or dashboards
- README marketing or setup expansion unrelated to the compatibility contract
