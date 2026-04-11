## 1. TDD Coverage

- [x] 1.1 Add package tests that prove `Select` returns `compat_linear_default` for Linear settings and fails explicitly for unsupported provider kinds.
- [x] 1.2 Add package tests that prove the selected bundle uses `config.EffectivePromptTemplate` and preserves the blank-prompt fallback.
- [x] 1.3 Add package tests that prove the bundle exposes exactly the existing Linear ToolBridge `linear_graphql` tool spec and handler, with no extra tools.
- [x] 1.4 Add a dependency-guard style test or equivalent check that keeps `internal/workflow` out of `internal/orchestrator`, `internal/tracker`, `internal/workspace`, `internal/runner`, and `internal/domain`.

## 2. Implementation

- [x] 2.1 Implement the minimal `internal/workflow` package surface for `Select` and the `compat_linear_default` bundle factory.
- [x] 2.2 Wire the selected bundle to the existing Linear ToolBridge by reusing `ToolSpecs()` and the bridge handler directly.
- [x] 2.3 Keep workflow selection free of file loading, reload ownership, and provider-neutral core boundary widening.

## 3. Verification

- [x] 3.1 Run `go test ./internal/workflow/...` and fix any package-level failures.
- [x] 3.2 Run the broader compile and test gates needed by the change, at minimum `go test ./...`.
- [x] 3.3 Run repository build/lint verification relevant to the workflow bundle change, at minimum `make build` and `make lint`.
- [x] 3.4 Run `openspec validate --type change linear-workflow-bundle` and capture the result for handoff.
