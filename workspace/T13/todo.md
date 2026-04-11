# T13 Residual Notes

## Accepted Residuals

- No runtime e2e was run for T13. The task only adds workflow bundle selection and Codex injection shape; `workspace/T13/test_strategy.md` records why package tests, compile checks, repo build/lint, and OpenSpec validation are the right evidence.
- The Codex injection proof is compile-time/package-level. It verifies that `Bundle.DynamicTools` and `Bundle.ToolHandler` fit `codex.SessionOptions`, but it does not start a real Codex app-server session.
- The dependency guard is a package-level negative-list check using `go list -deps`. It protects the forbidden V1 imports for `internal/workflow`; a broader repository import-policy tool is deferred until there is evidence it is needed.

## Blocking Items

None.
