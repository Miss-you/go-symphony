# T19 New Implementation View

The current codebase already contains most of the runtime plumbing needed for the two verification flows. What is missing is not core capability, but a small operator-facing layer that packages the existing seams into a repeatable “probe Linear” path and a controlled “single-item runtime/Codex smoke” path.

## Existing seams that are already in place

1. The Linear read adapter is real and typed. `internal/trackers/linear.NewReader` validates API key and project slug, applies the default Linear endpoint, and exposes the read contract through `ListCandidates`, `ListByStates`, and `RefreshByIDs` ([`internal/trackers/linear/reader.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/trackers/linear/reader.go#L71), [`internal/config/settings.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/config/settings.go#L169), [`internal/config/settings.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/config/settings.go#L300)).

2. The runtime assembly already branches cleanly by provider. `StartRuntime` loads typed settings from `config.Store`, chooses a reader via `runtimeReader`, runs terminal cleanup before dispatch, and then starts the orchestrator and worker manager ([`internal/cli/runtime.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/cli/runtime.go#L65), [`internal/cli/runtime.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/cli/runtime.go#L91), [`internal/cli/runtime.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/cli/runtime.go#L102), [`internal/cli/runtime.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/cli/runtime.go#L253)).

3. The workflow selection seam already keeps Linear-specific tool injection out of the orchestrator. `defaultBundleFactory` calls `workflow.Select` for Linear settings and returns a no-tool memory bundle for memory settings; `runCodexLoop` then threads the selected bundle into `codex.StartSession` ([`internal/cli/runtime.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/cli/runtime.go#L283), [`internal/cli/runtime.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/cli/runtime.go#L482)).

4. The Codex app-server session layer already has the right test seam. `codex.StartSession` accepts a transport factory, tool handler, non-interactive mode, and timeout settings, then bootstraps `initialize`, `thread/start`, and turn handling without hard-coding process execution ([`internal/codex/session.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/codex/session.go#L256), [`internal/codex/session.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/codex/session.go#L319), [`internal/codex/session.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/codex/session.go#L405), [`internal/codex/session.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/codex/session.go#L529), [`internal/codex/session.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/codex/session.go#L571)).

5. The memory provider is already deterministic and isolated. `internal/trackers/memory.NewReader` snapshots its seed data, returns deep copies on read, and supports state-based and ID-based reads that are good enough for local verification without touching Linear ([`internal/trackers/memory/reader.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/trackers/memory/reader.go#L16), [`internal/trackers/memory/reader.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/trackers/memory/reader.go#L25)).

6. The current CLI still has only the main runtime launch path. `cmd/symphony/main.go` and `internal/cli/main.go` route into `StartRuntime` after the acknowledgement guard, workflow-path validation, and log setup; there is no separate operator command for “read Linear only” or “smoke one runtime item then exit” ([`cmd/symphony/main.go`](file:///Users/lihui/Documents/GitHub/go-symphony/cmd/symphony/main.go#L12), [`internal/cli/main.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/cli/main.go#L39)).

7. The tests prove the seams are usable, but only from code. Runtime tests already inject a fake reader and fake Codex transport, while the e2e test only verifies the HTTP/dashboard shell and shutdown behavior; there is still no operator-facing wrapper that exposes those seams as a command-line workflow ([`internal/cli/runtime_test.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/cli/runtime_test.go#L22), [`internal/cli/runtime_e2e_test.go`](file:///Users/lihui/Documents/GitHub/go-symphony/internal/cli/runtime_e2e_test.go#L1)).

## What is missing for the two validation goals

1. A Linear-only probe entrypoint. Operators need a way to load the current workflow config, instantiate `internal/trackers/linear.Reader`, and print or inspect candidate reads, state reads, and refresh-by-ID reads without starting workers or Codex.

2. A guarded runtime/Codex smoke entrypoint. Operators need a way to run exactly one small runtime path with a deterministic item seed and the real `codex app-server` transport, so the Codex session and worker loop can be exercised end to end without depending on live Linear data churn.

3. A seed-loading path for memory-backed smoke runs. `RuntimeOptions.MemoryItems` already exists, but it is only reachable from Go callers and tests (`internal/cli/runtime.go:34`). There is no CLI plumbing that lets an operator provide a fixture file or inline seed set and feed it into `StartRuntime`.

4. A small shared helper layer for the two commands. Right now, `main.go`, `StartRuntime`, `runtimeReader`, and `codex.StartSession` are reusable, but the code that wires “load config -> choose seam -> run one verification mode -> emit a compact result” does not exist.

## Minimal implementation shape

The smallest useful change is an operator-facing verification command package, not a change to core packages.

Suggested shape:

```text
cmd/symphony-verify
  ├── linear   -> probe Linear reads only
  └── runtime  -> seed one memory item, start runtime, run one Codex turn
```

### Linear probe mode

- Load the workflow file or typed settings through the existing config store.
- Create `linear.NewReader(settings.Provider, nil)` directly.
- Run:
  - `ListCandidates(ctx)`
  - `ListByStates(ctx, settings.Provider.TerminalStates)`
  - `RefreshByIDs(ctx, []string{...})`
- Print a compact, human-readable summary and a machine-readable option if needed.

This keeps the probe focused on the adapter seam already present in `internal/trackers/linear`, and it avoids starting the orchestrator, workspace manager, or Codex process transport.

### Runtime/Codex smoke mode

- Reuse `StartRuntime`.
- Accept a small memory-seed file or inline JSON/YAML fixture and populate `RuntimeOptions.MemoryItems`.
- Force a single-item smoke setup by using the memory provider, `max_concurrent_agents: 1`, and a narrow terminal/active-state set.
- Let the real `codex app-server` command run through `codex.StartProcessTransport`.
- Exit after the first successful turn or after a clearly bounded timeout, and print the dashboard URL plus the final snapshot summary.

This keeps the smoke path on the existing runtime seam instead of introducing a second runtime implementation.

## Why this is the right boundary

The current code already proves the important pieces separately:

- Linear read behavior lives in the Linear adapter.
- Runtime assembly lives in `internal/cli`.
- Codex protocol handling lives in `internal/codex`.
- Memory-backed deterministic reads live in `internal/trackers/memory`.

So the missing work is orchestration at the edge, not new core abstractions. The implementation should stay away from `internal/orchestrator`, `internal/tracker`, and `internal/codex` protocol internals unless a tiny helper is needed to make the operator command readable.
