# T19 Code Review

## Findings

1. **Medium** - The published verification template is not copy-paste runnable as written. In [`docs/verification-workflows.md`](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t19-verification-workflows/docs/verification-workflows.md#L87) the minimal workflow omits `hooks.timeout_ms`, but [`internal/config/settings.go`](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t19-verification-workflows/internal/config/settings.go#L346) requires that field to be positive. Copying the doc sample will fail settings validation before either `symphony-verify` command can start. The same snippet also hard-codes `/Users/lihui/Documents/GitHub/go-symphony`, which makes the example workstation-specific instead of portable.

2. **Low** - The boundary test for the Linear probe is too weak to prove the intended isolation. [`cmd/symphony-verify/main_test.go`](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t19-verification-workflows/cmd/symphony-verify/main_test.go#L141) only scans the text of `linear.go` for forbidden imports. That catches one file, but it does not protect the `linear` path at the package boundary: a future helper added in another file in the same package could still pull runtime/Codex/workspace dependencies into the probe path without failing this test. A stronger assertion would inspect the package import graph or move the probe into its own package.

## Validation

- `go test ./...` passes in the current worktree.
