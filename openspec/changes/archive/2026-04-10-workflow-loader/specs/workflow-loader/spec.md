## ADDED Requirements

### Requirement: Workflow Path Resolution
The Go runtime SHALL resolve the active `WORKFLOW.md` path from an explicit runtime override when one is set; otherwise it SHALL use `WORKFLOW.md` in the current process working directory.

#### Scenario: Explicit path override wins
- **WHEN** a caller sets an explicit workflow file path
- **THEN** the loader uses that path instead of the default cwd path

#### Scenario: Default path uses current working directory
- **WHEN** no explicit workflow file path is configured
- **THEN** the loader uses `<current-working-directory>/WORKFLOW.md`

### Requirement: Workflow File Parsing
The Go runtime SHALL parse `WORKFLOW.md` as Markdown with optional YAML front matter and expose the decoded root map together with the trimmed prompt body.

#### Scenario: Prompt-only workflow file
- **WHEN** the workflow file does not start with `---`
- **THEN** the loader returns an empty config map and the full file contents as the trimmed prompt template

#### Scenario: Workflow file with front matter and prompt body
- **WHEN** the workflow file starts with `---`, contains YAML front matter, and later closes with `---`
- **THEN** the loader returns the decoded root map as config and the remaining trimmed body as the prompt template

#### Scenario: Unterminated front matter
- **WHEN** the workflow file starts with `---` and does not contain a closing `---`
- **THEN** the loader treats the remainder of the file as front matter and returns an empty prompt template

### Requirement: Typed Workflow Load Errors
The Go runtime SHALL surface typed loader failures for missing files, YAML parse failures, and front matter that does not decode to a root map.

#### Scenario: Missing workflow file
- **WHEN** the loader reads a workflow path that does not exist
- **THEN** it returns a typed missing-workflow-file error that includes the path context

#### Scenario: Invalid YAML
- **WHEN** the workflow front matter cannot be decoded as YAML
- **THEN** the loader returns a typed workflow-parse error

#### Scenario: Non-map front matter
- **WHEN** the workflow front matter decodes successfully but the root value is not a map
- **THEN** the loader returns a typed workflow-front-matter-not-map error

### Requirement: Blank Prompt Compatibility Helper
The Go runtime SHALL preserve a blank workflow prompt in the raw loaded workflow object and also expose a narrow compatibility helper that returns Symphony's built-in default prompt template for downstream prompt-building code.

#### Scenario: Raw blank prompt stays blank
- **WHEN** the workflow file parses successfully and its prompt body is blank
- **THEN** the loaded workflow stores an empty prompt template string

#### Scenario: Compatibility helper returns the default prompt template
- **WHEN** downstream code asks for the effective prompt template from a workflow whose prompt body is blank
- **THEN** the helper returns the built-in default prompt template instead of an empty string

### Requirement: Hot Reload Preserves Last Known Good
The Go runtime SHALL cache the last known good workflow, poll the active workflow file for changes every second, and keep serving the cached workflow when a reload fails after a prior successful load.

#### Scenario: Valid reload replaces cached workflow
- **WHEN** the active workflow file changes to another valid workflow
- **THEN** the store replaces the cached workflow with the new loaded workflow

#### Scenario: Invalid reload keeps the previous workflow
- **WHEN** the active workflow file changes to invalid content after a valid workflow was already loaded
- **THEN** the store logs the reload failure and keeps the previous cached workflow active

#### Scenario: Current returns the last known good workflow after reload failure
- **WHEN** a caller asks for the current workflow after a failed reload and the store already has a known-good workflow
- **THEN** the store returns the cached workflow instead of replacing it with an error state

#### Scenario: Failed path switch keeps retrying the requested path
- **WHEN** a caller switches the workflow path to a missing or invalid file after a known-good workflow has already been loaded
- **THEN** the store keeps serving the previous workflow while continuing to retry the newly requested path on future reload attempts

#### Scenario: Startup fails without a known-good workflow
- **WHEN** the store starts and the initial workflow path is missing or invalid
- **THEN** store initialization fails instead of starting without a valid workflow
