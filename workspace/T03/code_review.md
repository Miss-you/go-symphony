# Findings

No findings. I did not find a clear contract mismatch, regression, or missing required behavior in `internal/config/*.go` or `internal/config/*_test.go` relative to `workspace/T03/final_impl.md` and `openspec/changes/workflow-loader/specs/workflow-loader/spec.md`.

# Open Questions

- The store currently swallows reload errors in `Current()` after logging and returns the cached workflow. That matches the stated last-known-good behavior, but it means callers have no direct signal that the underlying file is broken. If the broader runtime expects explicit error propagation later, that contract will need to be defined outside T03.
- `EffectivePromptTemplate()` is exported even though the final implementation notes say the exact exported name is intentionally not frozen. If later tasks want a narrower API surface, this may need cleanup.

# Residual Risks

- There is no test that asserts the exact log message shape on reload failure, only that a log entry is emitted.
- The hot-reload tests rely on file overwrites and polling behavior, but they do not pin down every race around `Current()` vs the background ticker.
- I did not see coverage for `ClearWorkflowPath()`, though the implementation appears consistent with the path-switch retry model.
