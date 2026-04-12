## ADDED Requirements

### Requirement: CLI requires explicit guardrails acknowledgement
The CLI MUST require the Symphony guardrails acknowledgement flag before performing startup side effects.

#### Scenario: Acknowledgement missing stops before startup
- **WHEN** the CLI is invoked without `--i-understand-that-this-will-be-running-without-the-usual-guardrails`
- **THEN** it exits nonzero with the guardrails acknowledgement banner
- **AND** it does not check the workflow file, configure logs, apply port overrides, or start the runtime

#### Scenario: Acknowledgement present allows validation
- **WHEN** the CLI is invoked with `--i-understand-that-this-will-be-running-without-the-usual-guardrails`
- **THEN** it proceeds to argument validation and workflow startup

### Requirement: CLI parses workflow path and host options compatibly
The CLI SHALL accept an optional workflow path plus `--logs-root` and `--port` options using Symphony-compatible validation and precedence.

#### Scenario: Default workflow path
- **WHEN** no workflow path is provided
- **THEN** the CLI uses `WORKFLOW.md` from the current working directory

#### Scenario: Explicit workflow path
- **WHEN** one workflow path is provided
- **THEN** the CLI expands that path and uses it for runtime startup

#### Scenario: Flags are order agnostic
- **WHEN** valid flags appear before or after the workflow path
- **THEN** the CLI parses the same workflow path and option values

#### Scenario: Invalid arguments return usage
- **WHEN** the CLI receives an unknown flag, missing option value, blank logs root, invalid port, negative port, or more than one workflow path
- **THEN** it exits nonzero with the Symphony usage text

#### Scenario: Logs root selects log file
- **WHEN** `--logs-root <path>` is provided
- **THEN** the CLI expands `<path>` and configures process logs under `<path>/log/symphony.log`

#### Scenario: Port override wins
- **WHEN** `--port <port>` is provided and `WORKFLOW.md` also contains `server.port`
- **THEN** the runtime uses the CLI port override
- **AND** port `0` requests an ephemeral listener

### Requirement: CLI surfaces startup failure cleanly
The CLI MUST fail before running when required startup inputs are invalid, and it MUST return clear operator-facing error text.

#### Scenario: Missing workflow file
- **WHEN** the selected workflow file does not exist
- **THEN** the CLI exits nonzero with `Workflow file not found: <expanded path>`
- **AND** it does not start the runtime

#### Scenario: Runtime startup failure
- **WHEN** runtime startup fails after workflow validation
- **THEN** the CLI exits nonzero with `Failed to start Symphony with workflow <expanded path>: <reason>`
- **AND** it does not render the offline shutdown frame

### Requirement: CLI renders offline status on normal shutdown
The CLI SHALL render Symphony's minimal offline dashboard frame exactly once when the runtime shuts down normally.

#### Scenario: Normal shutdown
- **WHEN** the runtime has started and the process context is canceled normally
- **THEN** the CLI closes the runtime
- **AND** it writes a terminal frame containing `SYMPHONY STATUS`, `app_status=offline`, and the closing border
- **AND** the frame contains no timestamp and no normal running or retry sections
- **AND** the CLI exits with status code `0`

#### Scenario: Shutdown close failure
- **WHEN** runtime close returns an error during shutdown
- **THEN** the CLI reports the close error on stderr
- **AND** it exits nonzero after best-effort cleanup

### Requirement: Final parity verification is explicit
The implementation MUST maintain a parity-oriented verification matrix for CLI/bootstrap behavior and document any live-provider e2e limits.

#### Scenario: No-network e2e smoke
- **WHEN** e2e-tagged tests run without Linear or Codex credentials
- **THEN** the suite includes a no-network runtime smoke test that starts the memory path, serves the dashboard/API on an ephemeral port, and shuts down cleanly

#### Scenario: Real-provider e2e unavailable
- **WHEN** real Linear/Codex credentials are not available
- **THEN** live-provider e2e is explicitly skipped or documented as not run instead of being silently treated as passed
