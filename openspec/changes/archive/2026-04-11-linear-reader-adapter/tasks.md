## 1. Reader Contract Tests

- [x] 1.1 Add package-scoped tests that lock `internal/trackers/linear.Reader` to `tracker.TrackerReader`, candidate pagination, state-based no-op/no-routing behavior, refresh-by-ID ordering, routing, normalization, error taxonomy, and context cancellation.
- [x] 1.2 Add fake client coverage for query payloads and response decoding so the reader can be exercised without a real Linear connection.

## 2. Candidate and State Reads

- [x] 2.1 Implement the `Reader` constructor and candidate read path with project-scoped active-state pagination and Linear-order preservation.
- [x] 2.2 Implement the state-based read path as a project-scoped cleanup query that returns empty on empty normalized input and never applies assignee routing.

## 3. Refresh, Normalization, and Routing

- [x] 3.1 Implement refresh-by-ID batching at 50 IDs, request-order restoration, and missing-ID omission.
- [x] 3.2 Implement payload normalization into `domain.WorkItem`, including labels, blockers, timestamps, and priority.
- [x] 3.3 Implement assignee routing and `Routable` mapping for no-assignee, exact-match, mismatch, and `me` resolution cases.

## 4. Error Handling and Verification

- [x] 4.1 Implement stable error classification for missing credentials, transport/status failures, GraphQL errors, malformed payloads, missing cursors, and missing viewer identity.
- [x] 4.2 Run `go test ./internal/trackers/linear/...` and the broader repo verification gates, then record any residual notes in the task workspace before implementation closes.
