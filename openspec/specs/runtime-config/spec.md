## Purpose

Define the normative typed runtime-config contract for the Go runtime so later core packages can consume provider-neutral settings derived from `WORKFLOW.md` without reparsing raw YAML maps.

## Requirements

### Requirement: Typed Runtime Settings
The Go runtime SHALL normalize the `WORKFLOW.md` front matter into a typed `Settings` model that downstream runtime packages consume instead of reparsing raw YAML maps.

#### Scenario: Legacy tracker input normalizes to provider settings
- **WHEN** the workflow front matter contains the legacy `tracker.*` configuration used by Symphony
- **THEN** the config layer returns typed provider-neutral settings under `Settings.Provider`

#### Scenario: LoadSettings returns typed settings directly
- **WHEN** a caller loads workflow config through `LoadSettings`
- **THEN** it receives a typed `Settings` value without reading `Workflow.Config` directly

#### Scenario: CurrentSettings returns typed settings from the store
- **WHEN** a runtime caller asks the reload store for the current typed config through `CurrentSettings`
- **THEN** it receives the cached typed `Settings` value without reparsing raw YAML maps

### Requirement: Runtime Config Defaults And Resolution
The Go runtime SHALL apply Symphony-compatible defaults, env fallbacks, and path handling while normalizing typed settings.

#### Scenario: Concrete defaults are applied
- **WHEN** optional runtime config fields are omitted from `WORKFLOW.md`
- **THEN** the typed settings use the documented defaults for provider endpoint, polling interval, workspace root, agent limits, Codex timeouts, hooks timeout, observability refresh, and server host

#### Scenario: Linear env fallbacks resolve
- **WHEN** a Linear workflow omits `tracker.api_key` or `tracker.assignee`, or references them through env tokens
- **THEN** the config layer resolves those values from `LINEAR_API_KEY` and `LINEAR_ASSIGNEE` using Symphony-compatible behavior

#### Scenario: Workspace root resolves env tokens and home expansion
- **WHEN** `workspace.root` uses `$VAR` indirection or a `~`-prefixed local path
- **THEN** the config layer resolves the env token before path handling, expands `~` for local paths, and falls back to the default workspace root when the resolved value is missing or empty

### Requirement: Typed Validation Fails Before Runtime Use
The Go runtime SHALL validate typed settings in `internal/config` before runtime code consumes them.

#### Scenario: Unsupported provider kind is rejected
- **WHEN** `tracker.kind` is not one of the supported values `linear` or `memory`
- **THEN** typed config normalization fails with a configuration error

#### Scenario: Linear config requires project and credentials
- **WHEN** `tracker.kind` is `linear` and the normalized config is missing the required API key or project slug
- **THEN** typed config normalization fails with a configuration error

#### Scenario: Invalid typed config rejects startup
- **WHEN** the workflow file parses successfully but typed settings validation fails during initial load
- **THEN** store startup fails instead of booting with semantically invalid runtime config

### Requirement: Reload Keeps Raw And Typed Config In Sync
The Go runtime SHALL cache raw workflow and typed settings as one atomic last-known-good snapshot during reload.

#### Scenario: Valid reload replaces the whole snapshot
- **WHEN** the active workflow file changes to content that raw-parses and typed-validates successfully
- **THEN** the store atomically replaces the cached raw workflow and typed settings together

#### Scenario: Typed normalization failure keeps the previous snapshot
- **WHEN** the workflow file raw-parses successfully but typed normalization or validation fails during reload
- **THEN** the store keeps the previous raw workflow and typed settings snapshot active

#### Scenario: Failed path switch keeps retrying the requested path
- **WHEN** the desired workflow path changes to a missing or invalid file after a known-good snapshot already exists
- **THEN** the store keeps serving the previous snapshot while continuing to retry the requested path on future reload attempts
