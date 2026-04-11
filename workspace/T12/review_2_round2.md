# T12 Final Implementation Plan Re-Review

Score: 91/100

- Symphony alignment and source fidelity: 28/30
- Go-native simplicity and maintainability: 18/20
- No overdesign / clean boundaries: 18/20
- Implementation clarity and testability: 13/15
- Verification coverage and landing safety: 14/15

Accepted: Yes

## High-Severity Issues

None.

The previous blocker is resolved. The plan now states an explicit unsupported-tool payload shape:
`{"error":{"message":"Unsupported dynamic tool: <tool>.","supportedTools":["linear_graphql"]}}`
and it keeps that as a bridge-local failed tool result instead of routing through `codex.ErrUnsupportedTool`.
That is specific enough to preserve the Symphony contract for unknown tools.

## Medium / Low Issues

1. The verification section is solid, but it still stays slightly abstract about the OpenSpec gate.
   - It says to validate the change if one is created, which is directionally right.
   - Naming the exact `openspec validate <change>` command would make landing risk easier to audit, but this is not a blocker.

2. `TestBridgeDoesNotRequireCoreTrackerWrites` is now well-scoped, but it depends on `go list -deps` assertions staying stable.
   - That is acceptable for a boundary guard, but the test should fail with a clear dependency list if the import graph drifts.

## Decision

Accept this plan.

It now covers the remaining T12 risks cleanly: explicit supported-tools encoding, Linear write-helper scope in the compatibility shell, and verification gates that cover both the bridge package and the broader repo before closure.
