# T06 Review 2

Scores: 26/12/18/12/10
Total: 78/100

## High Severity Issues

1. Failure-retry claim semantics were undefined. Continuation explicitly kept the item claimed until retry revalidation, but failure retry did not say whether the claim was retained or dropped. That ambiguity could cause duplicate dispatch or stuck items.

2. Retry attempt and backoff progression were not frozen precisely enough. The earlier draft stored attempts and required exponential backoff, but it did not define exactly when attempts incremented, whether continuation and failure shared a lineage, or how retry reschedule failures advanced the counter.

## Medium / Low Suggestions

1. Freeze exact stable sort keys for `Running` and `Retrying` instead of saying only “deterministically”.

2. State whether aggregate `CodexTotals` are lifetime cumulative or some other projection, and whether `RateLimits` means “latest observed” rather than merged state.

3. Turn the acceptance direction into explicit must-pass verification checks tied to the TDD list.
