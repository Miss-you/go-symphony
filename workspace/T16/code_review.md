# T16 Code Review

## Findings

1. **High** `internal/dashboard/live.go:30-49,64-69`
   `ForceIdleRerender()` can return `true`, but `Enqueue()` still drops the frame at line 34 whenever the rendered content matches `lastContent`. That means a stable fingerprint cannot actually force the promised one-second idle rerender through this API, because duplicate suppression wins before fingerprint or age are considered. The live redraw contract in T16 is therefore not implementable as written.

2. **Medium** `internal/dashboard/renderer.go:31-80`
   The renderer emits the frame body directly and never prefixes a home/clear ANSI sequence. The task plan explicitly calls for live terminal output to repaint in place, but this implementation will append new frames over old ones unless some outer caller clears the terminal first. That is a visible compatibility regression for the terminal dashboard path.

3. **Medium** `internal/observability/dashboard.go:186-198`
   `nextRefresh()` returns `0s` when `NextPollAt` is due now or already overdue. The documented compatibility contract only allows `checking now...`, a positive countdown, or `n/a`. Emitting `0s` creates a frame string that is outside the accepted vocabulary and will diverge from the Elixir behavior on immediate refresh cycles.

4. **Medium** `internal/dashboard/renderer_test.go:37-53,87-111`
   The provenance gate is too weak to catch the class of drift it claims to enforce. `TestFixtureProvenance()` only checks five hard-coded fixtures, and `normalizedSkeleton()` strips almost all frame content, so it would miss added fixtures, broken derived entries, row-count drift, or most cell-level changes. This leaves the executable provenance requirement mostly documentary instead of enforceable.

## Residual Risks

- The package tests pass, but this task still does not exercise a real terminal session, so live redraw behavior is only unit-tested.
- `T16` intentionally stops short of wiring dashboard startup/shutdown into the CLI; end-to-end integration still depends on later work.
