# T19 Review 2

Score: 68/100

Rubric breakdown:

- Symphony alignment: 21/30
- Go-native maintainability: 13/20
- No overdesign / boundary fit: 10/20
- Implementation clarity / testability: 11/15
- Verification coverage / safety: 13/15

## High-severity issues

1. `--only-issue` is placed on the production `cmd/symphony` CLI instead of a dedicated smoke / verification entrypoint.
   - Evidence: `workspace/T19/final_impl_v1.md:54-73`, and the current main CLI is a single all-in-one runtime launcher with only workflow/logs/port flags at `internal/cli/main.go:51-108` and `internal/cli/main.go:132-213`.
   - Why this is a problem: this turns a verification-only behavior into a user-facing runtime mode for the main binary. It also means the filter affects `ListCandidates`, `ListByStates`, and `RefreshByIDs` inside `StartRuntime`, so the runtime semantics diverge from the normal production path. That is the wrong boundary for a helper whose purpose is to let an operator validate a live flow safely.
   - Concrete fix: keep `cmd/symphony` unchanged and add a separate smoke/verification command or wrapper. If you need issue scoping, inject a filtered `tracker.TrackerReader` into `StartRuntime` from that wrapper, rather than teaching the main CLI a special-case flag.

2. The Linear probe is not specified in a way that can be tested offline without hitting the network.
   - Evidence: `workspace/T19/final_impl_v1.md:30-52` says the probe should construct `linear.NewReader(settings.Provider, nil)` and then call the live read methods. The testing section only promises to "reject non-Linear settings without network calls" at `workspace/T19/final_impl_v1.md:97-100`.
   - Why this is a problem: the command itself is supposed to be a verification helper, but the draft does not introduce a reader/client injection seam for the successful path. Without that, unit tests can only cover parsing and failure cases, not the actual report-generation path. That weakens the whole point of adding the probe.
   - Concrete fix: structure the probe around a `runProbe(ctx, deps, args)` function or a small probe service that accepts a `tracker.TrackerReader` or `linear.Client` factory. Then test the live logic with `memory.NewReader` or a fake Linear client so the command can be exercised fully offline.

## Additional notes

- The terminal-cleanup filtering inside `--only-issue` is the riskiest part of the proposal. Scoping `ListByStates` makes the smoke run safer in the sense that it will not delete unrelated local workspaces, but it also means the smoke no longer verifies the real startup-cleanup behavior. If the goal is a controlled smoke, that needs to be called out explicitly and kept off the main CLI boundary.
- The probe workflow is the right direction, but the draft should state clearly that "no network" applies to the automated tests, not to the operator workflow itself. Otherwise the design overpromises what the helper can prove.

## Recommendation

Revise the design so that:

1. The main `symphony` binary stays production-shaped.
2. Verification helpers live in a separate command or wrapper.
3. The Linear probe has an injected reader/client seam and is covered by offline tests.

