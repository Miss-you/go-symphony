# T11 Code Review

## Critical

None.

## Important

1. **Context cancellation is still wrapped as a generic request error during body read**
   - File: [internal/trackers/linear/reader.go:576-579](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t11-linear-reader-adapter/internal/trackers/linear/reader.go#L576)
   - What’s wrong: `HTTPClient.GraphQL` returns `RequestError` for `io.ReadAll(resp.Body)` failures unconditionally. If the caller cancels the context after the HTTP response is received but before the body is consumed, that cancellation is converted into `RequestError` instead of propagating as `context.Canceled` / deadline exceeded.
   - Why it matters: the task contract explicitly requires context cancellation to bubble through the client layer as a context-derived error. This code path breaks that guarantee and makes cancellation indistinguishable from transport failure.
   - How to fix: check `errors.Is(err, context.Canceled)` and `errors.Is(err, context.DeadlineExceeded)` for the body-read error path as well, and return the raw context error in those cases.

2. **Malformed JSON payloads are classified as request failures instead of payload failures**
   - File: [internal/trackers/linear/reader.go:584-589](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t11-linear-reader-adapter/internal/trackers/linear/reader.go#L584)
   - What’s wrong: if Linear returns HTTP 200 but the body is not valid JSON, `decoder.Decode` returns a `RequestError`. That collapses malformed payloads into the same bucket as network/request failures.
   - Why it matters: the OpenSpec contract for this adapter requires distinct handling for malformed payloads versus transport/request failures. Collapsing them reduces diagnosability and weakens the error taxonomy the rest of the runtime depends on.
   - How to fix: return a payload-classification error for decode failures, such as `ErrUnknownPayload` or a dedicated malformed-payload error type, instead of `RequestError`.

## Minor

1. **`RefreshByIDs` empty-input behavior is implemented but not explicitly locked by a direct test**
   - File: [internal/trackers/linear/reader_test.go:152-191](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t11-linear-reader-adapter/internal/trackers/linear/reader_test.go#L152)
   - What’s wrong: the package suite covers batching and ordering, but it does not assert that `RefreshByIDs([]string{})` returns `[]` without calling the client.
   - Why it matters: the task strategy explicitly calls for explicit empty-input coverage for both `ListByStates` and `RefreshByIDs`. Without a direct test, this edge case can regress silently.
   - How to fix: add a small empty-input test that asserts `RefreshByIDs` returns an empty slice and makes zero GraphQL calls.

## Recommendations

- Add one regression test for cancellation during response-body read, not just for transport cancellation.
- Add one regression test for invalid JSON payloads so the error taxonomy stays stable.
- Add the explicit empty-input `RefreshByIDs` test before closing T11 so the package gate matches the written strategy.

## Assessment

**Ready to merge?** No

**Reasoning:** The adapter is close and the broad gates are passing, but two error-path regressions remain in the HTTP client layer, and the test suite is missing one explicit contract edge that the task strategy called out. Those are small fixes, but they should be closed before merge.
