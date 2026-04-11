# T13 Test Strategy

## Purpose

This task adds a narrow `internal/workflow` selection layer and one concrete bundle, `compat_linear_default`. The tests need to prove four things, not just exercise code paths:

1. Linear settings select the only supported bundle.
2. Prompt fallback behavior stays aligned with `config.EffectivePromptTemplate`.
3. The bundle reuses the existing Linear ToolBridge surface without adding a second tool system.
4. The workflow package stays a compatibility-shell leaf and can flow directly into Codex session bootstrap.

The strategy below maps each functional goal to the smallest evidence that can prove it.

## Evidence Map

| Functional goal | What the test evidence proves | Required evidence |
| --- | --- | --- |
| Select the Linear workflow bundle | `Select(raw, settings)` returns `compat_linear_default` for `config.ProviderLinear` and fails explicitly for any other provider kind. | `internal/workflow` package test coverage for selection and unsupported-provider behavior. |
| Preserve prompt fallback semantics | The bundle uses `config.EffectivePromptTemplate`, so a blank workflow body still yields Symphony's default prompt template and a non-blank body is preserved unchanged. | `internal/workflow` package test coverage for bundle construction and prompt template resolution. |
| Reuse the existing Linear ToolBridge surface | The bundle exposes exactly the Linear bridge tool spec list and the bridge itself as the handler, with no extra dynamic tools and no local reimplementation of `linear_graphql`. | `internal/workflow` package test coverage that inspects `DynamicTools` shape and handler compatibility. |
| Keep workflow selection a thin compatibility layer | `internal/workflow` does not depend on orchestrator/tracker/workspace/runner/domain, and the bundle values can be passed to Codex bootstrap without adapter code. | Package-level dependency guard plus compile-time integration shape check against `codex.SessionOptions`. |

## Package Tests For `internal/workflow`

The package gate for this task is `go test ./internal/workflow/...`.

Required assertions:

1. `Select` returns a bundle whose `ID` is `compat_linear_default` when the loaded settings describe `config.ProviderLinear`.
2. `Select` returns an explicit unsupported-provider error for a non-Linear provider kind.
3. `CompatLinearDefault` or the selected bundle reports `PromptTemplate` from `config.EffectivePromptTemplate(raw)`.
4. A blank workflow prompt body still resolves to the default template from `internal/config`.
5. A populated workflow prompt body is preserved unchanged.
6. The bundle exposes exactly one dynamic tool, `linear_graphql`.
7. The bundle tool handler is the existing Linear bridge type or value, not a shim that changes `linear_graphql` behavior.

These tests prove the task-specific behavior directly at the package boundary, which is the right place to lock the contract before any Codex bootstrap wiring relies on it.

## Compile And Integration Shape Checks For Codex Injection

This task does not need a runtime e2e scenario. The question is structural: can the workflow bundle feed Codex session startup without translation code?

Use compile-time or package-level shape checks to prove:

1. A workflow bundle value can be assigned directly into `codex.SessionOptions.Config.DynamicTools`.
2. The bundle tool handler satisfies `codex.ToolHandler` and can be passed directly to `codex.SessionOptions.ToolHandler`.
3. The integration surface does not require an adapter layer, helper wrapper, or extra transformation package between `internal/workflow` and `internal/codex`.

The evidence should be a compile check in the affected packages, not a fake runtime test. The right signal is that the repository builds with the workflow bundle wired into the existing Codex startup path.

Recommended compile scope:

- `go test ./internal/workflow/...`
- if the session bootstrap edges move, expand to `go test ./internal/workflow/... ./internal/codex/... ./internal/toolbridge/...`

That combination proves the bundle shape, the tool-handler interface, and the Codex injection path all line up.

## Build, Lint, And Repo Gates

The broader repository gates are still required because this task changes a shared startup seam:

1. `go test ./...`
   - Proves the new workflow package does not break compilation or behavior elsewhere in the tree.
   - This is the broad repo-level sanity gate after the package tests are green.
2. `make build`
   - Proves the repository compiles as a whole through the normal build target.
   - Useful as a second compile signal after package tests.
3. `make lint`
   - Proves the change does not introduce static-analysis or style regressions.
   - Important here because the workflow package is intentionally small and boundary-focused; lint failures would usually mean it drifted into a broader abstraction or violated local conventions.

These gates are meaningful for T13 because the task touches a startup boundary that other packages consume indirectly.

## E2E Applicability

E2E is not applicable for this task.

Reason:

- T13 does not change orchestrator behavior, tracker writes, workspace semantics, or runner execution.
- The task is about workflow selection and Codex bootstrap shape, which are best proven by package tests and compile-time wiring checks.
- A full e2e run would not add much signal beyond what the package tests and repo build already prove for this change.

So `go test ./...` plus `make build` and `make lint` are sufficient repository-level gates here; `make test-e2e` is not required for acceptance.

## OpenSpec Validation Gate

The OpenSpec change must validate cleanly before this task can be closed.

Required gate:

- `openspec validate --type change linear-workflow-bundle`

What it proves:

- the change artifacts are complete
- the task scope and spec scope still match
- the written spec remains internally consistent after the test strategy is added

This gate is not a substitute for code tests. It is the artifact-level proof that the change package is structurally complete and aligned with the task board.

## Verification Order

Run verification in this order:

1. `go test ./internal/workflow/...`
2. compile-shape checks for Codex injection, expanding to `internal/codex` and `internal/toolbridge` if needed
3. `go test ./...`
4. `make build`
5. `make lint`
6. `openspec validate --type change linear-workflow-bundle`

The package tests prove the workflow contract. The compile-shape checks prove the bundle can flow into Codex. The repo gates prove the change does not disturb the rest of the tree. The OpenSpec validation proves the artifact set is complete.
