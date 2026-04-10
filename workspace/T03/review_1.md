# T03 Review

Scores: 24/20/15/12/10
Total: 81/100

## High Severity Issues

1. The blank-prompt contract is internally inconsistent and does not match the source behavior. Line 38 says the loader should keep a blank prompt blank and defer fallback to a later prompt builder, but line 47 and the testing list at line 189 require a helper in `T03` that returns Symphony's built-in default prompt template. The original implementation is explicit that blank workflow bodies fall back to the default prompt in `Config`, and that fallback is part of the compatibility contract, not an optional later concern.
2. The path-resolution section adds a new absolute-path normalization rule that is not present in the approved design or the Elixir behavior. Line 94 requires expanding to absolute paths before storing or returning them, while the source only establishes precedence (`explicit override`, then `<cwd>/WORKFLOW.md`) and preserves the resolved path context in errors. This risks changing user-visible paths, cache keys, and error text in ways that are not source-faithful.

## Medium / Low Suggestions

- Line 47 should be reconciled with line 38 by choosing one blank-prompt story and carrying it consistently through the loader, cache, and future prompt builder. As written, the document would send implementers in two directions.
- The store API at lines 130-135 is directionally correct, but it should explicitly call out that invalid reloads must preserve the last-known-good workflow and still keep retrying the newly configured path, matching the original store semantics.
- The testing shape should include explicit coverage for the default-prompt fallback path, not just the raw loader cases, because that is part of the compatibility surface in the source implementation.
