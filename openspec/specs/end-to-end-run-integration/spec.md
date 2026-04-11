## Purpose

Define the runtime assembly contract for composing tracker readers, workspace lifecycle, runner host execution, Codex app-server sessions, workflow-selected tools, and orchestrator scheduling into complete Symphony-compatible runs.

## Requirements

### Requirement: Runtime assembly keeps provider-specific wiring out of the orchestrator
The runtime SHALL assemble the Symphony execution loop through a thin process boundary that uses `config.Store`, the tracker reader, the workspace manager, the orchestrator, the workflow selector, and the Codex app-server session layer without leaking provider-specific wiring into `internal/orchestrator`.

#### Scenario: Startup assembles from the current snapshot
- **WHEN** the runtime process starts
- **THEN** it loads the current typed config snapshot from `config.Store`
- **AND** it builds the runtime from that snapshot instead of reparsing raw workflow data inside the worker or orchestrator

#### Scenario: Orchestrator does not import provider-specific wiring
- **WHEN** the runtime is built for dispatch
- **THEN** provider-specific tracker or workflow packages remain outside `internal/orchestrator`
- **AND** the orchestrator receives only the provider-neutral inputs it needs to schedule work

### Requirement: Startup cleanup completes before first dispatch
The runtime SHALL perform terminal workspace cleanup before the first dispatch cycle begins.

#### Scenario: Terminal workspaces are swept before dispatch
- **WHEN** startup discovers issue workspaces that belong to terminal items
- **THEN** the runtime removes those workspaces before the orchestrator starts polling for new work

#### Scenario: Cleanup failures stop startup
- **WHEN** startup workspace cleanup fails
- **THEN** the runtime does not begin dispatch
- **AND** the failure is surfaced as a startup error

### Requirement: Memory runs execute with no network access
The runtime SHALL provide a memory execution path that uses an explicit no-network bundle and does not fall back to a live Linear HTTP client.

#### Scenario: Memory run uses an unsupported-tool handler
- **WHEN** the runtime starts a memory-backed run
- **THEN** it injects a no-network bundle with no dynamic tools
- **AND** unsupported tool usage is reported through the runtime event stream instead of reaching a live provider client
- **AND** the runtime does not construct the Linear workflow bundle, Linear bridge, or Linear HTTP client for that memory run

#### Scenario: Memory run completes without outbound provider traffic
- **WHEN** a memory-backed run executes successfully
- **THEN** no provider network traffic is required for the run to complete
- **AND** the runtime remains valid for local verification without external dependencies

### Requirement: Linear runs receive workflow-selected tool injection
The runtime SHALL inject Linear workflow/tool capabilities only when the selected workflow requires them.

#### Scenario: Linear workflow enables Linear dynamic tools
- **WHEN** the selected workflow is Linear-backed
- **THEN** the runtime injects the Linear bridge and the workflow-selected tool set
- **AND** the worker can advertise `linear_graphql` through the session start path when that workflow requires it

#### Scenario: Non-Linear runs do not receive Linear tools
- **WHEN** the selected workflow is not Linear-backed
- **THEN** the runtime does not inject Linear dynamic tools
- **AND** the runtime stays on the provider-neutral path

### Requirement: Workers refresh after every completed turn
The runtime SHALL refresh the current item after each completed turn before deciding whether to continue, and it SHALL treat `max_turns` as a normal completion boundary when the item is still active.

#### Scenario: Active item continues before max turns
- **WHEN** a worker completes a turn and the refreshed item is still active
- **AND** the turn count is below `agent.max_turns`
- **THEN** the worker starts the next turn with continuation guidance

#### Scenario: Active item at max turns exits normally
- **WHEN** a worker completes a turn and the refreshed item is still active
- **AND** the turn count has reached `agent.max_turns`
- **THEN** the worker emits a normal run-completed result
- **AND** it returns control to the orchestrator without treating the exit as a failure

#### Scenario: Inactive or missing item exits normally
- **WHEN** a worker completes a turn and the refreshed item is inactive or missing
- **THEN** the worker emits a normal run-completed result
- **AND** it exits without attempting another turn

### Requirement: Runtime events normalize Codex behavior into stable categories
The runtime SHALL normalize Codex app-server and worker lifecycle behavior into stable runtime events instead of exposing raw transport detail.

#### Scenario: Codex session and tool events normalize predictably
- **WHEN** the Codex session reports start, approval, user input, tool success, tool failure, unsupported tool, malformed message, or unknown message behavior
- **THEN** the worker emits the corresponding normalized runtime event category

#### Scenario: Successful turn completion is counted once
- **WHEN** a turn completes successfully
- **THEN** the runtime emits one turn-completed event for that turn
- **AND** token totals come from the turn result rather than a duplicate Codex sink event

#### Scenario: Failed and cancelled turns normalize as run failures
- **WHEN** Codex reports `turn_failed`, `turn_cancelled`, or a turn timeout
- **THEN** the worker emits a normalized run-failed event with the failure category preserved in the message
- **AND** it does not also emit a normal run-completed event for that turn

### Requirement: Retry metadata preserves continuation and failure lineage
The runtime SHALL preserve retry lineage and distinguish continuation retries from failure retries in the metadata it projects and schedules.

#### Scenario: Normal completion seeds continuation retry metadata
- **WHEN** a worker exits normally because the item remains active or reaches the max-turn boundary
- **THEN** the runtime keeps the claim and records a continuation retry entry with attempt `1`

#### Scenario: Failure retry advances the retry lineage
- **WHEN** a worker fails, times out, or reports a retry-worthy stall recovery
- **THEN** the runtime records failure retry metadata that advances the attempt lineage instead of replacing it with a continuation retry

#### Scenario: Stale retry delivery is ignored
- **WHEN** an older retry callback arrives after a newer retry entry has replaced it
- **THEN** the runtime ignores the stale callback
- **AND** it keeps the newer retry metadata intact

### Requirement: Terminal cleanup distinguishes cleanup intent from non-terminal invalidation
The runtime SHALL remove issue workspaces only when the runtime has terminal cleanup intent and SHALL not treat non-terminal invalidation as workspace cleanup.

#### Scenario: Terminal resolution allows workspace cleanup intent
- **WHEN** reconciliation determines that a running item is terminal
- **THEN** the runtime may mark the run for cleanup intent
- **AND** the workspace removal path may run for that terminal item

#### Scenario: Non-terminal invalidation preserves the workspace
- **WHEN** reconciliation determines that a running item is missing, non-active, or unroutable but not terminal
- **THEN** the runtime stops the run without treating it as terminal cleanup
- **AND** it preserves workspace state unless a separate cleanup intent applies

### Requirement: Shutdown is idempotent and closes runtime resources in order
The runtime SHALL shut down idempotently and close the active session, worker activity, orchestrator, and config store without requiring callers to coordinate exact one-time semantics.

#### Scenario: Duplicate shutdown calls are safe
- **WHEN** shutdown is requested more than once
- **THEN** the runtime performs closure only once
- **AND** repeated calls return without double-closing the active session or other runtime resources

#### Scenario: Shutdown waits for cleanup order
- **WHEN** the runtime receives cancellation or termination
- **THEN** it closes the orchestrator, active workers, Codex session, and config store in lifecycle order

### Requirement: Config store lifecycle uses the current snapshot before worker creation
The runtime SHALL create and retain `config.Store` for the active process, read the current typed snapshot from it before worker creation, and keep the last-known-good snapshot stable across the worker lifetime.

#### Scenario: Worker creation uses the current typed snapshot
- **WHEN** the runtime constructs a worker
- **THEN** it reads the current typed settings from `config.Store`
- **AND** it does not reparse the raw workflow file inside the worker constructor

#### Scenario: Snapshot reuse stays stable during the run
- **WHEN** the store already has a last-known-good snapshot
- **THEN** the runtime keeps serving that snapshot to the worker until a newer valid snapshot replaces it
