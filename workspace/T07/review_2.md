# T07 Review 2

## High Severity Findings

- `workspace/T07/final_impl_v1.md:110-137` narrows T07 to a local-filesystem lifecycle and treats SSH as something for later tasks, but the Elixir `workspace.ex` behavior that this task is meant to match includes remote workspace creation/removal and terminal cleanup fan-out across configured SSH hosts. As written, a Go implementation could pass `go test ./internal/workspace/...` while leaving remote workspaces orphaned and startup terminal cleanup incomplete. If SSH parity is intentionally deferred, that exception needs to be stated explicitly and the task board / acceptance criteria need to change with it.
- `workspace/T07/final_impl_v1.md:71-83` is not explicit enough about canonicalizing and validating the workspace root itself. The Elixir implementation canonicalizes the root, rejects root collisions, and detects symlink escapes relative to the configured root; this plan only says the config layer resolves env vars and `~`, then workspace should "only validate the derived path". That is too weak for a safety boundary, because a symlinked root can produce incorrect acceptance/rejection behavior or allow path escape bugs that package tests would not catch unless the root-canonicalization case is pinned down.

## Scores

- Symphony parity/source faithfulness: 17/30
- Go-native simplicity/maintainability: 16/20
- Anti-overdesign / boundary cleanliness: 12/20
- Implementation clarity / testability: 12/15
- Verification coverage / safety: 8/15
- Total: 65/100

## Verdict

Needs revision before implementation/spec generation. The plan is directionally good, but it still leaves two parity-critical behaviors underspecified: remote workspace cleanup semantics and root safety handling. Those should be made explicit before T07 is treated as an execution-ready workspace contract.
