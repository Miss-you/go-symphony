## Why

`go-symphony` already has typed config and a provider-neutral runtime domain, but it still lacks the single runtime owner that turns those contracts into real orchestration behavior. `T06` needs to land that scheduler core now so later workspace, runner, tracker, Codex, and observability tasks can integrate against one source of scheduling truth instead of continuing to depend on placeholders.

## What Changes

- Add the first real `internal/orchestrator` implementation as the sole owner of mutable scheduling state.
- Implement polling cadence, refresh coalescing, deterministic candidate ordering, dispatch gating, retry scheduling, running-item reconcile, stall recovery, and snapshot projection in provider-neutral form.
- Keep worker-to-orchestrator mutation input limited to `domain.RunEvent` and snapshot output limited to `domain.Snapshot`.
- Use package-private collaborator seams for candidate refresh, run start/stop, host admission hints, and timer control so `T06` does not prematurely freeze `T07` to `T10` interfaces.
- Defer workspace hook behavior, host-selection policy, tracker interface freeze, and Codex protocol parsing to their dedicated follow-up tasks.

## Capabilities

### New Capabilities

- `runtime-orchestrator`: Defines the provider-neutral scheduler core for polling, dispatch, retry, reconcile, stall recovery, and snapshot generation.

### Modified Capabilities

- `runtime-domain-model`: Clarify how the orchestrator consumes the existing `Snapshot`, `RetryEntry`, `ActiveRun`, `RunEvent`, and aggregate totals/rate-limit fields without widening the exported domain surface.

## Impact

- Affected code: `internal/orchestrator`
- Closely related existing code: `internal/config`, `internal/domain`
- Downstream dependents unlocked: `internal/workspace`, `internal/runner`, `internal/tracker`, `internal/codex`, `internal/observability`
- No user-facing API break is intended in this change; the effect is to replace an empty package with the approved scheduler core behavior
