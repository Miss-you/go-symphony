# T16 Code Review Follow-up

## Addressed Findings

1. `RenderGate.Enqueue` now lets a stable fingerprint rerender after the idle interval instead of dropping duplicate content before the idle check.
2. Terminal frames now include a home/clear ANSI prefix so repeated live renders repaint in place.
3. Overdue and due-now poll windows now render `checking now...` instead of `0s`.
4. Fixture provenance now covers every Go snapshot fixture, rejects provenance entries for missing fixtures, verifies mapped source copies exist, requires derived reasons, and compares normalized frame skeletons plus running and retry rows.
5. Successful Codex turn completion now emits a humanized usage message so dashboard event text is not overwritten by an empty completion event.

## Regression Evidence

- `go test ./internal/observability/... ./internal/dashboard/... ./internal/cli/... -run 'TestProjector|TestRenderGate|TestFixtureProvenance|TestRenderSnapshot|TestTurnCompletedEventMessage|TestWorkerEmitsHumanizedCodexEvent' -v`
- `go test ./internal/observability/... ./internal/dashboard/... ./internal/cli/...`
