## Purpose

Define the normative compatibility contract that downstream `go-symphony` tasks must preserve for parity scope, terminology mapping, and architecture boundaries.

## Requirements

### Requirement: Compatibility contract defines required parity surfaces
The repository MUST define a normative compatibility contract for `go-symphony` that freezes the user-facing parity surfaces approved for V1. That contract MUST include `WORKFLOW.md` loading semantics, default unattended Linear workflow behavior, workflow hot reload and last-known-good behavior, prompt rendering, tracker polling and reconciliation, workspace lifecycle hooks, local and SSH worker execution behavior, Codex app-server integration, `linear_graphql` behavior, HTTP API compatibility, terminal dashboard compatibility, web dashboard compatibility at `/`, CLI/bootstrap behavior, and shutdown/offline rendering behavior.

#### Scenario: Later task needs parity scope
- **WHEN** a later implementation task needs to know what Symphony behavior it must preserve
- **THEN** the repository contains a spec artifact that explicitly names the parity surfaces without requiring the task to reinterpret the approved design prose

### Requirement: Compatibility contract defines terminology mapping
The repository MUST define a normative terminology mapping between Symphony/Elixir vocabulary and Go internal vocabulary. That mapping MUST preserve the approved design's intent, including `issue` to `WorkItem`, `tracker.kind` to `provider`, and `linear_graphql` to a Linear ToolBridge capability, while allowing compatibility surfaces to remain issue-centric.

#### Scenario: Later task introduces core domain names
- **WHEN** a later task names provider-neutral core types or compatibility-surface DTOs
- **THEN** the repository contains a contract that makes the intended Symphony-to-Go term mapping explicit

### Requirement: Compatibility contract defines core and compatibility-shell boundaries
The repository MUST define normative boundary rules for the provider-neutral core and compatibility shell. Those rules MUST state that the orchestrator is the single mutable runtime state owner, the core depends on tracker read behavior rather than generic tracker writes, ticket writes may exist in workflow/runtime tooling without widening the core tracker interface, provider-specific write behavior belongs in compatibility-shell tooling, and observability remains projection-only rather than a second runtime truth source.

#### Scenario: Later task proposes new tracker or observability abstractions
- **WHEN** a later task proposes widening tracker interfaces or giving observability its own runtime state
- **THEN** the repository contains a contract that makes those boundary violations explicit

### Requirement: Compatibility contract defines explicit V1 non-goals
The repository MUST define the explicit V1 non-goals for `go-symphony`. Those non-goals MUST include no universal tracker write interface in core, no universal workpad abstraction in core, no fake provider-agnostic default workflow, no second observability state machine, no oversized provider-specific core model leakage, and no Lark task support in the first implementation phase.

#### Scenario: Later task scopes new abstractions
- **WHEN** a later task attempts to introduce behavior or abstractions that were explicitly excluded from V1
- **THEN** the repository contains a contract that names those exclusions as out of scope

### Requirement: Contract changes co-evolve with implementation scope changes
If a later change alters parity scope, terminology mapping, or the core-versus-compatibility-shell boundary, that same change MUST update the `compatibility-contract` spec in the repository.

#### Scenario: Later change revises compatibility rules
- **WHEN** a later implementation or design change modifies compatibility scope or boundary rules
- **THEN** that same change updates the `compatibility-contract` spec instead of leaving the contract stale
