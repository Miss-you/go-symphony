# T11 Final Implementation

## Review Gate

`final_impl_v1.md` passed rubric review after one correction round.

Round-two review results:

- `review_1_round2.md`: 92 / 100, no high-severity issues
- `review_2_round2.md`: 89 / 100, no high-severity issues
- average: 90.5 / 100

Key review corrections accepted into this final plan:

- make `ListByStates` explicitly project-scoped, state-only, empty-input-safe, and independent of assignee routing
- make `Routable` a mandatory adapter contract for candidate and refresh-by-ID reads instead of an advisory mapping
- add test expectations that prove the cleanup-oriented state read cannot regress through routing reuse

Acceptance decision:

- average score exceeds the `>= 80` threshold
- no reviewer reported a remaining high-severity issue
- the one non-blocking precision note is resolved below by requiring `ListByStates` to leave `Routable` nil

## Task Goal

T11 replaces the placeholder `internal/trackers/linear` package with a real read adapter that implements the existing provider-neutral `tracker.TrackerReader` contract while preserving Symphony's current Linear read behavior.

The adapter must preserve:

- candidate polling
- state-based reads used by cleanup-oriented runtime work
- refresh-by-ID reads
- assignee routing
- Linear payload normalization into `domain.WorkItem`
- Linear-specific error classification

## Final Scope

T11 lands:

- a concrete read-only `internal/trackers/linear.Reader`
- GraphQL query execution for candidate fetch, state-based fetch, refresh-by-ID, and viewer lookup
- normalization from Linear issue payloads into `domain.WorkItem`
- worker-assignee routing mapped into `domain.WorkItem.Routable`
- package-scoped tests covering reader behavior and parity-sensitive edge cases

T11 does not land:

- tracker write behavior
- `linear_graphql` tool injection
- comment creation
- issue state mutation
- workflow selection
- runtime wiring beyond package-level reader proof
- any new core tracker abstraction beyond `TrackerReader`

## Package Shape

`internal/trackers/linear` stays small and read-only.

Public package surface:

- `type Reader struct`
- `func NewReader(settings config.ProviderSettings, client Client) (*Reader, error)`
- `type Client interface`
- typed errors or exported sentinel errors only where tests and callers need stable classification

`Reader` implements `tracker.TrackerReader` directly:

- `ListCandidates(context.Context) ([]domain.WorkItem, error)`
- `ListByStates(context.Context, []string) ([]domain.WorkItem, error)`
- `RefreshByIDs(context.Context, []string) ([]domain.WorkItem, error)`

The client dependency should be thin and testable. It should only execute a GraphQL query with variables, return a decoded payload, and surface transport/status failures in a way the reader can classify. The package may include an HTTP-backed client, but tests should use fakes.

## Read Contract

### Candidate Reads

Candidate reads are project-scoped and active-state-scoped.

`ListCandidates` must:

- require Linear API credentials and project slug through the constructed reader settings
- resolve worker-assignee routing once before reading pages
- query by `project.slugId`
- query by configured active states
- page with `first: 50` and `after`
- request all fields needed to populate `domain.WorkItem`
- preserve Linear ordering across pages
- return all visible candidate items with `Routable` set, rather than filtering unroutable items out

### State-Based Reads

`ListByStates` is a distinct cleanup-oriented read path, not a candidate-routing path.

It must:

- return an empty slice without calling Linear when the normalized state list is empty
- query by configured Linear `project.slugId`
- filter only by the requested state names
- normalize requested states by trimming whitespace, dropping blank entries, and de-duplicating the remaining names before the request
- not resolve `me`
- not apply assignee routing
- set `Routable` to nil on returned items, because the cleanup path in Symphony does not use worker assignment as a visibility gate

The method may share GraphQL field-selection, pagination, and payload-normalization helpers with candidate reads, but it must not reuse candidate-specific routing semantics.

### Refresh-By-ID Reads

`RefreshByIDs` must:

- accept tracker-internal Linear IDs, not human identifiers
- return an empty slice without calling Linear when input is empty after de-duplication
- de-duplicate IDs before making requests
- chunk GraphQL requests in batches of 50 IDs
- use a Linear issue lookup by `[ID!]!`
- preserve the caller's requested visible order in the returned slice
- omit missing IDs without error
- set `Routable` using the same mandatory routing rules as candidate reads

## GraphQL Operations

The adapter should use four read query shapes:

- candidate issues by project slug, active states, `first`, and `after`
- issues by project slug and arbitrary state names for `ListByStates`
- issues by ID list for `RefreshByIDs`
- viewer lookup for `Assignee == "me"`

Issue queries must request:

- `id`
- `identifier`
- `title`
- `description`
- `priority`
- `state.name`
- `branchName`
- `url`
- `assignee.id`
- `labels.nodes.name`
- `inverseRelations(first: 50)` with relation type and blocker issue identity/state
- `createdAt`
- `updatedAt`
- `pageInfo.hasNextPage` and `pageInfo.endCursor` on paginated queries

If a paginated response reports `hasNextPage=true` but has no non-empty `endCursor`, the reader must return the Linear missing-cursor error.

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
- `BlockedBy` from inverse relations whose trimmed, lowercased relation type is `blocks`
- `CreatedAt` and `UpdatedAt` from ISO-8601 timestamps; parse failures become nil
- non-object issue payloads are dropped rather than converted into partial items

Normalization must preserve the runtime-relevant fields already frozen in `domain.WorkItem`, especially blockers, labels, priority, routing eligibility, and timestamps.

## Routing Contract

For candidate and refresh-by-ID reads, `Routable` is mandatory:

- no assignee configured: `true`
- blank or whitespace assignee configured: `true`
- configured assignee matches the item's Linear assignee ID exactly: `true`
- configured assignee does not match: `false`
- configured assignee is present and the item has no assignee: `false`
- configured assignee is `me`: resolve viewer ID first, then match exactly
- unresolved `me` lookup returns a missing-viewer-identity error, not `false`

Routing is not the same as dispatch policy. The reader exposes a routing signal; the orchestrator remains responsible for deciding whether to dispatch.

`ListByStates` does not apply this contract and must leave `Routable` nil.

## Error Taxonomy

The reader should classify errors so later runtime code can distinguish config, transport, payload, and pagination failures.

Required categories:

- missing API token
- missing project slug
- missing viewer identity for `me`
- transport or request failure
- non-200 HTTP status
- top-level GraphQL errors
- unknown or malformed payload shape
- missing pagination cursor when Linear says more pages exist

`context.Context` must be passed through the reader API and honored by the client layer. Cancellation and deadline expiry should bubble back as context-derived errors instead of being wrapped into a generic Linear failure.

## TDD And Verification

Start with failing package tests for:

1. `Reader` satisfying `tracker.TrackerReader`
2. candidate query paging and payload normalization
3. `ListByStates` project scoping, normalized empty-input no-op, and no-routing behavior
4. refresh-by-ID batching and request-order restoration
5. routing for no assignee, blank assignee, exact ID match, mismatch, missing issue assignee, and `me`
6. label, blocker, timestamp, priority, and routability normalization
7. missing pagination cursor handling
8. GraphQL error classification
9. context cancellation propagation

The formal task gate is:

`go test ./internal/trackers/linear/...`

Broader verification should still run before closure:

- `go test ./...`
- `make build`
- `make lint`
- `make test-e2e` when applicable to the current repository state

## Deferred Items

Deferred to T12:

- `linear_graphql` write behavior
- comment creation
- state mutation
- any toolbridge behavior that writes back to Linear

Deferred to later runtime tasks:

- constructing this reader from full runtime config
- wiring the reader into the orchestrator service
- workflow selection
- end-to-end Linear orchestration runs

## Main Risks

- treating assignee routing as a query filter instead of a `Routable` signal
- accidentally applying assignee routing to `ListByStates`
- returning refresh-by-ID results in Linear response order instead of caller request order
- collapsing distinct Linear error classes into a generic error
- omitting `me` resolution or silently treating viewer lookup failure as unroutable
- normalizing blockers or labels differently from Symphony
- forgetting the missing-end-cursor guard on paginated reads

## Bottom Line

T11 should land a narrow, testable, read-only Linear adapter that matches the current Symphony Linear read contract and nothing broader.

That keeps provider-specific read behavior in `internal/trackers/linear`, preserves the frozen core `TrackerReader`, and leaves Linear writes for T12.
