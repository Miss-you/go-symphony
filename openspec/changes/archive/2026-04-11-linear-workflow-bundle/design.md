## Context

T13 sits at the boundary between the already-frozen workflow loader/config layer and the existing Linear compatibility shell. The repository already has typed config, a prompt-template helper, Codex session bootstrap, and the Linear ToolBridge. What is missing is a narrow package that chooses the concrete workflow bundle from loaded settings and exposes the values Codex startup needs without turning `internal/workflow` into a second runtime layer.

The design has to stay compatible with the current boundary rules:

- core packages remain provider-neutral
- provider-specific write behavior stays in the compatibility shell
- workflow selection must not read files, reload config, or own runtime state
- the first supported bundle is Linear-only

## Goals / Non-Goals

**Goals:**

- Provide a small `internal/workflow` selector that maps loaded workflow/config state to a concrete bundle.
- Make `compat_linear_default` the only supported bundle for this slice.
- Reuse `config.EffectivePromptTemplate` for prompt fallback.
- Reuse the existing Linear ToolBridge as the source of dynamic tool specs and tool handling.
- Return an explicit unsupported-provider error for non-Linear settings.

**Non-Goals:**

- No workflow registry or plugin system.
- No provider-agnostic fallback bundle.
- No change to orchestrator, tracker, workspace, runner, or domain behavior.
- No new workflow file loading or reload logic.
- No new `linear_graphql` implementation or argument normalization layer.

## Decisions

### 1. Use a small selector plus a value bundle, not a registry
The package should expose a simple `Select(raw config.Workflow, settings config.Settings) (Bundle, error)` entry point and return a value-shaped `Bundle`.

Why:

- The task only needs one concrete bundle.
- A registry would add indirection with no current benefit.
- A value result is easy to hand to Codex session bootstrap and easy to test.

Alternatives considered:

- **Workflow registry**: rejected because it invites generic bundle loading before there is more than one supported bundle.
- **Interface hierarchy**: rejected because the package does not need polymorphic behavior yet.

### 2. Make Linear the only supported selection path
`Select` should map `config.ProviderLinear` to `compat_linear_default` and return an explicit unsupported-provider error for any other provider kind.

Why:

- T13 is intentionally Linear-specific.
- Silent fallback would hide misconfiguration and blur the compatibility boundary.
- Returning a typed or clearly classified error keeps future bundles additive.

Alternatives considered:

- **Generic default bundle**: rejected because it would weaken the provider boundary and obscure unsupported settings.
- **Nil bundle on unsupported providers**: rejected because it pushes error handling downstream and makes startup behavior ambiguous.

### 3. Reuse the existing Linear bridge directly
`compat_linear_default` should call `toolbridge/linear.New(settings.Provider, nil)` and then surface `bridge.ToolSpecs()` and the bridge itself as the bundle tool surface.

Why:

- The Linear bridge already owns `linear_graphql` semantics.
- Rewrapping the bridge would duplicate error handling and tool normalization.
- This keeps workflow selection thin and leaves compatibility-shell logic in the compatibility shell.

Alternatives considered:

- **Rebuild the tool spec in workflow**: rejected because it duplicates bridge knowledge.
- **Adapter around the bridge**: rejected because the task does not need another abstraction layer.

### 4. Resolve prompt text through config, not workflow code
`compat_linear_default` should compute the effective prompt template with `config.EffectivePromptTemplate(raw)`.

Why:

- Workflow selection is not the owner of raw file parsing semantics.
- The config layer already defines the blank-prompt fallback behavior.
- Using the helper keeps prompt compatibility centralized.

Alternatives considered:

- **Copy the fallback template into workflow**: rejected because it would fork a config concern into a workflow concern.

### 5. Keep the bundle immutable after construction
The bundle should be returned as a plain value with no mutation API.

Why:

- Codex bootstrap only needs snapshot values.
- Immutability keeps the selector easy to reason about and test.
- The workflow layer should not become a runtime state owner.

## Risks / Trade-offs

- [Risk] A new `internal/workflow` package could drift into a generic runtime layer. → Mitigation: keep the surface to `Select` plus one bundle factory and keep dependencies limited to `internal/config`, `internal/codex`, and `internal/toolbridge/linear`.
- [Risk] Unsupported-provider behavior could be weakened by a permissive fallback. → Mitigation: require an explicit error path and test it.
- [Risk] Tool spec drift could happen if workflow duplicates Linear tool construction. → Mitigation: source `DynamicTools` from `bridge.ToolSpecs()` only.
- [Risk] The bundle could accidentally start owning reload or file-loading logic. → Mitigation: restrict the package to already-loaded `config.Workflow` and `config.Settings` values.

## Migration Plan

No data migration is required.

Implementation can land in one pass:

1. Add tests for selection, unsupported-provider behavior, prompt fallback, and tool-bridge wiring.
2. Implement `internal/workflow` with the minimal selector and `compat_linear_default` factory.
3. Wire the bundle values into the existing Codex startup path.
4. Validate package tests and repo-level build/test gates.

Rollback is straightforward: remove the selector wiring and restore the previous startup path because no persisted state changes.

## Open Questions

- Should the unsupported-provider error be a package sentinel or a typed config-style error? The design only requires that it be explicit and testable.
- Should `Bundle.ID` remain a dedicated string type or use a plain string? The current implementation plan prefers a typed ID for clarity, but the external behavior is the same either way.
