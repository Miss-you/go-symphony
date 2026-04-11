# T11 Go Baseline

## What Exists In Go Today

The current Go baseline already freezes the read-only tracker core and the provider-neutral domain/config contracts:

- `internal/tracker/TrackerReader` exposes exactly three read methods: `ListCandidates`, `ListByStates`, and `RefreshByIDs`.
- `internal/trackers/memory.Reader` is the only concrete tracker adapter today. It implements `TrackerReader`, snapshots seed data, and deep-copies returned `domain.WorkItem` values.
- `internal/domain.WorkItem` already carries the fields the later tracker adapter must populate: identity, title, description, state, priority, branch name, URL, assignee ID, labels, blockers, routability, and timestamps.
- `internal/config.Settings.Provider` already normalizes legacy `tracker.*` input into provider-neutral settings, while still retaining Linear-specific configuration fields such as `Endpoint`, `APIKey`, `Project`, `Assignee`, `ActiveStates`, and `TerminalStates`.
- `internal/config` already accepts `ProviderLinear` and `ProviderMemory`, resolves `LINEAR_API_KEY` and `LINEAR_ASSIGNEE`, and validates that Linear config has the required project/API key.

The `internal/trackers/linear` package is still a placeholder: it contains only `doc.go` and no implementation or tests.

## Elixir Reference Behavior To Preserve

The Elixir tracker boundary is broader, but the Go task only needs the read path and normalization behavior that the runtime actually depends on.

### Tracker boundary

`SymphonyElixir.Tracker` exposes five callbacks in Elixir:

- `fetch_candidate_issues/0`
- `fetch_issues_by_states/1`
- `fetch_issue_states_by_ids/1`
- `create_comment/2`
- `update_issue_state/2`

For Go parity, T11 should preserve only the read behavior in the core boundary. The write callbacks are compatibility-shell behavior and belong outside `internal/tracker`.

### Linear adapter shape

`SymphonyElixir.Linear.Adapter` delegates reads to `SymphonyElixir.Linear.Client` and writes to GraphQL mutations. The important read-side contract is:

- candidate reads come from the current Linear project and active states
- state-based reads use a state-name list
- refresh-by-id reads use Linear IDs and return the requested items in request order
- assignee routing is enforced through a configured worker-assignee filter

### Linear client behavior

The Linear client in Elixir is where the real contract lives:

- Candidate query:
  - filters by `project.slugId`
  - filters by `state.name in active_states`
  - requests `id`, `identifier`, `title`, `description`, `priority`, `state.name`, `branchName`, `url`, `assignee.id`, `labels.nodes.name`, `inverseRelations`, `createdAt`, and `updatedAt`
  - paginates with `first` and `after`
- Refresh-by-id query:
  - requests issues by ID in batches of 50
  - preserves requested ID order in the returned list
  - omits missing IDs without error
- Normalization:
  - state names are compared case-insensitively after trimming
  - labels are lowercased
  - blockers are derived only from `inverseRelations` entries whose type normalizes to `blocks`
  - timestamps are parsed from ISO-8601 strings
  - priority is kept as an integer when present
- Assignee routing:
  - if no assignee is configured, every fetched issue is considered routed to the worker
  - if an assignee is configured, only issues whose assignee matches are routed
  - `me` is special-cased and resolved through a viewer query
  - unresolved or mismatched assignee configuration makes the item unrouted

### Error behavior

The Elixir client distinguishes these error classes that matter for parity:

- missing API token
- missing project slug
- missing viewer identity when resolving `me`
- GraphQL transport/status failure
- GraphQL payload with `errors`
- unknown payload shape
- missing pagination cursor when `hasNextPage` is true

## Existing Tests And Coverage Gaps

Go-side coverage today is strong for the frozen contracts, but empty for the Linear adapter itself:

- `internal/tracker/tracker_test.go` locks the interface shape to the three read methods.
- `internal/trackers/memory/reader_test.go` verifies deep-copy behavior, state normalization, request-order refresh, and empty-input handling.
- `internal/domain/domain_contract_test.go` locks the exported runtime model shape.
- `internal/trackers/linear` has no tests at all.

That means T11 has no Go-side proof yet for:

- candidate query normalization into `domain.WorkItem`
- assignee routing, including `me`
- pagination across multiple Linear pages
- refresh-by-id batching and order preservation
- Linear-specific error mapping
- blocker and label extraction from Linear payloads
- empty-input behavior specific to the Linear adapter

## Minimal Go-Native Shape For T11

The smallest useful Go adapter shape is:

- a concrete `internal/trackers/linear.Reader`
- constructor taking the provider settings or a thin client dependency
- `ListCandidates(context.Context) ([]domain.WorkItem, error)`
- `ListByStates(context.Context, []string) ([]domain.WorkItem, error)`
- `RefreshByIDs(context.Context, []string) ([]domain.WorkItem, error)`

That adapter should stay read-only, use `domain.WorkItem` directly, and keep Linear-specific query/pagination/error details inside `internal/trackers/linear`.

The adapter should not widen `TrackerReader`, introduce writes into the core, or add generic tracker abstractions beyond what the frozen contract already requires.

## Constraints And Risks

- The approved design keeps the Go core provider-neutral, so Linear-specific behavior belongs in the compatibility shell, not in `internal/tracker`.
- `internal/config` already validates `linear` vs `memory`, so T11 should rely on typed settings rather than reparsing raw workflow maps.
- The Elixir adapter uses `assigned_to_worker` as a routing gate; in Go that should become the existing `domain.WorkItem.Routable` field or equivalent adapter-derived logic, not a new core concept.
- `WorkItem.BlockedBy`, `Labels`, `Priority`, `Routable`, `CreatedAt`, and `UpdatedAt` all need to survive normalization because later scheduler and observability logic depends on them.
- Refresh-by-ID must preserve request order for visible matches, because the Elixir runtime relies on that ordering during revalidation.
- Missing or unrouted items should be treated as adapter normalization results, not as core tracker write or policy failures.

## T11 Implication

T11 is not about inventing new tracker behavior. It is about replacing the placeholder `internal/trackers/linear` package with a real read adapter that reproduces the Elixir Linear read contract on top of the already-frozen Go `TrackerReader` and `domain.WorkItem` shapes.
