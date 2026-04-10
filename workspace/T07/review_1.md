# T07 Review 1

## Findings

- High severity: `after_run` is described as best-effort, but the plan never requires it to run on failure paths. The Elixir reference calls `Workspace.run_after_run_hook/3` from an `after` block, so the post-run hook still fires when the Codex run raises or exits early. As written, a T07 implementation can pass the listed tests while silently skipping `after_run` on failed runs, which breaks observed parity and can leak cleanup side effects. See [workspace/T07/final_impl_v1.md](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t07-workspace-lifecycle/workspace/T07/final_impl_v1.md#L63) and [workspace/T07/original_impl.md](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t07-workspace-lifecycle/workspace/T07/original_impl.md#L122).

- Medium severity: terminal cleanup is under-specified for the Elixir parity case that removes workspaces across configured SSH hosts on startup. The final plan keeps SSH policy outside T07, which is reasonable, but it never defines the handoff that preserves the multi-host sweep from `remove_issue_workspaces/2` and `run_terminal_workspace_cleanup/0`. Without that, a local-only implementation can look correct in package tests while leaving stale remote workspaces behind. See [workspace/T07/final_impl_v1.md](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t07-workspace-lifecycle/workspace/T07/final_impl_v1.md#L122) and [workspace/T07/original_impl.md](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t07-workspace-lifecycle/workspace/T07/original_impl.md#L141).

## Scores

- Symphony parity/source faithfulness: 20/30
- Go-native simplicity/maintainability: 17/20
- Anti-overdesign / boundary cleanliness: 16/20
- Implementation clarity/testability: 11/15
- Verification coverage/safety: 10/15

Total: 74/100

## Verdict

Not ready to accept as-is. The plan needs an explicit guarantee that `after_run` executes on failure paths, plus a clearer terminal-cleanup handoff for the host-aware parity case before it can safely move to spec.
