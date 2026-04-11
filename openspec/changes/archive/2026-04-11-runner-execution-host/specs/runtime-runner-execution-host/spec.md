## ADDED Requirements

### Requirement: Host-aware command execution contract
The system SHALL provide a runner execution contract that runs shell commands through one host-aware API for both local and SSH execution.

The contract MUST accept a command, working directory, timeout, and optional worker host. It MUST execute local commands without SSH when the worker host is empty, and it MUST execute remote commands through SSH when the worker host is non-empty.

Command results MUST preserve combined output and process exit status. Timeout and process-start failures MUST be distinguishable from non-zero command exits.

#### Scenario: Local command executes in requested directory
- **WHEN** a command request has an empty worker host and a working directory
- **THEN** runner executes the command locally in that working directory
- **AND** returns the command output and exit status

#### Scenario: SSH command uses remote transport
- **WHEN** a command request has a non-empty worker host
- **THEN** runner constructs an SSH invocation for that host
- **AND** wraps the remote command through `bash -lc`
- **AND** does not use workspace package logic to decide SSH arguments

#### Scenario: Command timeout is distinct
- **WHEN** command execution exceeds the request timeout
- **THEN** runner returns a timeout error distinguishable from a non-zero command exit

### Requirement: SSH argument construction
The system SHALL construct SSH command arguments compatibly with the current Symphony transport behavior.

Runner MUST use `ssh -T`, MUST include `-F <config>` when `SYMPHONY_SSH_CONFIG` is set, MUST parse `host:port` into `-p <port> <host>`, MUST preserve user prefixes such as `user@host:port`, and MUST handle bracketed IPv6 hosts without treating IPv6 colons as port separators.

#### Scenario: Host port is parsed
- **WHEN** runner builds an SSH invocation for `worker.example:2222`
- **THEN** the SSH arguments include `-p 2222`
- **AND** the target host argument is `worker.example`

#### Scenario: SSH config env var is honored
- **WHEN** `SYMPHONY_SSH_CONFIG` is set
- **THEN** runner includes `-F <config>` in the SSH arguments

#### Scenario: Bracketed IPv6 does not misparse
- **WHEN** runner builds an SSH invocation for a bracketed IPv6 host
- **THEN** runner does not split IPv6 address colons as a port separator

### Requirement: Stateless worker-host selection
The system SHALL provide a deterministic worker-host selector for runner-backed admission.

The selector MUST use configured SSH hosts, optional per-host capacity, preferred host, and caller-supplied host-load data. It MUST NOT retain mutable runtime state.

When no SSH hosts are configured, the selector MUST admit local execution with an empty host. When hosts are configured, the selector MUST honor an eligible preferred host first, otherwise choose the least-loaded eligible configured host using configuration order as the stable tie-breaker. If every configured host is at capacity, the selector MUST reject admission.

#### Scenario: No hosts admits local execution
- **WHEN** no SSH hosts are configured
- **THEN** runner host selection admits the run with an empty host

#### Scenario: Preferred host is eligible
- **WHEN** a preferred host is configured and below capacity
- **THEN** runner host selection returns the preferred host

#### Scenario: Least-loaded fallback is deterministic
- **WHEN** no preferred host is eligible and multiple configured hosts have capacity
- **THEN** runner host selection returns the least-loaded eligible host
- **AND** ties are broken by configured host order

#### Scenario: All hosts full rejects admission
- **WHEN** all configured hosts are at the per-host capacity
- **THEN** runner host selection rejects admission
