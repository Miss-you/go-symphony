## Why

`go-symphony` already loads `WORKFLOW.md` and has the Linear compatibility shell in place, but it still lacks a small Go-native selection layer that turns the loaded config into a concrete workflow bundle. T13 fills that gap by making the initial workflow explicitly Linear-specific while keeping the selection boundary narrow and compatible with the existing runtime shape.

## What Changes

- Add a new workflow-selection capability that chooses the active workflow bundle from loaded `config.Workflow` and typed `config.Settings`.
- Introduce the first concrete bundle, `compat_linear_default`, for `config.ProviderLinear`.
- Expose the bundle values needed to bootstrap Codex session startup: effective prompt template, dynamic tool specs, and the existing Linear ToolBridge handler.
- Return an explicit unsupported-provider error for non-Linear settings instead of inventing a generic fallback bundle.
- Keep workflow selection out of orchestrator, tracker, workspace, and runner packages.

## Capabilities

### New Capabilities
- `workflow-selection`: Select a concrete workflow bundle from loaded workflow config and typed settings, with the first supported bundle being `compat_linear_default` for Linear settings.

### Modified Capabilities
- None

## Impact

- Adds a new `internal/workflow` package boundary for bundle selection and compatibility-shell wiring.
- Reuses `internal/config` for effective prompt resolution and typed settings, and reuses `internal/toolbridge/linear` for dynamic tool specs and tool handling.
- Affects Codex session bootstrap code that consumes dynamic tool specs and a tool handler.
- Does not change tracker, domain, orchestrator, workspace, or runner behavior.
