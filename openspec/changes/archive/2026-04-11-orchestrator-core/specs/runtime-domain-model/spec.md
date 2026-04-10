## MODIFIED Requirements

### Requirement: Core runtime snapshot remains projection-only and complete enough for observability
The core domain model MUST define `ActiveRun`, `Snapshot`, `CodexTotals`, and rate-limit-related types that capture the running-item, retry-queue, polling, aggregate token, and rate-limit facts later observability layers need. The snapshot contract MUST exclude orchestrator-private process refs, timers, and other mutable runtime wiring.

#### Scenario: Later dashboard tasks project runtime state
- **WHEN** later API, terminal dashboard, or web dashboard code reads a runtime snapshot
- **THEN** the snapshot provides running-item state, retry state, polling state, aggregate Codex totals, and rate-limit data from one projection source
- **AND** the snapshot does not require those packages to read orchestrator-private mutable state directly

#### Scenario: Core snapshot stays free of mutable runtime wiring
- **WHEN** later tasks inspect the exported core snapshot types
- **THEN** they do not find process identifiers, timer refs, or monitor refs embedded in the provider-neutral domain model

#### Scenario: Orchestrator keeps private scheduling wiring out of the domain model
- **WHEN** the later `internal/orchestrator` task stores claims, stale-delivery guards, completion bookkeeping, or opaque run handles privately
- **THEN** those private scheduler details remain outside the exported `domain.Snapshot`, `domain.ActiveRun`, and `domain.RetryEntry` types

### Requirement: Workers report through a stable tagged run-event contract
The core domain model MUST define a tagged `RunEvent` and closed `RunEventKind` vocabulary for worker-to-orchestrator reporting. The initial stable event set MUST cover workspace creation, workspace path discovery, runner host selection, Codex event receipt, turn completion, run completion, run failure, and retry scheduling. Workers MUST report through these events rather than mutating shared runtime state directly.

#### Scenario: Worker reports lifecycle progress
- **WHEN** a later worker implementation reports runtime progress back to the orchestrator
- **THEN** it emits one of the approved `RunEventKind` values with item identity and runtime context
- **AND** the event acts as an input to orchestrator-owned state transitions instead of mutating shared scheduling state directly

#### Scenario: New runtime package depends on the event contract
- **WHEN** a later runtime package compiles against the core event vocabulary
- **THEN** the exported event kinds are stable enough for that package to reason about workspace, runner, Codex, completion, failure, and retry lifecycle updates without inventing parallel event naming

#### Scenario: Orchestrator updates runtime truth only from run events
- **WHEN** the later orchestrator task receives workspace, runner, Codex, turn, completion, failure, or retry notifications from workers
- **THEN** it updates private running, retry, and aggregate runtime state from `domain.RunEvent`
- **AND** it does not require workers to write directly into shared mutable scheduler state

#### Scenario: Retry notifications remain event-only inputs
- **WHEN** a later worker reports `RunEventRetryScheduled`
- **THEN** the event acts as worker-reported metadata only
- **AND** the worker still does not gain direct write access to shared retry bookkeeping
