## Purpose

Define the workspace lifecycle contract for `internal/workspace`, including deterministic naming, path safety, lifecycle hooks, create/reuse/remove behavior, and orchestrator-driven terminal cleanup handoff.

## Requirements

### Requirement: Deterministic workspace naming and path safety
The system SHALL derive a workspace path from the resolved workspace root and a normalized identifier, and it SHALL reject unsafe local paths.

The system MUST normalize identifiers by replacing every character outside `[a-zA-Z0-9._-]` with `_`, and it MUST use `issue` when the identifier is empty or nil-equivalent.

For local filesystem paths, the system MUST canonicalize the configured root and the derived workspace path before accepting the result, and it MUST reject:
- the workspace root itself
- paths outside the configured root
- symlink escapes that resolve outside the configured root

#### Scenario: Deterministic path for a safe identifier
- **WHEN** the system derives a workspace path for identifier `MT-123` under a resolved root
- **THEN** the same identifier always maps to the same workspace path under that root

#### Scenario: Unsafe identifier is normalized
- **WHEN** the system derives a workspace path for identifier `MT/123:alpha`
- **THEN** the identifier is normalized to a safe path segment before path construction

#### Scenario: Root equality is rejected
- **WHEN** the derived workspace path resolves to the canonical workspace root itself
- **THEN** the system rejects the path as unsafe

#### Scenario: Symlink escape is rejected
- **WHEN** a workspace root symlink resolves outside the canonical root boundary
- **THEN** the system rejects the derived local workspace path

### Requirement: Workspace creation and run-phase hook ordering
The system SHALL create a workspace if needed, reuse an existing directory without deleting its contents, replace stale non-directory paths with a directory, and run `after_create` only when a workspace is newly created.

The system SHALL provide a run-phase lifecycle helper that executes `before_run`, then the run body only if `before_run` succeeds, and then `after_run` on every exit path after workspace creation succeeds.

The system MUST treat `after_create` and `before_run` as blocking and fatal, and it MUST treat `after_run` as best-effort and non-fatal.

#### Scenario: Existing workspace is reused
- **WHEN** the system creates a workspace where a directory already exists
- **THEN** the directory is reused and `after_create` is not run

#### Scenario: Stale file path is replaced
- **WHEN** the system creates a workspace where a non-directory path exists
- **THEN** the stale path is replaced with a directory before the workspace is used

#### Scenario: `before_run` failure still triggers `after_run`
- **WHEN** `before_run` fails before the run body starts
- **THEN** `after_run` still executes before the lifecycle helper returns

#### Scenario: Run-body failure still triggers `after_run`
- **WHEN** the run body fails after workspace creation and `before_run` succeeds
- **THEN** `after_run` still executes before the lifecycle helper returns

### Requirement: Best-effort removal and terminal cleanup fan-out
The system SHALL remove workspaces when the orchestrator requests cleanup, and it SHALL run `before_remove` as a best-effort hook that must not block removal.

The system SHALL treat terminal cleanup as orchestrator-driven rather than workspace-driven, and the shared cleanup entry point MUST support both runtime terminal cleanup and startup terminal sweeps.

When no explicit worker host is supplied, the system MUST preserve Elixir-style fan-out cleanup across the configured worker-host list for the identifier.

#### Scenario: Cleanup removes the workspace path
- **WHEN** the orchestrator requests cleanup for a workspace path
- **THEN** the system removes the workspace even if `before_remove` fails

#### Scenario: Non-terminal invalidation does not clean up
- **WHEN** a running item is invalidated for a non-terminal reason
- **THEN** the system does not remove the workspace implicitly

#### Scenario: Hostless terminal sweep fans out
- **WHEN** the shared cleanup entry point is called without an explicit worker host and worker hosts are configured
- **THEN** the system removes the identifier's workspaces across all configured worker hosts

### Requirement: Structured lifecycle errors
The system SHALL surface structured lifecycle errors that distinguish path validation failures, hook failures, workspace preparation failures, and workspace removal failures.

The system MUST preserve enough context in the error to distinguish:
- invalid or unreadable workspace paths
- root collision
- outside-root rejection
- symlink escape
- hook timeout
- hook command failure
- workspace prepare failure
- workspace remove failure

#### Scenario: Hook timeout is distinguishable
- **WHEN** a lifecycle hook exceeds the configured timeout
- **THEN** the returned error identifies the timeout as a hook failure rather than a path or removal failure

#### Scenario: Prepare failure is distinguishable
- **WHEN** workspace creation cannot complete successfully
- **THEN** the returned error identifies the failure as workspace preparation failure

### Requirement: Workspace delegates command execution to runner
The workspace lifecycle package SHALL keep ownership of workspace path, hook, create/reuse/remove, and cleanup policy while delegating local and SSH command execution to the runner contract.

Workspace MUST use runner command execution for lifecycle hooks and host-addressed remote workspace commands. Workspace MUST NOT construct SSH transport arguments or launch local shell processes directly.

#### Scenario: Hook command uses runner executor
- **WHEN** workspace runs a configured lifecycle hook
- **THEN** it sends the command, workspace path, timeout, and worker host to runner execution
- **AND** preserves the existing fatal or best-effort hook policy

#### Scenario: Remote lifecycle command uses runner executor
- **WHEN** workspace performs a host-addressed remote create or remove operation
- **THEN** workspace keeps the lifecycle decision and remote workspace command content
- **AND** delegates command execution to runner for the selected host

#### Scenario: Lifecycle policy remains in workspace
- **WHEN** workspace creates, reuses, removes, or cleans up a workspace
- **THEN** path derivation, safety validation, hook ordering, and cleanup fan-out remain workspace-owned behavior
