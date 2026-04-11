## Purpose

Define the Symphony-compatible terminal dashboard contract for projecting runtime snapshots, rendering ANSI frames, preserving live redraw cadence, and proving fixture provenance.

## Requirements

### Requirement: Terminal dashboard projects runtime snapshots without owning runtime state
The system SHALL project `domain.Snapshot` into a terminal dashboard view model without reading orchestrator-private state or owning runtime truth.

#### Scenario: Snapshot fields become dashboard rows
- **WHEN** a snapshot contains running, retry, polling, token, and rate-limit data
- **THEN** the dashboard projection exposes running rows, retry rows, aggregate token totals, rate-limit state, throughput, and next-refresh text
- **AND** the projection does not mutate the snapshot

#### Scenario: Projection remains provider-neutral
- **WHEN** the dashboard projection formats snapshot data
- **THEN** it uses provider-neutral domain fields such as item identifiers and retry entries
- **AND** it does not import Linear tracker packages or tracker write behavior

### Requirement: Terminal dashboard renders the Symphony-compatible ANSI frame
The system SHALL render a deterministic ANSI terminal frame matching the current Symphony terminal dashboard contract.

#### Scenario: Idle dashboard renders compatibility labels
- **WHEN** the projected view has no running items and no retry entries
- **THEN** the rendered frame contains `SYMPHONY STATUS`, `Agents`, `Throughput`, `Runtime`, `Tokens`, `Rate Limits`, `Project`, `Next refresh`, `Running`, and `Backoff queue`
- **AND** it contains `No active agents` and `No queued retries`

#### Scenario: Running rows preserve table contract
- **WHEN** the projected view has running entries
- **THEN** the rendered frame contains the headers `ID`, `STAGE`, `PID`, `AGE / TURN`, `TOKENS`, `SESSION`, and `EVENT`
- **AND** running rows include identifier, state, runtime/turn count, token count, compact session, and event summary

#### Scenario: Retry rows preserve queue contract
- **WHEN** the projected view has retry entries
- **THEN** every retry entry is rendered in due-time order
- **AND** there is no three-row cap
- **AND** retry errors normalize CR/LF and escaped newline sequences to spaces

#### Scenario: Rate limits preserve compact summaries
- **WHEN** rate-limit data is present
- **THEN** the frame renders the limit id, primary bucket, secondary bucket, and credits summary
- **AND** credits can render as unlimited, numeric balance, available, none, or n/a

#### Scenario: Offline status has a minimal frame
- **WHEN** the dashboard renders offline status
- **THEN** the output contains `SYMPHONY STATUS`, `app_status=offline`, and the closing border
- **AND** it does not render the normal running and retry sections

### Requirement: Dashboard live redraw semantics match Symphony cadence
The terminal dashboard SHALL provide a presentation-only render gate that matches the source dashboard's coalescing and idle rerender behavior.

#### Scenario: Changed frames are coalesced before the interval
- **WHEN** a frame changes before `render_interval_ms` has elapsed since the last render
- **THEN** the render gate stores the new frame as pending instead of emitting immediately

#### Scenario: Pending frames flush after the interval
- **WHEN** a pending frame reaches its computed flush time
- **THEN** the render gate emits that frame and clears the pending state

#### Scenario: Idle snapshots still rerender periodically
- **WHEN** the snapshot fingerprint has not changed for at least one second
- **THEN** the render gate allows a rerender so time-derived display remains fresh

### Requirement: Codex events are humanized for dashboard-compatible output
The system SHALL store user-facing Codex event summaries in runtime event messages when enough payload data is available.

#### Scenario: Lifecycle and turn events are summarized
- **WHEN** Codex reports thread, turn, item, approval, user-input, tool, command, token, streaming, malformed, or unknown events
- **THEN** the runtime event message uses the same human-readable vocabulary as the terminal dashboard compatibility contract where the Go payload has enough data

#### Scenario: Missing payload data falls back safely
- **WHEN** a Codex event lacks fields needed for a richer summary
- **THEN** the summary falls back to the normalized event or method name without panicking

### Requirement: Dashboard fixtures retain executable source provenance
The system SHALL prove terminal dashboard fixtures are tied to the source Symphony fixture set or explicitly derived.

#### Scenario: Go fixtures map to source fixtures
- **WHEN** dashboard fixture tests run
- **THEN** every Go expected fixture has an entry in `provenance.json`
- **AND** mapped source fixture files exist under `testdata/status_dashboard_snapshots/source/`

#### Scenario: Provenance check catches drift
- **WHEN** a Go fixture maps to a source fixture
- **THEN** the provenance test compares their normalized frame skeletons
- **AND** only declared adaptations are allowed

#### Scenario: Derived fixtures require explicit reason
- **WHEN** a Go fixture has no source fixture equivalent
- **THEN** `provenance.json` marks it as derived and records why
