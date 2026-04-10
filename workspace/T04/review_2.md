# T04 Review 2

Scores: 24/16/15/11/8
Total: 74/100

## High Severity Issues

1. The plan does not define fail-fast behavior for typed-config normalization/validation at startup. Lines 58-67 say validation belongs in `internal/config`, and lines 86-93 talk about invalid reload coverage, but there is no explicit contract that an initial typed-settings failure should reject startup the way Symphony does today for invalid config. That leaves room for an implementation that successfully loads `WORKFLOW.md` and only discovers the semantic error later, which would be a behavioral regression.
2. The raw cache / typed cache boundary is not atomic. Lines 79-82 say `Store` may cache derived typed settings, and line 82 says bad reloads should keep the previous good settings, but the doc never states what happens when raw parsing succeeds and typed normalization fails. Without an explicit atomic snapshot contract, the implementation can easily end up with raw workflow state from the new file and typed settings from the old file, or vice versa.

## Medium / Low Suggestions

- The settings API shape is still underspecified. Line 34 leaves the type name as `Settings or RuntimeConfig`, and line 47 introduces a neutral provider field without saying whether `tracker.kind` remains the canonical input/output field. Freeze one source of truth now or downstream packages will drift.
- The verification matrix should explicitly include the path/default edge cases from the source contract, not just generic env fallback coverage. In particular, `workspace.root` needs coverage for `~` expansion and empty-vs-missing env reference handling, which are observable in the Elixir reference behavior.
- Line 61 says to accept only the provider kinds supported by the current product scope, but it would be safer to name those values explicitly in the plan so T04 does not accidentally narrow support or leave room for interpretation.
