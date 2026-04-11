# T13 New Implementation Shape

## Scope

This note documents the clean Go-native shape for workflow selection and the first concrete `compat_linear_default` workflow bundle, based on the current repository state and the OpenSpec parity contract.

I inspected these paths directly:

- `internal/config/workflow.go`
- `internal/config/settings.go`
- `internal/config/store.go`
- `internal/codex/session.go`
- `internal/toolbridge/linear/bridge.go`
- `internal/orchestrator/state.go`
- `internal/orchestrator/service.go`
- `internal/domain/types.go`
- `internal/workflow/doc.go`
- `openspec/specs/workflow-loader/spec.md`
- `openspec/specs/runtime-config/spec.md`
- `openspec/specs/linear-toolbridge/spec.md`
- `openspec/specs/codex-app-server-protocol/spec.md`
- `openspec/specs/runtime-orchestrator/spec.md`
- `openspec/specs/compatibility-contract/spec.md`
- `docs/plans/2026-04-10-go-symphony-design.md`

## What Already Exists

The repo already has the right low-level hooks for T13:

- `internal/config/workflow.go` owns raw `WORKFLOW.md` loading, prompt extraction, and `EffectivePromptTemplate`.
- `internal/config/settings.go` normalizes the legacy workflow front matter into typed `Settings`, including `Settings.Provider.Kind`.
- `internal/config/store.go` caches the raw `Workflow` and typed `Settings` atomically through `Current()` and `CurrentSettings()`.
- `internal/codex/session.go` already accepts `DynamicTools` and a single injected `ToolHandler`, and it preserves raw string tool arguments plus top-level `contentItems`.
- `internal/toolbridge/linear/bridge.go` already provides the Linear compatibility-shell bridge with one dynamic tool, `linear_graphql`, and provider-specific write helpers.
- `internal/orchestrator/service.go` and `state.go` remain provider-neutral and own runtime mutation only; they have no workflow-specific logic today.
- `internal/domain/types.go` contains no workflow or provider-specific write fields, which is the correct boundary.

The important structural fact is that there is no usable `internal/workflow` API yet. The package exists only as a placeholder, so T13 should define a small coordination layer rather than a broad new subsystem.

## Current Boundary Reading

The current contracts point in one direction:

```
config.Store.Current()
config.Store.CurrentSettings()
        |
        v
internal/workflow.Select(...)
        |
        v
workflow bundle
  - prompt/template
  - Codex dynamic tool specs
  - injected tool handler
        |
        v
codex.StartSession(...)
```

That means `internal/workflow` should be a thin selector and bundle factory, not:

- a second config loader
- a prompt parser
- a tracker write layer
- an orchestrator coordinator
- a universal provider abstraction

## Proposed Go-Native Shape

Keep `internal/workflow` value-oriented and small:

```go
package workflow

type ID string

const CompatLinearDefault ID = "compat_linear_default"

type Bundle struct {
    ID             ID
    PromptTemplate string
    DynamicTools   []codex.ToolSpec
    ToolHandler    codex.ToolHandler
}

func Select(raw config.Workflow, settings config.Settings) (Bundle, error)
func CompatLinearDefault(raw config.Workflow, settings config.Settings) (Bundle, error)
```

Why this shape fits the repo:

- `Bundle` is a plain immutable value, which keeps the workflow layer easy to test.
- `Select` can stay a simple switch on the typed provider kind for V1.
- `CompatLinearDefault` can be the only concrete bundle for now, with future bundles added by explicit cases rather than a registry framework.
- The selector consumes the already-loaded raw `Workflow` and typed `Settings`, so it does not re-open or reparse files.

I would not add interfaces unless a second provider bundle actually needs them. The current task is about a single Linear-specific bundle, so a function plus a small value type is enough.

## `compat_linear_default` Wiring

The first concrete bundle should be Linear-specific and should assemble the existing bridge rather than inventing new runtime behavior.

Recommended wiring:

1. Take the raw `config.Workflow` and typed `config.Settings`.
2. Derive the effective prompt template with `config.EffectivePromptTemplate(raw)`.
3. Build the Linear bridge with `toolbridge/linear.New(settings.Provider, nil)`.
4. Set the bundle's `DynamicTools` to `bridge.ToolSpecs()`.
5. Set the bundle's `ToolHandler` to the bridge itself.

That keeps the ownership split clean:

- `internal/toolbridge/linear` owns the provider-specific write surface and the `linear_graphql` tool.
- `internal/workflow` decides when that bridge is part of the active runtime bundle.
- `internal/codex` stays generic and just accepts the handler and dynamic tool list.

This is the right place to connect tool injection because `codex.SessionOptions` already has the exact extension points needed:

- `DynamicTools []ToolSpec`
- `ToolHandler ToolHandler`

So `compat_linear_default` does not need a custom Codex wrapper or a separate workflow runtime type. It just assembles the values Codex already knows how to consume.

## Selection Rules

For V1, selection should stay explicit and provider-keyed:

- `linear` -> `compat_linear_default`
- anything else -> return an error or an explicit unsupported-workflow result

I would avoid a fake fallback bundle. The design and compatibility contract already say the first workflow is explicitly Linear-specific and that there is no provider-agnostic default workflow.

If future work needs user-selected workflow names, that should come through a dedicated config key or selector input later. It should not be smuggled into the current bundle as an implied default.

## Testing Implications

The tests for `internal/workflow` should prove the coordination shape, not just compile the package:

- selection returns `compat_linear_default` for `ProviderLinear`
- the returned prompt template comes from `config.EffectivePromptTemplate`
- the returned dynamic tools contain exactly the Linear bridge surface expected for this slice
- the returned tool handler dispatches `linear_graphql`
- unsupported provider kinds do not silently fall back to a generic bundle

At the repo boundary, the later integration gate should confirm that the bundle output can be passed directly into `codex.StartSession` without extra glue.

## Risks

- The biggest risk is accidental overdesign. A registry, strategy hierarchy, or generic provider bundle abstraction would add complexity before there is evidence for it.
- A second risk is selection drift. If workflow selection starts reading raw YAML again instead of using `config.Store.Current()` / `CurrentSettings()`, the code will duplicate parsing and weaken last-known-good semantics.
- A third risk is boundary creep. `internal/workflow` should not import `internal/orchestrator`, `internal/tracker`, or `internal/domain`.
- A final risk is a fake generic default. That would violate the compatibility contract and blur the fact that the first bundle is intentionally Linear-specific.

## Bottom Line

The clean V1 shape is:

- `config` owns loading and typed normalization
- `workflow` owns a tiny selector plus `compat_linear_default`
- `toolbridge/linear` owns Linear tool/write behavior
- `codex` consumes the assembled handler and tool specs
- `orchestrator` stays workflow-agnostic and provider-neutral

That is small enough to land cleanly and still leaves room for later provider bundles without forcing a new abstraction today.
