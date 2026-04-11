## ADDED Requirements

### Requirement: Linear reader adapter preserves candidate polling behavior
The Linear reader adapter MUST fetch candidate issues from the configured Linear project, filter by the configured active states, and preserve Linear ordering across cursor-paginated pages. Candidate reads MUST request the fields needed to populate `domain.WorkItem` and MUST return all visible candidate items rather than filtering unroutable items out at query time.

#### Scenario: Candidate polling uses project-scoped active-state pagination
- **WHEN** the reader lists candidate items
- **THEN** it queries Linear with the configured project slug and active states
- **AND** it pages with a stable cursor until all visible items are collected
- **AND** the returned slice preserves Linear page order

#### Scenario: Candidate polling does not pre-filter unroutable items
- **WHEN** the reader normalizes candidate results
- **THEN** it sets routing metadata on each item
- **AND** it does not remove items merely because they are unroutable

### Requirement: Linear reader adapter supports cleanup-oriented state reads
The Linear reader adapter MUST implement state-based reads as a distinct cleanup-oriented path. State-based reads MUST be project-scoped, MUST filter only by the requested state names, MUST return an empty slice without calling Linear when the normalized state list is empty, and MUST not apply assignee routing.

#### Scenario: Empty state input is a no-op
- **WHEN** the caller asks for items by an empty or fully blank state list
- **THEN** the reader returns an empty slice
- **AND** it does not call Linear

#### Scenario: State-based reads do not reuse assignee routing
- **WHEN** the reader fetches cleanup-oriented state results
- **THEN** it does not resolve `me`
- **AND** it does not set routing based on the worker assignee filter

### Requirement: Linear reader adapter preserves refresh-by-ID ordering and batching
The Linear reader adapter MUST refresh items by Linear ID in batches of 50, MUST preserve the caller's requested visible order in the returned slice, and MUST omit missing IDs without error.

#### Scenario: Refresh-by-ID preserves request order
- **WHEN** the caller refreshes IDs in a specific order
- **THEN** the reader returns matching visible items in that same order
- **AND** it ignores missing IDs instead of failing the refresh

#### Scenario: Refresh-by-ID chunks large requests
- **WHEN** the caller refreshes more than 50 IDs
- **THEN** the reader issues multiple Linear queries in batches of 50
- **AND** it merges the results back into request order

### Requirement: Linear reader adapter normalizes Linear payloads into runtime items
The Linear reader adapter MUST normalize Linear payloads into `domain.WorkItem` values with provider-neutral runtime fields. It MUST populate identity, title, description, state, priority, branch name, URL, assignee ID, lowercase labels, blockers derived from `blocks` relations, and ISO-8601 timestamps when present.

#### Scenario: Linear payload normalizes into a runtime item
- **WHEN** a Linear issue payload contains the expected fields
- **THEN** the reader produces a `domain.WorkItem`
- **AND** the work item carries the fields needed by later runtime logic

#### Scenario: Blockers and labels are normalized consistently
- **WHEN** a Linear issue includes labels and inverse relations
- **THEN** labels are lowercased
- **AND** blockers are captured only from relations whose type normalizes to `blocks`

### Requirement: Linear reader adapter maps assignee routing into Routable
The Linear reader adapter MUST map assignee semantics into `domain.WorkItem.Routable` for candidate and refresh-by-ID reads. If no assignee is configured, `Routable` MUST be `true`. If an assignee is configured, `Routable` MUST be `true` only when the item assignee matches the configured assignee, or when `me` resolves to the current viewer ID and matches the item assignee. If the configured assignee does not match, `Routable` MUST be `false`. If `me` cannot be resolved, the reader MUST return a missing-viewer-identity error.

#### Scenario: Exact assignee match is routable
- **WHEN** the configured assignee matches the Linear assignee ID exactly
- **THEN** the reader marks the item routable

#### Scenario: Assignee mismatch is not routable
- **WHEN** the configured assignee does not match the item assignee
- **THEN** the reader marks the item not routable

#### Scenario: `me` resolves through the viewer identity
- **WHEN** the configured assignee is `me`
- **THEN** the reader resolves the viewer identity first
- **AND** it marks items routable only when the viewer ID matches the item assignee

### Requirement: Linear reader adapter surfaces stable error classification
The Linear reader adapter MUST surface distinct error categories for missing API token, missing project slug, missing viewer identity for `me`, transport or request failure, non-200 HTTP status, top-level GraphQL errors, malformed payloads, and missing pagination cursors. The reader MUST honor `context.Context` cancellation and deadlines through the client layer.

#### Scenario: Missing token and project slug are distinct errors
- **WHEN** the reader is constructed or invoked without required Linear credentials
- **THEN** it returns a missing-token or missing-project-slug error

#### Scenario: Pagination integrity failure is explicit
- **WHEN** Linear reports more pages but omits the next cursor
- **THEN** the reader returns a missing-pagination-cursor error

#### Scenario: Context cancellation bubbles up
- **WHEN** the caller cancels the context during a Linear request
- **THEN** the reader returns a context-derived error
- **AND** it does not wrap the cancellation as a generic Linear failure
