## Overview

T16 implements the terminal dashboard as a compatibility projection over `domain.Snapshot`. The implementation keeps mutable runtime truth inside `internal/orchestrator` and adds only presentation state needed to match the Elixir dashboard's redraw behavior.

The implementation has three layers:

1. `internal/observability`: converts `domain.Snapshot` plus display context into a stable dashboard view model and computes rolling throughput.
2. `internal/dashboard`: renders the view model as ANSI terminal text and owns the presentation-only render gate.
3. `internal/cli`: continues to own runtime assembly, with one narrow change to store humanized Codex event summaries in `RunEvent.Message`.

## Source Compatibility

The source Symphony behavior is documented in `workspace/T16/original_impl.md`. The Go implementation should preserve:

- top-level frame labels and section order
- running table headers
- no-active and no-retry fallback rows
- retry queue ordering and newline sanitization
- compact rate-limit summaries and credits variants
- rolling throughput display
- `Next refresh` values
- minimal offline frame
- humanized Codex event text
- live redraw coalescing and once-per-second idle rerender

## View Model Boundary

`internal/observability` may define:

- `DashboardMode`
- `DashboardContext`
- `DashboardView`
- `RunningRow`
- `RetryRow`
- `Projector`

The projector may cache only token samples and the one-second throughput throttle. It must not own work-item state, retry state, run lifetimes, or poll state.

`DashboardContext` carries presentation context that is not part of the snapshot, such as max agent count, project URL, dashboard URL, and the current time.

## Renderer Boundary

`internal/dashboard` consumes `observability.DashboardView` and returns a deterministic string. It should:

- use only the standard library
- avoid terminal UI frameworks
- keep formatting helpers small and table-driven
- use exact fixture assertions for the frame contract
- keep `RenderOffline` and unavailable rendering explicit

The renderer should not call the orchestrator, tracker, config store, or CLI runtime.

## Live Redraw Gate

The Elixir dashboard rate-limits rendering while still forcing periodic idle refreshes. T16 should encode that behavior in `internal/dashboard.RenderGate` instead of wiring a long-running terminal process into CLI.

`RenderGate` owns only presentation timing:

- last rendered content
- last rendered timestamp
- pending content
- pending flush time
- last snapshot fingerprint

It must prove:

- first frame renders immediately
- unchanged content is suppressed
- changed content before `render_interval_ms` is pending
- pending content flushes after the interval
- unchanged snapshot fingerprints can still trigger an idle rerender after one second

This keeps live cadence compatible while leaving full CLI startup/shutdown integration to T18.

## Event Humanization

The Go runtime currently stores `domain.RunEvent.Message` as a string. Because `domain.Snapshot` does not carry raw Codex payloads, event summarization must happen before the event reaches the orchestrator.

T16 should add `observability.SummarizeCodexEvent(codex.Event)` and call it from `cli.emitCodexEvent`. This shares humanized text with terminal and HTTP projections without importing `internal/dashboard` into CLI.

The implementation should preserve useful fallbacks when payload data is missing, but it should use payload details when available for commands, token usage, approval, user-input, and streaming events.

## Fixture Provenance

The dashboard fixtures are a parity contract, so they need executable provenance:

- Copy unmodified Elixir source fixtures into `internal/dashboard/testdata/status_dashboard_snapshots/source/`.
- Store Go expected fixtures beside the renderer tests.
- Add `provenance.json` mapping every Go fixture to a source fixture or an explicit `derived` reason.
- Add `TestFixtureProvenance` that verifies source files exist, mappings cover all Go fixtures, and normalized source/Go frame skeletons match except for declared adaptations such as the missing Go pid field.
- Record the same mapping and adaptations in `workspace/T16/fixture_provenance.md`.

This prevents accidental hand-transcription from becoming local truth.

## Non-Goals

- Do not change `internal/domain` or `internal/orchestrator` ownership.
- Do not add a live TUI or graph view.
- Do not wire dashboard process startup/shutdown into CLI in T16.
- Do not implement web dashboard behavior.
- Do not add third-party terminal or snapshot dependencies.
