# T09 Code Review

Reviewer found four blocking issues in the first implementation pass:

1. `StartProcessTransport` merged stderr into the protocol stdout stream.
2. Object-shaped `codex.approval_policy` values were flattened into strings.
3. The real transport reader used `bufio.Scanner`, which has a 64 KiB token limit and can race EOF against buffered lines.
4. Unsupported tool failures returned `unsupported tool` instead of the protocol sentinel `unsupported_tool_call`.

Fixes applied:

- stderr is drained separately and is not parsed as protocol JSON;
- `Config.ApprovalPolicy` now preserves `any` and only string `never` triggers auto-approval;
- process transport now uses ordered `bufio.Reader.ReadBytes('\n')` framing through one result channel;
- unsupported tool responses now return `{"success": false, "error": "unsupported_tool_call"}`;
- regression tests cover policy map pass-through, stderr separation, long protocol lines, and exact unsupported-tool payloads.

Follow-up status:

- `go test -count=1 ./internal/codex/...` passed after fixes.
- Full post-fix verification passed: `go test -count=1 ./internal/codex/...`, `go test -count=1 ./...`, `make build`, `make lint`, `make test-e2e`, and `openspec validate codex-app-server-protocol`.
- Follow-up code review reported no blocking issues remain.

PR AI review follow-up:

- `PRRT_kwDOR-rnWM56R2iF`: `NonInteractive` now gates `item/tool/requestUserInput` fallback responses; interactive mode returns `ErrApprovalRequired` instead of silently answering.
- `PRRT_kwDOR-rnWM56R2iG`: request/response waits now return the parent context error when the caller's context expires, instead of wrapping it as `ErrReadTimeout`.
- `PRRT_kwDOR-rnWM56R2iI`: streamed turn waits now return the parent context error when the caller's context expires, instead of wrapping it as `ErrTurnTimeout`.
- Regression tests cover disabled non-interactive user input and parent context deadlines for both read and turn receive paths.
