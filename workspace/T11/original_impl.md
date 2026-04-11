# T11 Original Implementation Research

Scope: current Elixir Symphony Linear reader behavior for parity planning in go-symphony.

## Sources Inspected

- `/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/linear/client.ex`
  - Core Linear reader/client. Owns GraphQL query construction, pagination, normalization, assignee routing, and GraphQL error mapping.
- `/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/linear/adapter.ex`
  - Tracker adapter boundary. Delegates reads to the client and exposes write mutations that are outside the reader focus for T11.
- `/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/linear/issue.ex`
  - Normalized issue struct returned by the reader.
- `/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/tracker.ex`
  - Tracker boundary used by the orchestrator; shows which read methods the runtime consumes.
- `/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/config/schema.ex`
  - Tracker config defaults and env resolution for `LINEAR_API_KEY` / `LINEAR_ASSIGNEE`.
- `/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/orchestrator.ex`
  - Downstream consumer of Linear read results; important for understanding why `assigned_to_worker` and terminal-state refreshes matter.
- `/Users/lihui/Documents/GitHub/symphony/elixir/test/symphony_elixir/workspace_and_config_test.exs`
  - Direct Linear client normalization and pagination tests.
- `/Users/lihui/Documents/GitHub/symphony/elixir/test/symphony_elixir/extensions_test.exs`
  - Adapter delegation tests and GraphQL mutation validation.
- `/Users/lihui/Documents/GitHub/symphony/elixir/test/symphony_elixir/core_test.exs`
  - Orchestrator-facing behavior for fetch-empty cases, reassignment, and dispatch eligibility.
- `/Users/lihui/Documents/GitHub/symphony/SPEC.md`
  - Product contract for Linear-compatible issue tracker integration.

## Data Model

The normalized Linear issue type is `SymphonyElixir.Linear.Issue`:

- `id`, `identifier`, `title`, `description`, `priority`, `state`, `branch_name`, `url`, `assignee_id`
- `blocked_by` as a list of `{id, identifier, state}` maps
- `labels` as lowercase strings
- `assigned_to_worker` as a boolean routing flag
- `created_at` / `updated_at` as `DateTime` values or `nil`

Normalization rules from the client:

- `priority` stays an integer only; any non-integer becomes `nil`
- `labels` are lowercased
- `blocked_by` comes from `inverseRelations` whose trimmed, lowercased `type` is `blocks`
- timestamps are parsed from ISO-8601; parse failures become `nil`
- a non-map payload normalizes to `nil` and is dropped

## GraphQL and Query Behavior

### Candidate fetch

`fetch_candidate_issues/0`:

- requires `tracker.api_key`
- requires `tracker.project_slug`
- queries Linear with `project: { slugId: { eq: $projectSlug } }`
- filters by `tracker.active_states`
- uses `first: 50`, `relationFirst: 50`, and optional `after`
- requests `assignee { id }`, `labels.nodes.name`, `inverseRelations`, `createdAt`, `updatedAt`, `branchName`, `state.name`

### Refresh by IDs

`fetch_issue_states_by_ids/1`:

- returns `{:ok, []}` for an empty input without calling Linear
- de-duplicates IDs before calling Linear
- uses GraphQL variable type `[ID!]!`
- chunks requests in batches of 50
- re-sorts the final result back into the original requested ID order

### State-specific fetch

`fetch_issues_by_states/1`:

- normalizes input by `to_string/1` and `Enum.uniq/1`
- returns `{:ok, []}` for an empty normalized set without calling Linear
- is used for startup terminal cleanup
- does not apply assignee routing; it fetches by project + states only

## Pagination, Ordering, and Routing

- Candidate pagination is cursor-based with `pageInfo.hasNextPage` and `pageInfo.endCursor`.
- Pagination preserves API order across pages by prepending each page and reversing once at the end.
- If `hasNextPage` is true but `endCursor` is missing or blank, the client returns `{:error, :linear_missing_end_cursor}`.
- Refresh-by-ID results are returned in the caller’s requested ID order, not Linear’s response order.
- `assigned_to_worker` is computed from `tracker.assignee`, but the reader does not filter candidate issues out at the query level.
- If `tracker.assignee` is:
  - `nil`, every normalized issue gets `assigned_to_worker: true`
  - a non-empty string other than `"me"`, only issues whose assignee ID matches that exact string get `assigned_to_worker: true`
  - `"me"`, the client resolves `viewer.id` first and matches against that ID
  - blank or whitespace, routing collapses to `nil` and behaves like “no routing filter”
- A missing viewer identity when resolving `"me"` returns `{:error, :missing_linear_viewer_identity}`.

## Error Classification

Reader-level errors are classified as:

- `{:error, :missing_linear_api_token}`
- `{:error, :missing_linear_project_slug}`
- `{:error, {:linear_api_request, reason}}` for transport/request failure
- `{:error, {:linear_api_status, status}}` for non-200 responses
- `{:error, {:linear_graphql_errors, errors}}` for top-level GraphQL errors
- `{:error, :linear_unknown_payload}` for payloads that do not match expected shapes
- `{:error, :linear_missing_end_cursor}` for broken pagination metadata

The client logs a trimmed response body and operation name on non-200 GraphQL responses.

## Behavior That Must Be Preserved

- Project-scoped active-state candidate fetch using `slugId` and cursor pagination.
- Empty-input no-op behavior for both `fetch_issues_by_states([])` and `fetch_issue_states_by_ids([])`.
- Exact `assigned_to_worker` semantics, especially `"me"` resolution and exact-ID matching.
- Lowercased labels, `blocks`-derived blockers, integer-only priority, and ISO-8601 timestamp parsing.
- Refresh-by-ID reordering to the original request order.
- `linear_missing_end_cursor` as an explicit pagination integrity error.
- Error taxonomy and no-silent-fallback behavior for malformed payloads.

## Behavior That Can Stay In the Linear-Specific Shell

- Raw GraphQL transport details and request logging.
- `linear_graphql` tool exposure and mutation-related behavior.
- Comment/state mutation operations in `Linear.Adapter`.
- Orchestrator decisions that consume `assigned_to_worker`, rather than the reader itself filtering dispatch eligibility.

## Concrete Parity Risks for Go

- Forgetting that `assigned_to_worker` is a routing signal, not a query-time filter.
- Returning refresh-by-ID results in API order instead of request order.
- Losing the exact `"me"` resolution path and the `missing_linear_viewer_identity` failure mode.
- Collapsing GraphQL, HTTP status, and transport failures into one generic error.
- Forgetting the `linear_missing_end_cursor` guard, which the Elixir code treats as a distinct integrity failure.
- Normalizing labels or blockers differently from the current client, which would change dispatch eligibility and prompt content.
