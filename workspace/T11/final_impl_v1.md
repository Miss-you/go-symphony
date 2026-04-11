# T11 Final Implementation v1

## Review Gate

This plan is intended to pass the T11 rubric review with:

- average score `>= 80`
- no high-severity issues
- no scope drift into tracker writes or provider-agnostic abstractions

Rubric focus:

- Symphony alignment and source fidelity
- Go-native simplicity
- clean boundaries
- implementation clarity and testability
- verification coverage

## Task Goal

T11 replaces the placeholder `internal/trackers/linear` package with a real read adapter that reproduces the current Symphony Linear read contract on top of the already-frozen Go `TrackerReader` and `domain.WorkItem` shapes.

The adapter must preserve the current product contract for:

- candidate polling
- state-based reads
- refresh-by-ID reads
- assignee routing
- normalization of Linear payloads into runtime items
- Linear-specific error classification

## Final Scope

T11 lands:

- a concrete `internal/trackers/linear` reader implementation
- GraphQL query handling for candidate fetch, refresh-by-ID, and viewer lookup
- normalization from Linear payloads into `domain.WorkItem`
- routing metadata derived from Linear assignee data
- package-scoped tests covering the reader contract

T11 does not land:

- tracker write behavior
- `linear_graphql` tool injection
- comment creation
- state mutation
- workflow selection
- any new core tracker abstraction beyond `TrackerReader`

## Package Shape

`internal/trackers/linear` should stay small and read-only.

Proposed exported surface:

- `type Reader struct`
- `func NewReader(settings config.ProviderSettings, client Client) (*Reader, error)`
- `type Client interface` for GraphQL execution

Proposed internal helpers:

- `candidateQuery`
- `issuesByIDsQuery`
- `viewerQuery`
- payload normalization helpers
- pagination helpers
- assignee routing helpers
- error classification helpers

The reader should implement `tracker.TrackerReader` directly.

The client dependency should be thin and testable. It should carry only the ability to send a GraphQL query, receive a decoded payload, and return transport or status errors in a form the reader can classify.

## Read Contract

### Candidate reads

Candidate reads are project-scoped and active-state-scoped.

The candidate query must:

- filter by `project.slugId`
- filter by `state.name in active_states`
- page with `first: 50`
- page with `after`
- request the fields needed to populate `domain.WorkItem`

The reader must preserve Linear ordering across pages and must not reorder candidates itself.

### State-based reads

`ListByStates` is a distinct cleanup-oriented read path, not a candidate-routing path.

It must:

- return an empty slice without calling Linear when the normalized state list is empty
- query the configured Linear project with `project.slugId`
- filter only by the requested state names
- normalize requested states by trimming whitespace and de-duplicating exact post-trim strings before the request
- avoid assignee routing entirely
- leave `Routable` unset or neutral rather than using the worker-assignee filter

The method may share GraphQL field-selection and payload-normalization helpers with candidate reads, but it must not reuse candidate-specific routing semantics. This preserves the Elixir `fetch_issues_by_states/1` behavior used by startup terminal-workspace cleanup.

This method exists because the frozen core tracker contract includes it and the Symphony contract needs it for cleanup-oriented runtime work.

### Refresh-by-ID reads

Refresh-by-ID reads must:

- accept tracker-internal IDs
- deduplicate empty input into an empty response
- chunk GraphQL requests in batches of 50 IDs
- preserve the caller’s requested visible order in the returned slice
- omit missing IDs without error

## GraphQL Operations

The adapter should use three query shapes.

### Candidate query

Use a Linear issue query with:

- `project.slugId`
- `state.name in active_states`
- `first` and `after` pagination
- `labels.nodes.name`
- `inverseRelations`
- `assignee.id`
- `createdAt`
- `updatedAt`
- `priority`
- `branchName`
- `url`

### Refresh-by-ID query

Use a Linear issue lookup by ID list with:

- `[ID!]!`
- batching at 50 IDs
- no reliance on response order

### Viewer query

Use a viewer lookup only when routing configuration requires `me`.

The viewer query exists to resolve the current Linear identity before assignee matching.

## Normalization Rules

Normalize Linear payloads into `domain.WorkItem` with these rules:

- `ID` from Linear `id`
- `Identifier` from Linear `identifier`
- `Title` from Linear `title`
- `Description` from Linear `description`
- `State` from `state.name`
- `Priority` only when Linear returns an integer
- `BranchName` from `branchName`
- `URL` from `url`
- `AssigneeID` from `assignee.id`
- `Labels` as lowercase names
- `BlockedBy` from inverse relations whose relation type normalizes to `blocks`
- `Routable` from assignee routing
- `CreatedAt` and `UpdatedAt` from ISO-8601 timestamps

Normalization must preserve the runtime-relevant fields already frozen in `domain.WorkItem`, especially:

- `BlockedBy`
- `Labels`
- `Priority`
- `Routable`
- `CreatedAt`
- `UpdatedAt`

### `Routable` mapping

`Routable` should encode whether the item is assigned to the current worker.

Mandatory mapping for candidate and refresh-by-ID reads:

- no assignee configured: `true`
- configured assignee matches the item assignee: `true`
- configured assignee does not match: `false`
- configured assignee is present and item assignee is missing: `false`
- `me` resolves via viewer lookup first, then matches on viewer ID
- unresolved `me` lookup is an error, not a silent `false`

The adapter should not invent a new core routing concept. It should map Linear assignee semantics into the already-frozen `domain.WorkItem.Routable` field.

`ListByStates` is the exception: because the Elixir cleanup read does not apply assignee routing, this path must not use worker-assignee matching to set `Routable=false`.

## Assignee Routing

Assignee routing is a reader concern because it changes how the runtime interprets candidate visibility.

Rules:

- blank assignee configuration means no routing filter
- exact assignee string matches exact Linear assignee ID
- `me` triggers a viewer lookup and then matches the viewer ID
- malformed or unresolved `me` resolution returns a routing error

Routing is not the same as dispatch policy.

The reader should still return all visible candidate items from Linear, but each item must carry its routing signal in `Routable` so later orchestration can apply policy cleanly.

## Error Taxonomy

The reader should classify errors so later runtime code can distinguish config, transport, payload, and pagination problems.

Required categories:

- missing API token
- missing project slug
- missing viewer identity for `me`
- transport or request failure
- non-200 HTTP status
- top-level GraphQL errors
- unknown or malformed payload shape
- missing pagination cursor when Linear says more pages exist

Error handling should preserve context for diagnosis without leaking secrets.

`context.Context` must be passed through the reader API and honored by the client layer. Cancellation and deadline expiry should bubble back as context-derived errors rather than being wrapped into a generic Linear failure.

## TDD Plan

Use TDD at the package boundary.

Start with failing tests for:

1. `TrackerReader` satisfaction by the new reader
2. candidate query paging and payload normalization
3. `ListByStates` project scoping, normalized empty-input no-op, and no assignee-routing behavior
4. refresh-by-ID batching and request-order restoration
5. assignee routing, including `me`
6. label, blocker, timestamp, priority, and routability normalization
7. missing pagination cursor handling
8. GraphQL error classification
9. context cancellation propagation

The tests should prove the contract, not just the happy path.

Recommended test structure:

- reader constructor and interface conformance
- query payload tests via a fake GraphQL client
- normalization tests against fixture payloads
- routing tests for nil, exact ID, and `me`
- state-based read tests proving empty input does not call the client and configured assignee does not suppress cleanup results
- error tests for each classification bucket

## Verification Plan

The task board gate is:

`go test ./internal/trackers/linear/...`

That gate should prove:

- the package compiles
- the reader implements `TrackerReader`
- the GraphQL helper paths work under test fakes
- normalization and routing are correct
- error taxonomy is stable

Broader verification may be run if needed for confidence, but the task gate remains the package-scoped linear reader test suite.

## Risks And Deferred Items

### Deferred to T12

- `linear_graphql` write behavior
- comment creation
- state mutation
- any toolbridge behavior that writes back to Linear

### Deferred to later runtime tasks

- consuming the reader in broader runtime paths beyond package-level proof
- workflow selection
- end-to-end orchestration wiring

### Main parity risks

- treating assignee routing as a query filter instead of a routing signal
- losing refresh-by-ID order restoration
- collapsing distinct Linear error classes into one generic failure
- omitting `me` resolution or failing to surface viewer lookup errors
- normalizing blockers or labels differently from Symphony’s current contract
- forgetting the missing-end-cursor guard on paginated reads

## Bottom Line

T11 should land a narrow, testable, read-only Linear adapter that matches the current Symphony Linear read contract and nothing broader.

That keeps provider-specific read behavior in `internal/trackers/linear`, preserves the frozen core `TrackerReader`, and leaves Linear writes for T12.
