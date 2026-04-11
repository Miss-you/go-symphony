# T12 Final Implementation v1 Review

Score: 64/100

Accepted: No

## High Severity Issues

1. The plan's `contentItems` change is internally incomplete and cannot produce the claimed protocol shape as written. `workspace/T12/final_impl_v1.md` proposes adding `ToolContentItem` and `ToolResult.ContentItems`, and it explicitly says `linear_graphql` output should be returned as top-level `contentItems` (`workspace/T12/final_impl_v1.md:138-149`). But the actual Codex session path still serializes tool results as `{"id": ..., "result": result}` in `internal/codex/session.go:529-542`. Without an explicit change to that response framing, the new field will still be nested under `result`, so the plan does not actually achieve the stated Symphony parity target.

2. The plan under-specifies T12 relative to the approved design task. The design task says T12 should implement `linear_graphql` and other Linear-specific write capabilities in the compatibility shell (`docs/plans/2026-04-10-go-symphony-design.md:12-22`, `docs/plans/2026-04-10-go-symphony-design-task.md:45`). `workspace/T12/final_impl_v1.md` narrows the slice to only one tool and defers every other Linear write helper to a later, unspecified phase (`workspace/T12/final_impl_v1.md:78-99, 260`). That may be a reasonable future split, but it is not a faithful statement of the approved T12 scope unless the task board is updated to make the narrower scope explicit.

## Medium / Low Issues

1. The verification gate is too narrow for the changes the plan itself proposes. The plan still lists only `go test ./internal/toolbridge/...` as the primary gate (`workspace/T12/final_impl_v1.md:220-223`), but its own Codex boundary adjustment requires protocol changes in `internal/codex` for raw string arguments and result framing (`workspace/T12/final_impl_v1.md:132-151`). The package gate should include `./internal/codex/...` or the plan should stop claiming the Codex protocol boundary is part of the work.

2. The test plan puts too much confidence in struct-level serialization checks. `TestCodexToolResultSerializesContentItemsAtTopLevel` is framed as a serialization guard (`workspace/T12/final_impl_v1.md:194-196`), but the actual compatibility risk is the `item/tool/call` response payload emitted by `Session.handleToolCall`. The test should pin the end-to-end written protocol shape, not only the Go struct tag shape.

3. The core boundary rule itself is good, but the narrative around it is slightly overstated. The plan says `internal/codex` should only gain enough shape to carry tool result and arguments that the app-server already supports (`workspace/T12/final_impl_v1.md:151`), yet the required raw-string argument parity means the core protocol layer must also preserve a non-object argument variant. That is still generic protocol work, but it is more than a small shape tweak.

## Rubric Breakdown

- Symphony alignment and source fidelity: 18/30
- Go-native simplicity and maintainability: 15/20
- No overdesign / clean boundaries: 11/20
- Implementation clarity and testability: 10/15
- Verification coverage and landing safety: 10/15

## Summary

The plan is directionally correct on the main architectural seam: keep `tracker` writes out of core and put the bridge in the compatibility shell. The two blockers are the incomplete `contentItems` protocol story and the mismatch between the stated T12 scope and the approved task wording. As written, I would not accept this plan for implementation yet.
