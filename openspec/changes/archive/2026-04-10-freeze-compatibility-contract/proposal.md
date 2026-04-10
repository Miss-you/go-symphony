## Why

The approved design already defines parity scope, terminology mapping, and V1 boundary rules for go-symphony, but today those rules live only in design prose. That is not durable enough for downstream execution: later tasks would need to reopen the design and reinterpret the contract, which invites scope drift and boundary regressions before implementation even starts.

## What Changes

- Create a stable OpenSpec capability that freezes the go-symphony compatibility contract.
- Promote the approved design's normative compatibility checklist, terminology mapping, and explicit non-goals into spec-level requirements.
- Make the provider-neutral core versus compatibility-shell split an explicit contract that later tasks must preserve.
- Require later changes that alter parity scope or boundary rules to update the compatibility contract in the same change.

## Capabilities

### New Capabilities

- `compatibility-contract`: Defines the normative parity surfaces, terminology mapping, and V1 boundary/non-goal rules that downstream go-symphony tasks must implement against.

### Modified Capabilities

- None.

## Impact

- OpenSpec change artifacts under `openspec/changes/freeze-compatibility-contract/`
- Main spec sync target at `openspec/specs/compatibility-contract/spec.md`
- Task documentation and workspace artifacts for `T01`
- Downstream implementation tasks that need a stable contract for parity and boundary decisions
