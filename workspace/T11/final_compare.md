# T11 Final Comparison

## Compared Inputs

- `workspace/T11/original_impl.md`
- `workspace/T11/final_impl.md`
- `openspec/changes/linear-reader-adapter/specs/linear-reader-adapter/spec.md`
- `internal/trackers/linear/reader.go`
- `internal/trackers/linear/reader_test.go`

## Result

The implementation matches the approved T11 scope and the current Symphony Linear read behavior closely enough to close the task.

Preserved behavior:

- candidate reads are project-scoped by Linear project slug and active states
- candidate reads use cursor pagination and preserve Linear page order
- `ListByStates` is a separate project/state cleanup read, returns empty on empty normalized input, and does not apply assignee routing
- refresh-by-ID batches at 50 IDs, omits missing IDs, and restores caller-visible request order
- labels are lowercased, blockers come only from `blocks` inverse relations, timestamps parse from ISO-8601, and priority stays integer-only
- candidate and refresh reads map assignee routing into `domain.WorkItem.Routable`
- `me` is resolved through the viewer query and missing viewer identity is a distinct error
- transport/status, GraphQL, malformed payload, missing cursor, and context-cancellation failures remain distinguishable

Boundary check:

- no tracker writes were added to `internal/tracker`
- no `linear_graphql`, comment creation, or state mutation behavior was added
- no runtime/orchestrator wiring was added ahead of the later integration tasks

## Residual Risk

No high-severity residual risk is known. Remaining work is intentionally deferred to T12 and later runtime integration tasks as recorded in `todo.md`.
