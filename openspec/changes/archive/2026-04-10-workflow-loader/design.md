## Context

The approved design assigns `WORKFLOW.md` parsing, prompt/template loading, hot reload, and last-known-good retention to `internal/config`. The current repository still contains only placeholder packages, so `T03` must establish the first concrete workflow-loading behavior without collapsing later tasks such as typed config normalization (`T04`) or prompt rendering/workflow selection.

The Elixir implementation resolves the workflow path from an explicit runtime override or `<cwd>/WORKFLOW.md`, parses optional YAML front matter, trims the prompt body, polls every second for file changes, and preserves the last known good workflow when reload fails. The Go implementation must match those user-visible semantics while staying Go-native and narrow.

## Goals / Non-Goals

**Goals:**

- preserve Symphony-compatible `WORKFLOW.md` path resolution, parse rules, and reload behavior
- preserve Symphony-compatible blank-prompt fallback semantics without requiring prompt rendering in `T03`
- keep the first Go API limited to raw workflow loading plus reloadable caching
- make reload behavior deterministic and unit-testable
- leave room for `T04` to add typed normalization on top of the loader instead of replacing it

**Non-Goals:**

- provider-neutral typed config normalization
- tracker validation or environment-backed config coercion
- prompt rendering or template interpolation
- CLI/runtime wiring outside `internal/config`
- `fsnotify` or platform-specific watch mechanisms

## Decisions

### 1. Keep `T03` at the raw loader/store layer

`T03` will return a raw workflow shape containing the file path, front matter map, and trimmed prompt string. This matches the immediate task goal while keeping `T04` responsible for typed provider-neutral normalization.

Alternative considered:

- Parse straight into typed config structs now. Rejected because it couples `T03` to `T04` and makes the first loader harder to validate against raw Symphony behavior.

### 2. Implement reload via an internal polling store

The Go store will poll every second and compare a stable file stamp derived from file metadata plus content hash. This matches the current Symphony behavior closely enough without introducing OS-specific watcher complexity before the runtime is fully wired.

Alternative considered:

- Add `fsnotify` immediately. Rejected because it is extra machinery for little benefit at this stage and is not required for parity with the current coarse 1-second polling behavior.

### 3. Preserve last-known-good semantics inside the store

Initial load failures will fail startup because no good workflow exists yet. Later reload failures will log and retain the previously cached workflow. `Current()` will return the cached workflow after a failed reload if at least one known-good workflow has been loaded.

Alternative considered:

- Bubble every reload failure to callers and force them to handle cache fallback. Rejected because it spreads reload-state ownership outside `internal/config` and diverges from the approved design and existing Symphony behavior.

### 4. Keep loader errors typed and narrow

The loader will distinguish missing workflow files, YAML parse failures, and non-map front matter. Template syntax/rendering errors remain deferred to later prompt-building work so the loader surface stays focused.

Alternative considered:

- Introduce a single opaque load error now. Rejected because it weakens parity and makes later validation/tests less precise.

### 5. Keep blank-prompt fallback as a compatibility helper, not loader mutation

The raw loaded workflow object will preserve a blank prompt as blank. `internal/config` will also expose a narrow helper that returns Symphony's built-in default prompt template when the loaded prompt is blank. This keeps the raw loader/store faithful to the file while preserving the current compatibility behavior for downstream prompt-building code.

Alternative considered:

- Defer blank-prompt fallback entirely to a later task. Rejected because the source implementation treats default-prompt fallback as part of the workflow/config compatibility surface.

## Risks / Trade-offs

- Polling every second is simpler than event-driven watching, but reload visibility is not instantaneous. Mitigation: stay aligned with current Symphony behavior and keep the poll interval configurable only for tests.
- Keeping both `Prompt` and `PromptTemplate` duplicates data. Mitigation: preserve source compatibility now and revisit only if later tasks show the duplication is harmful.
- The store surface is slightly broader than a pure file loader because it owns polling, reload-on-read, and path switching. Mitigation: keep those behaviors package-local to `internal/config` and make their seams explicit for tests rather than introducing a cross-package config manager.
- Deferring template parsing means `T03` alone will not catch invalid prompt syntax. Mitigation: keep that behavior explicitly scoped to the later prompt-rendering task rather than mixing concerns here.
