## ADDED Requirements

### Requirement: Core runtime work items use a provider-neutral contract
The core domain model MUST define a provider-neutral `WorkItem` type for orchestration and prompt rendering. `WorkItem` MUST carry stable item identity, prompt/session-labeling fields, dispatch-relevant state, blocker references, routing eligibility, and prompt-visible timestamps and labels, while excluding provider config, provider payload fragments, tracker write state, and generic metadata escape hatches.

#### Scenario: Tracker adapter normalizes a provider item into the core domain
- **WHEN** a later tracker adapter converts provider data into `internal/domain.WorkItem`
- **THEN** the resulting type includes the approved prompt and orchestration fields needed by the runtime
- **AND** the type does not require any Linear-specific GraphQL payload fragments or tracker write semantics to exist in the core

#### Scenario: Routing eligibility is not explicitly denied
- **WHEN** a `WorkItem` has no explicit routing exclusion recorded in the domain model
- **THEN** the domain contract preserves a distinct “not explicitly denied” state instead of collapsing it into a false zero-value dispatch decision

### Requirement: Core runtime domain captures blockers, retry state, and polling state explicitly
The core domain model MUST define explicit `Blocker`, `RetryEntry`, and `PollingState` types. `Blocker` MUST remain a minimal identity-plus-state reference. `RetryEntry` MUST preserve item identity, attempt count, absolute retry schedule time, last error context, and workspace/host context. `PollingState` MUST preserve whether polling is currently in progress, the next scheduled poll instant, and the poll interval using Go-native timing types.

#### Scenario: Orchestrator exposes a blocked todo item
- **WHEN** a later orchestrator task evaluates a blocked `Todo` item
- **THEN** the core domain provides blocker identity and blocker state without requiring provider-specific relation payloads in the core model

#### Scenario: Observability needs retry queue timing
- **WHEN** later observability or compatibility-shell code projects retry information from the core domain
- **THEN** the domain model provides absolute retry schedule time and retry context that presenters can transform into countdown or timestamp views without reconstructing retry state from private orchestrator internals

### Requirement: Core runtime snapshot remains projection-only and complete enough for observability
The core domain model MUST define `ActiveRun`, `Snapshot`, `CodexTotals`, and rate-limit-related types that capture the running-item, retry-queue, polling, aggregate token, and rate-limit facts later observability layers need. The snapshot contract MUST exclude orchestrator-private process refs, timers, and other mutable runtime wiring.

#### Scenario: Later dashboard tasks project runtime state
- **WHEN** later API, terminal dashboard, or web dashboard code reads a runtime snapshot
- **THEN** the snapshot provides running-item state, retry state, polling state, aggregate Codex totals, and rate-limit data from one projection source
- **AND** the snapshot does not require those packages to read orchestrator-private mutable state directly

#### Scenario: Core snapshot stays free of mutable runtime wiring
- **WHEN** later tasks inspect the exported core snapshot types
- **THEN** they do not find process identifiers, timer refs, or monitor refs embedded in the provider-neutral domain model

### Requirement: Workers report through a stable tagged run-event contract
The core domain model MUST define a tagged `RunEvent` and closed `RunEventKind` vocabulary for worker-to-orchestrator reporting. The initial stable event set MUST cover workspace creation, workspace path discovery, runner host selection, Codex event receipt, turn completion, run completion, run failure, and retry scheduling. Workers MUST report through these events rather than mutating shared runtime state directly.

#### Scenario: Worker reports lifecycle progress
- **WHEN** a later worker implementation reports runtime progress back to the orchestrator
- **THEN** it emits one of the approved `RunEventKind` values with item identity and runtime context
- **AND** the event acts as an input to orchestrator-owned state transitions instead of mutating shared scheduling state directly

#### Scenario: New runtime package depends on the event contract
- **WHEN** a later runtime package compiles against the core event vocabulary
- **THEN** the exported event kinds are stable enough for that package to reason about workspace, runner, Codex, completion, failure, and retry lifecycle updates without inventing parallel event naming

### Requirement: Exported domain surface stays intentionally narrow
The `internal/domain` package MUST keep its stable exported surface centered on the task-approved domain vocabulary plus the concrete snapshot helper types already proven necessary by the landed implementation. Additional exported helper types MUST be introduced only when later packages demonstrably need to name them directly, and the contract tests MUST lock those additions in.

#### Scenario: Later task adds new helper types
- **WHEN** a later implementation task proposes exporting new helper types from `internal/domain`
- **THEN** that widening is justified by a direct downstream need rather than speculative future-proofing
- **AND** the contract tests are updated in the same change so the widened surface is deliberate
