## ADDED Requirements

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
