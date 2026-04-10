# T03 Final Implementation

## Review Gate

`final_impl_v1.md` required two review rounds.

Round 1 findings:

- high severity: blank-prompt contract was internally inconsistent
- high severity: the draft introduced absolute-path normalization that was not present in Symphony

Round 2 outcome after revision:

- `review_1_round2.md`: 99 / 100, no high-severity issues
- `review_2_round2.md`: 91 / 100, no high-severity issues
- average: 95 / 100

Acceptance decision:

- average score exceeds the `>= 80` threshold
- no reviewer reports a remaining high-severity issue
- low-risk suggestions were folded back into this final plan

## Final Scope

`T03` lands the raw `WORKFLOW.md` loader layer in `internal/config` and stops there.

It must provide:

- workflow path resolution with explicit override first, then `<cwd>/WORKFLOW.md`
- file loading and parsing for optional YAML front matter plus trimmed prompt body
- typed load errors for missing file, parse failure, and non-map front matter
- a hot-reload store with 1-second polling and last-known-good retention
- direct-load helpers that still work when no store is running
- a narrow compatibility helper for blank-prompt fallback semantics

It must not provide:

- the typed provider-neutral internal config model planned for `T04`
- prompt rendering or issue interpolation
- workflow-bundle selection
- orchestrator or CLI wiring
- `fsnotify` or other platform-specific watcher complexity

## Final Design

### Workflow shape

Define a raw loaded workflow type in `internal/config`:

```go
type Workflow struct {
	Path           string
	Config         map[string]any
	Prompt         string
	PromptTemplate string
}
```

Keep `Prompt` and `PromptTemplate` both present for source compatibility even if they are identical in `T03`.

Expose:

- `Load(path string) (Workflow, error)`
- one narrow compatibility helper for blank-prompt fallback semantics

That helper returns the built-in default prompt when the loaded prompt is blank. The raw loader still preserves the original blank body in `Prompt` / `PromptTemplate`. Its exact exported name is intentionally not part of the `T03` contract; it may remain package-local until a later task needs to freeze a broader prompt API.

### Parse contract

The Go loader must match Symphony's current behavior:

- front matter starts only when the file begins with `---`
- lines until the next `---` are YAML front matter
- missing closing `---` means front matter runs to EOF and prompt is empty
- no opening `---` means prompt-only file with empty config map
- decoded YAML root must be a map
- prompt body is trimmed before being stored
- path handling follows precedence rules only; the loader must not introduce new path normalization semantics beyond the selected workflow path string

### Error model

Use stable typed errors:

- `ErrMissingWorkflowFile`
- `ErrWorkflowParse`
- `ErrWorkflowFrontMatterNotMap`

These are load/parse errors only. Template parse/render errors remain outside `T03`.

### Store and reload model

Implement `internal/config/store.go` with a narrow cache:

- load once at startup and fail startup if the initial workflow is missing or invalid
- poll every 1 second
- compute a stable file stamp from file metadata plus content hash
- on successful reload, replace the cached workflow and stamp
- on failed reload, log and keep the last known good workflow

The store must track both:

- the desired resolved path
- the last successfully loaded workflow path

This preserves Symphony's behavior where a bad explicit path switch does not discard the current workflow, but future polls or `Current()` calls still retry the new path once it becomes valid.

Required methods:

- `NewStore(opts ...StoreOption) (*Store, error)`
- `Current() (Workflow, error)`
- `ForceReload() error`
- `SetWorkflowPath(path string) error`
- `ClearWorkflowPath() error`
- `Close() error`

`Current()` may attempt a defensive reload before returning. If reload fails after at least one known-good load, it should return the cached workflow and surface the failure through logging rather than replacing the cache with broken state.

To keep tests deterministic, the store should also accept internal seams for:

- file read / stat / hash behavior
- tick delivery or manual poll triggering
- logging capture

These seams are test support inside `internal/config`, not a new cross-package configuration abstraction.

## Test Focus

`go test ./internal/config/...` must prove:

- default path resolution
- explicit path override
- prompt-only file loading
- unterminated front matter behavior
- non-map front matter rejection
- missing file error shape
- prompt trimming
- blank prompt fallback helper
- store startup failure on invalid or missing workflow
- successful reload replacing cached state
- invalid reload retaining last-known-good state
- path-switch retry behavior after a failed reload
- reload failure preserving the last successfully loaded workflow object while future retries continue against the requested path
- reload behavior remains testable without real sleeps

## Deferred To Later Tasks

- typed internal config normalization and validation move to `T04`
- strict template parsing/rendering moves to the prompt-building task
- CLI startup wiring and runtime integration move to later orchestration/bootstrap tasks
- any richer file-watch mechanism must wait for evidence that 1-second polling is insufficient
