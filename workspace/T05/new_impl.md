# T05 New Implementation

## Scope

T05 should define the provider-neutral runtime domain that T06+ will consume for orchestration, retry handling, polling state, and snapshot projection.

This task is not about tracker reads, workflow parsing, or provider-specific writes. Those concerns already live in `internal/config` and will stay outside the core domain package.

## Current Go Evidence

- `internal/config/workflow.go` already owns raw `WORKFLOW.md` parsing and keeps the workflow payload separate from typed runtime state.
- `internal/config/settings.go` already normalizes legacy `tracker.*` config into a typed `Settings` model with provider-neutral `ProviderKind` values (`linear`, `memory`).
- `internal/config/store.go` already stores raw workflow plus typed settings in one atomic snapshot and preserves last-known-good data on reload failure.
- `internal/config/settings_test.go` already locks in the config contract for defaults, env fallback, validation, and atomic snapshot behavior.
- `internal/domain/doc.go` is still only a package placeholder, so T05 is the first place where the runtime domain is actually defined.
- `internal/orchestrator/doc.go`, `internal/tracker/doc.go`, and `internal/observability/doc.go` are also placeholders, which means T05 has to provide the stable types they will later depend on.

## Existing Constraints

The approved design imposes a narrow shape for the core:

- Core packages must stay provider-neutral.
- `WorkItem` must not carry Linear-specific naming or write semantics.
- The orchestrator is the single owner of mutable runtime state.
- Workers only report `RunEvent` values back to the orchestrator.
- `observability` must be projection-only, not a second state machine.
- No universal tracker write API and no provider-agnostic default workflow in V1.

The design also names the core domain concepts explicitly: `WorkItem`, `Blocker`, `RunEvent`, `Snapshot`, `RetryEntry`, and `PollingState`.

## Proposed Go-native Shape

`internal/domain` should become a small package of plain data types only:

- `WorkItem`: the provider-neutral unit of work, with only fields needed for orchestration and prompt rendering.
- `Blocker`: a structured reason an item cannot run yet, so the orchestrator can reason about pause conditions without provider-specific logic.
- `RetryEntry`: a retry record with backoff metadata and the item reference needed for rescheduling.
- `PollingState`: current poll cadence and countdown data for the runtime loop.
- `RunEvent`: a typed event stream from workers to the orchestrator, covering lifecycle milestones without mutating shared state directly.
- `Snapshot`: the orchestrator projection that downstream API, terminal dashboard, and web dashboard code can render.

The package should stay intentionally boring:

- no file I/O
- no YAML parsing
- no config loading
- no provider-specific fields unless they are truly runtime-neutral
- no logic that duplicates orchestrator ownership

For T06+, the domain package should make it easy to answer:

- what item is active
- what is blocked
- what should retry next
- what the current poll state is
- what the latest runtime snapshot should show

That suggests a minimal model with stable identifiers, provider-neutral state labels, timestamps where needed for retry/poll accounting, and enough structure for snapshots to be rendered consistently.

## Open Questions

- How small should `WorkItem` be in V1: only orchestration fields, or orchestration plus prompt-rendering fields?
- Should `Snapshot` carry a full item list, or only the aggregate runtime view that observability needs?
- Should `RunEvent` use a closed enum-style event type, or a small tagged struct with optional payloads?
- Do we need separate retry and polling timestamps in the core, or can orchestrator internals derive some of that?
- Should provider identity appear in `WorkItem`, or stay in config/orchestrator state only?
