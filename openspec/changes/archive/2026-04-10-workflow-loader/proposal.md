## Why

`T03` is the first real runtime behavior after the repo skeleton. Without a faithful `WORKFLOW.md` loader, the Go port cannot preserve Symphony's repository-owned workflow contract or support later tasks that depend on raw workflow/config input and last-known-good reload behavior.

## What Changes

- add a raw `WORKFLOW.md` loader in `internal/config` that resolves the active workflow path and parses optional YAML front matter plus trimmed prompt body
- add typed loader errors for missing files, parse failures, and non-map front matter
- add a narrow blank-prompt compatibility helper without pulling prompt rendering into `T03`
- add a reloadable store in `internal/config` that polls for workflow changes and preserves the last known good workflow on invalid reloads
- add focused unit tests for path resolution, parse edge cases, and reload fallback semantics
- defer typed config normalization and prompt rendering to later tasks

## Capabilities

### New Capabilities

- `workflow-loader`: repository-owned `WORKFLOW.md` discovery, parsing, reload, and last-known-good retention for the Go runtime

### Modified Capabilities

- None

## Impact

- `internal/config/`
- `go.mod` and Go dependencies for YAML parsing if needed
- `workspace/T03/`
- `docs/plans/2026-04-10-go-symphony-design-task.md`
