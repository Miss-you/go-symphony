## Purpose

Define the workflow-selection contract for turning already-loaded `WORKFLOW.md` state and typed runtime settings into a concrete workflow bundle. The V1 selector is intentionally narrow: the only supported bundle is the Linear compatibility bundle `compat_linear_default`.

## Requirements

### Requirement: Linear workflow selection is explicit and provider-bound
The system SHALL select `compat_linear_default` when the loaded settings describe a Linear provider. The system SHALL return an explicit unsupported-provider error for any other provider kind. The system SHALL NOT silently fall back to a generic bundle.

#### Scenario: Linear settings select compat_linear_default
- **WHEN** the caller selects a workflow bundle from loaded workflow/config state whose provider kind is Linear
- **THEN** the selected bundle has the ID `compat_linear_default`
- **AND** no unsupported-provider error is returned

#### Scenario: Non-Linear settings fail explicitly
- **WHEN** the caller selects a workflow bundle from loaded workflow/config state whose provider kind is not Linear
- **THEN** the selection fails with an explicit unsupported-provider error
- **AND** no generic fallback bundle is returned

### Requirement: Linear workflow bundles use the effective prompt template
The system SHALL expose the effective prompt template for the selected Linear workflow bundle by using `config.EffectivePromptTemplate` on the loaded workflow. When the raw prompt body is blank, the selected bundle SHALL still expose Symphony's built-in default prompt template. When the raw prompt body is present, the selected bundle SHALL expose that prompt body unchanged.

#### Scenario: Blank prompt body falls back to the default template
- **WHEN** the loaded workflow body is blank and the provider is Linear
- **THEN** the selected bundle exposes Symphony's built-in default prompt template

#### Scenario: Non-blank prompt body is preserved
- **WHEN** the loaded workflow body contains prompt text and the provider is Linear
- **THEN** the selected bundle exposes the prompt text unchanged as its effective prompt template

### Requirement: Linear workflow bundles wire the existing Linear ToolBridge surface
The system SHALL wire the selected Linear workflow bundle to the existing Linear ToolBridge dynamic tool surface. The bundle SHALL expose the bridge's dynamic tool specs and the bridge itself as the tool handler. The bundle SHALL expose exactly one dynamic tool named `linear_graphql` and SHALL NOT introduce extra dynamic tools or reimplement Linear GraphQL handling in `internal/workflow`.

#### Scenario: Bundle exposes the Linear bridge tool surface
- **WHEN** the selected bundle is created for Linear settings
- **THEN** its dynamic tool list contains exactly the `linear_graphql` tool spec from the existing Linear ToolBridge
- **AND** its tool handler is the existing Linear ToolBridge handler

#### Scenario: No extra tools are introduced
- **WHEN** the selected Linear workflow bundle is inspected
- **THEN** no additional dynamic tool names are exposed beyond `linear_graphql`

### Requirement: Workflow selection remains a compatibility-shell leaf
The system SHALL keep workflow selection outside the provider-neutral core boundary. The workflow-selection capability SHALL depend only on already-loaded workflow/config state and compatibility-shell wiring. It SHALL NOT require `internal/orchestrator`, `internal/tracker`, `internal/workspace`, `internal/runner`, or `internal/domain`.

#### Scenario: Dependency guard stays out of core runtime packages
- **WHEN** the workflow-selection package dependency graph is inspected
- **THEN** it does not include `internal/orchestrator`, `internal/tracker`, `internal/workspace`, `internal/runner`, or `internal/domain`

#### Scenario: Selection uses loaded state only
- **WHEN** a caller selects a bundle from already-loaded workflow/config state
- **THEN** the selector does not need to read workflow files or own reload behavior
