# T06 Code Review

Review date: 2026-04-11 01:00 CST

## Findings

1. Blocking, fixed: retry entries were projected into `state.retrying` but never scheduled for delivery by `service`, so continuation and failure retries could never redispatch.
2. Important, fixed: candidate dispatch called host admission twice per attempt, which could double-consume capacity or diverge from the source host-selection semantics when the admission hook has side effects.

## Resolution

- Added retry timer ownership to `internal/orchestrator/service` so retry entries are scheduled, stale timer deliveries are ignored through the existing nonce guard, and due retries resync timers after redispatch or reschedule.
- Added a package-private `service.applyRunEvent` path so run-completed and run-failed events can drive retry scheduling through the real service owner rather than reducer-only tests.
- Removed the pre-dispatch host admission from candidate processing so each dispatch attempt admits capacity exactly once.
- Added tests covering real retry timer redispatch and single-admission dispatch behavior.

## Exit Decision

No unfixed bug/regression-level findings remain after the fixes and fresh verification pass.
