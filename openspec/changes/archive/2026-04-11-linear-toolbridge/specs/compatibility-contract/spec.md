## MODIFIED Requirements

### Requirement: Compatibility contract defines required parity surfaces
The repository MUST define a normative compatibility contract for `go-symphony` that freezes the user-facing parity surfaces approved for V1. That contract MUST include `WORKFLOW.md` loading semantics, default unattended Linear workflow behavior, workflow hot reload and last-known-good behavior, prompt rendering, tracker polling and reconciliation, workspace lifecycle hooks, local and SSH worker execution behavior, Codex app-server integration, `linear_graphql` behavior, Linear ToolBridge write-helper behavior for comment creation and issue state updates, HTTP API compatibility, terminal dashboard compatibility, web dashboard compatibility at `/`, CLI/bootstrap behavior, and shutdown/offline rendering behavior.

#### Scenario: Later task needs parity scope
- **WHEN** a later implementation task needs to know what Symphony behavior it must preserve
- **THEN** the repository contains a spec artifact that explicitly names the parity surfaces without requiring the task to reinterpret the approved design prose

### Requirement: Compatibility contract defines core and compatibility-shell boundaries
The repository MUST define normative boundary rules for the provider-neutral core and compatibility shell. Those rules MUST state that the orchestrator is the single mutable runtime state owner, the core depends on tracker read behavior rather than generic tracker writes, ticket writes may exist in workflow/runtime tooling without widening the core tracker interface, provider-specific write behavior belongs in compatibility-shell tooling, Linear ToolBridge helpers do not expand `internal/tracker`, and observability remains projection-only rather than a second runtime truth source.

#### Scenario: Later task proposes new tracker or observability abstractions
- **WHEN** a later task proposes widening tracker interfaces or giving observability its own runtime state
- **THEN** the repository contains a contract that makes those boundary violations explicit
