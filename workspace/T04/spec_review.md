# T04 Spec Review

## Verdict

Needs a small revision.

## High Severity Issues

None.

## Scope Mismatches

1. `workspace/T04/final_impl.md` freezes `ParseSettings`, `LoadSettings`, and `CurrentSettings` as part of the typed API surface, but the OpenSpec change only codifies typed settings normalization and atomic raw-plus-typed snapshots. That leaves the API contract itself under-specified and makes it easier for a later implementation to drift away from the intended `internal/config` entry points. References: [workspace/T04/final_impl.md](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/workspace/T04/final_impl.md#L76), [openspec/changes/internal-config-model/specs/runtime-config/spec.md](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/openspec/changes/internal-config-model/specs/runtime-config/spec.md#L3).

2. `workspace/T04/test_strategy.md` proves the typed defaults, env/path handling, validation, and atomic reload behavior, but it does not explicitly verify the new typed entry points. Since the design now treats `LoadSettings` and `CurrentSettings` as the supported typed accessors, the verification matrix should name them directly instead of relying on indirect coverage. References: [workspace/T04/final_impl.md](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/workspace/T04/final_impl.md#L76), [workspace/T04/test_strategy.md](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/workspace/T04/test_strategy.md#L17).

## Notes

- The rest of the change is internally consistent: provider-neutral `Settings`, compatibility-only `tracker.*` parsing, explicit `linear`/`memory` provider kinds, and atomic raw-plus-typed snapshot reloads all line up.
- The verification plan already covers the important compatibility behaviors: concrete defaults, env/path resolution, startup fail-fast, and last-known-good reload fallback.
