# T08 Spec Review

## Findings

No high-severity spec issues remain.

The accepted `final_impl.md` and the OpenSpec change now agree on the key boundary:

- `internal/runner` owns local/SSH command execution and pure host selection
- `internal/workspace` keeps lifecycle policy and delegates command execution
- `internal/orchestrator` remains the owner of mutable runtime state and feeds host-load data into runner admission

`test_strategy.md` also does the right thing here. It does not just list commands mechanically; it explains what each gate proves and keeps the T08 task-board gate, package regression coverage, and repo-wide sweep in view.

## Residual Risk

The remaining risk is implementation discipline, not spec shape: the workspace refactor has to stay focused on delegating execution to runner without recreating transport logic under a new helper name. The current design and test strategy are sufficient to manage that risk.

## Verdict

Pass for implementation start.
