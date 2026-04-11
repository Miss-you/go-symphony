# T13 Final Implementation Plan v1

## Goal

Implement the first Go-native workflow selection layer for Symphony and the first concrete workflow bundle, `compat_linear_default`.

This task should:

- keep `WORKFLOW.md` as the source of prompt text and workflow config
- select a concrete workflow bundle from already-loaded `config.Workflow` + `config.Settings`
- wire the Linear compatibility shell into Codex through `DynamicTools` and `ToolHandler`
- preserve the current Linear-specific unattended posture without inventing a generic workflow system

## Non-Goals

T13 does not:

- add a provider-agnostic workflow registry
- add a universal tracker write API
- add a generic workpad abstraction
- add user-selectable workflow names or multiple bundles
- add non-Linear workflow bundles
- change orchestrator, tracker, workspace, or runner behavior
- change `config.Store` hot reload or last-known-good semantics
- reimplement `linear_graphql` semantics inside `internal/workflow`

## Symphony Parity Decisions and Exact Boundaries

### Preserve

- `WORKFLOW.md` remains the runtime source for both prompt text and front matter config.
- Prompt rendering still uses the existing config fallback behavior:
  - blank body falls back to `config.EffectivePromptTemplate`
  - prompt text itself is not re-parsed or reloaded by `internal/workflow`
- The first workflow bundle is explicitly Linear-specific.
- `linear_graphql` remains the only dynamic tool in the bundle.
- The bundle must be compatible with Codex session bootstrap without extra glue.

### Do not preserve as a new abstraction

- No generic workflow engine.
- No workflow lifecycle state machine.
- No provider-neutral tool registry.
- No fallback bundle for unsupported providers.

### Boundary rules

`internal/workflow` may depend on:

- `internal/config`
- `internal/codex`
- `internal/toolbridge/linear`

`internal/workflow` must not depend on:

- `internal/orchestrator`
- `internal/tracker`
- `internal/workspace`
- `internal/runner`
- `internal/domain`

Workflow selection should consume the already-loaded config snapshot. It should not read files, watch files, or own reload behavior.

## Proposed Go API and Files

Add a small, value-oriented package under `internal/workflow`:

### `internal/workflow/doc.go`

Document the package as the workflow selector and bundle factory.

### `internal/workflow/types.go`

Define the public bundle shape:

```go
type ID string

const CompatLinearDefault ID = "compat_linear_default"

type Bundle struct {
    ID             ID
    PromptTemplate string
    DynamicTools   []codex.ToolSpec
    ToolHandler    codex.ToolHandler
}
```

Bundle invariants:

- immutable after construction
- `DynamicTools` is safe to pass directly to `codex.SessionOptions`
- `PromptTemplate` is the effective template, not the raw body

### `internal/workflow/select.go`

Implement selection:

```go
func Select(raw config.Workflow, settings config.Settings) (Bundle, error)
```

Selection rules:

- `settings.Provider.Kind == config.ProviderLinear` -> `compat_linear_default`
- any other provider kind -> explicit unsupported error
- no default fallback to a fake generic bundle

### `internal/workflow/compat_linear_default.go`

Implement:

```go
func CompatLinearDefault(raw config.Workflow, settings config.Settings) (Bundle, error)
```

The function should:

- compute `PromptTemplate` with `config.EffectivePromptTemplate(raw)`
- create a Linear bridge with `toolbridge/linear.New(settings.Provider, nil)`
- assign `bridge.ToolSpecs()` to `DynamicTools`
- assign the bridge itself to `ToolHandler`
- return `ID: CompatLinearDefault`

If `toolbridge/linear.New` fails, return the error without wrapping it into a new workflow abstraction.

### `internal/workflow/select_test.go`

Add package tests for selection and unsupported-provider behavior.

### `internal/workflow/compat_linear_default_test.go`

Add package tests for bundle shape, prompt fallback, and bridge wiring.

## Expected `compat_linear_default` Behavior

The bundle should mirror the current Linear default workflow posture as closely as possible while staying Go-native:

- prompt text comes from the loaded `WORKFLOW.md` body
- if the body is blank, the bundle still exposes the default Linear prompt template from config
- the bundle is unattended by default because the workflow text and Codex configuration already encode that behavior
- the bundle exposes exactly one dynamic tool: `linear_graphql`
- the tool handler is the existing Linear bridge, not a wrapper that reinterprets arguments

The integration shape should remain simple:

```go
workflowBundle, err := workflow.Select(rawWorkflow, settings)
if err != nil {
    return err
}

session, err := codex.StartSession(ctx, codex.SessionOptions{
    Config: codex.Config{
        DynamicTools: workflowBundle.DynamicTools,
        // other config fields supplied by the existing startup path
    },
    ToolHandler: workflowBundle.ToolHandler,
})
```

`internal/workflow` should produce the values needed for this wiring, but it should not start sessions itself.

## Dynamic Tool Wiring

Use the existing Linear bridge as the single source of dynamic tool truth.

Wiring rules:

- `toolbridge/linear.New(settings.Provider, nil)` creates the bridge
- `bridge.ToolSpecs()` becomes the bundle tool list
- `bridge` itself becomes the bundle tool handler
- `linear_graphql` argument normalization, auth handling, and error shape remain in `internal/toolbridge/linear`

The workflow package should not duplicate:

- tool schema construction
- `linear_graphql` argument validation
- Linear GraphQL transport/error formatting

## TDD Plan

Write tests before implementation changes in `internal/workflow`.

### Step 1: selection tests

Create a failing test that proves:

- `Select` returns `compat_linear_default` for `ProviderLinear`
- unsupported provider kinds return an explicit error

### Step 2: bundle wiring tests

Create a failing test that proves:

- `PromptTemplate` comes from `config.EffectivePromptTemplate`
- the bundle tool list contains exactly the Linear bridge surface expected here
- the bundle handler accepts `linear_graphql`
- the bundle does not expose extra tools

### Step 3: integration shape test

Create a compile-time or package-level test that shows the bundle can be handed directly to `codex.SessionOptions` without adapter code.

### Step 4: implementation

Implement the minimal code needed to make the tests pass.

## Verification Plan

Run verification in this order:

1. `go test ./internal/workflow/...`
2. if the package wiring touches compile edges, expand to `go test ./internal/workflow/... ./internal/codex/... ./internal/toolbridge/...`
3. after the package gate is green, run the repo-level compile gate relevant to the change, at minimum `go test ./...`
4. if the broader tree remains stable, run `make build` and `make lint`

Do not mark the task as complete unless the package gate passes and the workflow bundle can be consumed by Codex with no extra translation layer.

## Risks

- The biggest risk is overdesign. A registry, provider strategy hierarchy, or generic default workflow would add structure without a real need.
- Another risk is boundary drift. If `internal/workflow` starts reading files, reloading config, or talking to orchestrator/tracker code, the package stops being a selector and becomes a second runtime layer.
- Another risk is tool drift. `linear_graphql` must stay in `internal/toolbridge/linear`; `internal/workflow` should only reference the bridge.
- Another risk is accidental fallback behavior. The first bundle is intentionally Linear-only, so unsupported providers should fail explicitly.

## Explicit Deferments

Defer these items to later work:

- additional provider workflow bundles
- any workflow selector driven by a user-supplied bundle name
- generic workflow registry or plugin loading
- non-Linear dynamic tools
- runtime session startup changes beyond passing through the bundle values
- any workflow-specific persistence beyond what `config.Store` already provides

## Deliverable Definition

T13 is complete when:

- `internal/workflow` exposes a small selector and a Linear bundle factory
- `compat_linear_default` is wired to the existing Linear bridge
- tests prove bundle selection, prompt fallback, and dynamic tool wiring
- the implementation stays within the provider-specific compatibility shell boundary
