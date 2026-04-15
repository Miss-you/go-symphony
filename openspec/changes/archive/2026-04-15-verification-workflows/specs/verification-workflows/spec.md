## ADDED Requirements

### Requirement: Linear probe verifies tracker reads without starting runtime
The verification command SHALL provide a read-only Linear probe that loads the selected workflow settings and exercises the configured Linear reader without starting the runtime, workspaces, orchestrator, or Codex app-server.

#### Scenario: Probe reads Linear surfaces
- **WHEN** an operator runs `symphony-verify linear` with a Linear workflow
- **THEN** the command loads runtime-compatible typed settings
- **AND** calls candidate, terminal-state, and refresh-by-ID reader paths
- **AND** prints a compact report with counts and summarized work items

#### Scenario: Probe rejects unsupported provider
- **WHEN** the selected workflow does not configure the Linear provider
- **THEN** the command exits nonzero before creating a Linear reader or starting runtime components

#### Scenario: Probe is testable without credentials
- **WHEN** package tests exercise the probe success path
- **THEN** they can inject a fake reader and do not require real Linear credentials or network access

### Requirement: Runtime smoke is scoped to one explicit issue
The verification command SHALL provide a guarded runtime smoke path that runs the existing Symphony runtime against a single explicit work item identifier.

#### Scenario: Smoke requires guardrails acknowledgement
- **WHEN** an operator runs `symphony-verify run` without the guardrails acknowledgement flag
- **THEN** the command exits nonzero before loading workflow state or starting runtime components

#### Scenario: Smoke requires selected issue
- **WHEN** an operator runs `symphony-verify run` without `--only-issue <id-or-identifier>`
- **THEN** the command exits nonzero before starting runtime components

#### Scenario: Smoke filters runtime reads
- **WHEN** `symphony-verify run --only-issue ABC-123` starts runtime
- **THEN** candidate, terminal-state, and refresh reads passed to the runtime include only work items whose normalized `ID` or `Identifier` matches `ABC-123`
- **AND** the existing production `cmd/symphony` behavior remains unchanged

### Requirement: Verification workflows are documented
The repository SHALL document the two-stage verification flow for operators.

#### Scenario: Operator follows staged verification docs
- **WHEN** an operator opens the verification workflow documentation
- **THEN** it includes copyable commands for the Linear probe and single-issue runtime smoke
- **AND** warns that the runtime smoke launches real Codex and may mutate the workspace and Linear issue
