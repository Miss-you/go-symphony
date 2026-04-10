# T03 Final Implementation v1

## Goal

Land the minimal `internal/config` workflow loader needed for parity with Symphony's `WORKFLOW.md` contract:

- resolve the active workflow path
- load and parse `WORKFLOW.md`
- retain the raw front matter map plus trimmed prompt body
- support hot reload with last-known-good retention
- expose a narrow API that later tasks can build on

`T03` should finish with a reliable raw loader and cache layer, not a full typed runtime config system.

## Non-Goals

- no provider-neutral internal config model yet; that belongs to `T04`
- no tracker-specific validation or normalization beyond preserving the raw front matter map
- no prompt rendering or issue interpolation yet; this task only loads the prompt template body
- no `fsnotify`/platform-specific file watcher dependency; polling is enough for V1 parity
- no orchestrator wiring, CLI startup wiring, or workflow-bundle selection in this task

## Required Compatibility Behaviors

### Path resolution

- Support an explicit runtime workflow-path override.
- Otherwise default to `<cwd>/WORKFLOW.md`.
- Preserve this precedence consistently for direct loads and the hot-reload store.

### Parse contract

- If the file starts with `---`, treat lines until the next `---` as YAML front matter.
- If there is no closing `---`, treat the remainder of the file as front matter and leave the prompt empty.
- If there is no opening `---`, treat the entire file as prompt body and use an empty config map.
- Front matter must decode to a map; non-map YAML is a typed error.
- Trim the prompt body before storing it in the loaded workflow object.
- Preserve a blank prompt as blank in the raw loaded workflow object.
- Preserve Symphony's current blank-prompt fallback behavior through a narrow config-level helper that later prompt-building code can call without changing the raw loader/store contract.

### Error surface

- Keep typed errors for:
  - missing workflow file
  - workflow parse error
  - workflow front matter not a map
- Distinguish load/parse failures from any future template-render failures.

### Hot reload and last-known-good

- A store layer in `internal/config` loads the workflow once at startup.
- The store polls every 1 second, matching Symphony's current coarse watch strategy.
- File change detection should use a stable stamp derived from file metadata plus content hash, not mtime alone.
- On a valid reload, replace the cached workflow.
- On an invalid reload or missing-path reload, keep the previous workflow cached and emit an operator-visible error.
- If the configured workflow path changes, attempt an immediate reload against the new path.
- When no store is running, direct load helpers should still work against the resolved path.

## Proposed Go Design

### `internal/config/workflow.go`

Define the raw loaded workflow shape:

- `type Workflow struct`
  - `Path string`
  - `Config map[string]any`
  - `Prompt string`
  - `PromptTemplate string`

Keep both `Prompt` and `PromptTemplate` for source compatibility with Symphony even if they are identical in `T03`.

Also add narrow fallback helpers for later callers:

- expose one narrow compatibility helper for blank-prompt fallback semantics

This helper preserves the current Symphony compatibility behavior for blank prompt bodies while keeping the raw loaded workflow unchanged.
Its exact exported shape is not part of the `T03` contract; it may stay package-local until a later task needs to freeze a broader prompt API.

### `internal/config/errors.go`

Define typed load errors with stable codes:

- `ErrMissingWorkflowFile`
- `ErrWorkflowParse`
- `ErrWorkflowFrontMatterNotMap`

Use a small error wrapper that can carry the path and underlying error when present.

### `internal/config/path.go`

Own path-resolution helpers:

- default path = `filepath.Join(os.Getwd(), "WORKFLOW.md")`
- explicit override support
- preserve the path string selected by precedence rules instead of adding new normalization semantics

Do not add environment-variable expansion here; that belongs to later config normalization, not workflow-file discovery.

### `internal/config/loader.go`

Implement pure load/parse helpers:

- `Load(path string) (Workflow, error)`
- `Parse(content []byte, path string) (Workflow, error)`

Parsing should:

- split front matter and prompt body
- decode YAML with `gopkg.in/yaml.v3`
- require a root map/object
- trim prompt body

Keep this layer side-effect free aside from file reads so tests stay simple.

### `internal/config/store.go`

Implement the runtime cache / reload loop:

- `type Store struct`
  - mutex-protected current workflow state
  - desired resolved path
  - last successfully loaded path
  - current stamp
  - poll interval (default 1s)
  - ticker / stop channel
  - injectable file read / stat / hash dependencies for deterministic tests
  - injectable tick source or manual poll trigger for deterministic tests
  - logger callback or `log.Logger`

Core methods:

- `NewStore(opts ...) (*Store, error)` loads the initial workflow and fails startup on missing/invalid workflow
- `Current() (Workflow, error)` returns the cached workflow; before returning, it should defensively reload if the path changed or the file stamp changed. If reload fails after at least one known-good load, log and return the cached workflow; return an error only when no known-good workflow exists yet
- `ForceReload() error` attempts reload and keeps the last known good workflow on failure
- `SetWorkflowPath(path string) error` updates the explicit path target and immediately attempts reload; on failure the old workflow remains active but future polls/current calls keep retrying the new path
- `ClearWorkflowPath() error` reverts to default path and immediately attempts reload under the same last-known-good rules
- `Close() error` stops polling

This keeps hot reload inside `internal/config`, where the approved design places it, without leaking mutable runtime state into `orchestrator`.
The extra seams are test-only support, not a new runtime abstraction boundary.
`Current()` reload-on-read and path mutation remain `internal/config` conveniences, not a new cross-package config-management API.

## Public/Internal API Sketch

The narrow surface for later tasks should look like this:

```go
type Workflow struct {
	Path           string
	Config         map[string]any
	Prompt         string
	PromptTemplate string
}

func Load(path string) (Workflow, error)

type Store struct { /* unexported fields */ }

func NewStore(opts ...StoreOption) (*Store, error)
func (s *Store) Current() (Workflow, error)
func (s *Store) ForceReload() error
func (s *Store) SetWorkflowPath(path string) error
func (s *Store) ClearWorkflowPath() error
func (s *Store) Close() error
```

This is intentionally smaller than the later typed-config API so `T04` can layer normalization on top instead of rewriting the loader.
The blank-prompt compatibility helper is behaviorally required but its exact exported API is intentionally not frozen here.

## Reload Semantics

1. Store startup loads the resolved workflow path and records a file stamp.
2. The store tracks both the desired workflow path and the last successfully loaded workflow.
3. Each 1-second poll checks whether the desired path changed or the current file stamp differs.
4. Unchanged desired path plus unchanged stamp means no-op.
5. Changed path or changed stamp triggers a full reload through the same loader path as startup.
6. Successful reload swaps in the new workflow, loaded path, and stamp.
7. Failed reload logs the error and preserves the previous workflow/stamp, but does not forget the desired path, so future retries can pick up a later fix.
8. Explicit `ForceReload`, `SetWorkflowPath`, and `ClearWorkflowPath` all use the same code path, so direct and poll-driven reload behavior cannot drift.

## Testing Shape

`T03` should be driven by unit tests in `internal/config/...` that prove:

- default path resolves to `<cwd>/WORKFLOW.md`
- explicit path override wins
- prompt-only files load as empty config + prompt body
- unterminated front matter yields config + empty prompt
- non-map front matter returns the dedicated error
- missing file returns the dedicated error with path context
- prompt body is trimmed
- blank prompt fallback helper returns the built-in default prompt template
- store startup fails on missing or invalid workflow
- valid reload replaces the cached workflow
- invalid reload keeps the last known good workflow
- path-change reload works
- reload failure preserves the last successfully loaded workflow object while future retries continue against the requested path
- store polling / defensive `Current()` reload path does not require restart
- reload behavior is testable without real sleeps by using injected tick / filesystem seams

## Open Questions / Deferred Work

- Template syntax validation and strict rendering should stay deferred until the prompt-building task; `T03` only loads the raw prompt template string.
- Front-matter normalization into provider-neutral typed structs belongs to `T04`, not here.
- `fsnotify` can be reconsidered later if polling becomes a demonstrated problem, but it should not complicate `T03`.
- Wiring the store into CLI/app startup is a downstream task once the runtime skeleton exists.
