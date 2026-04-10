# T03 Spec Review

## Verdict

Needs revision. The T03 artifacts are mostly aligned on the raw `WORKFLOW.md` loader/store boundary, but there are two closure risks: the review/test plan asks for broader verification than the task board records, and the final implementation language is a bit broader than the OpenSpec contract in a way that could leak later-task concerns into T03.

## High Severity Issues

- The task board gate for T03 is only `go test ./internal/config/...`, but `workspace/T03/test_strategy.md` adds repo-wide `go test ./...`, `make build`, `make lint`, and `make test-e2e` as acceptance-relevant checks. That verification burden is not reflected in the T03 row, so closure would be based on incomplete recorded gates.
- `workspace/T03/final_impl.md` adds `LoadDefault()`, `DefaultPromptTemplate()`, and `EffectivePromptTemplate()` as required API surface. The OpenSpec change only requires a narrow blank-prompt compatibility helper; it does not require a broader default-prompt API. This is an API-scope expansion that should be justified or trimmed before T03 is treated as closed.

## Scope Mismatches

- `workspace/T03/test_strategy.md` treats `make test-e2e` as part of the verification matrix, but the OpenSpec change explicitly keeps T03 out of full runtime wiring and the task board only names the package-local gate. If e2e is non-applicable, that exception needs to be recorded in the task artifacts and board.
- The final implementation text says the store should support `Current()`, `ForceReload()`, `SetWorkflowPath()`, `ClearWorkflowPath()`, and logging seams. Those are consistent with the design, but they push the store surface close to a general config manager. Keep the implementation narrowly framed as a workflow loader/store, not a broader config abstraction.

## Notes

- The core semantics do line up on the important parts: explicit-path precedence, `<cwd>/WORKFLOW.md` defaulting, YAML front matter parsing, typed load errors, blank-prompt preservation, and last-known-good reload behavior.
- The main cleanup needed is recording the full verification expectation consistently and trimming any API wording that implies more than the approved T03 loader/store contract.
