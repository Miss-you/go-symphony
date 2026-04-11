## ADDED Requirements

### Requirement: Orchestrator uses runner host selection without yielding state ownership
The orchestrator SHALL use runner host selection for worker-host admission while retaining sole ownership of mutable runtime state.

The orchestrator MUST derive host-load inputs from its private running-state map and pass those inputs to runner host selection. Runner MUST NOT read or mutate orchestrator state. Total concurrency and state-specific dispatch gates MUST remain orchestrator-owned.

#### Scenario: Host loads come from orchestrator state
- **WHEN** the orchestrator evaluates a candidate for dispatch
- **THEN** it derives current host loads from orchestrator-owned running entries
- **AND** passes those loads into runner host selection

#### Scenario: Runner returns only an admission decision
- **WHEN** runner host selection returns a selected host
- **THEN** the orchestrator records the selected host as runtime metadata
- **AND** runner does not own claims, running entries, retry entries, or cleanup intent

#### Scenario: Capacity rejection preserves orchestrator scheduling
- **WHEN** runner host selection rejects admission because every configured host is at per-host capacity
- **THEN** the orchestrator does not dispatch a new run for that candidate in the current admission attempt
- **AND** existing orchestrator retry and polling semantics remain the scheduling authority
