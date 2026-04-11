## Context

T11 restored the Linear read adapter and T09 restored the Codex app-server protocol, but the current Go tree still lacks the Symphony Linear ToolBridge surface. The remaining gap is not just a new tool implementation: Symphony's app-server contract also allows raw string tool arguments and top-level dynamic-tool `contentItems`, and the compatibility shell needs Linear write helpers without turning the provider-neutral core into a write API.

This change sits across the Codex protocol boundary, the Linear compatibility shell, and the later workflow/assembly layer that will inject the bridge into session configuration.

## Goals / Non-Goals

**Goals:**
- Preserve the Symphony `linear_graphql` dynamic tool contract in the compatibility shell.
- Keep `linear_graphql` as the only Codex-advertised Linear tool in this slice.
- Preserve raw string tool arguments and top-level `contentItems` in the Codex protocol boundary.
- Add provider-specific Linear write helpers for comment creation and issue state updates without widening tracker/core interfaces.
- Keep the bridge isolated from `internal/domain`, `internal/orchestrator`, and `internal/tracker`.

**Non-Goals:**
- No universal tracker write interface.
- No provider-specific write fields in the core domain model.
- No new core workflow abstraction for Linear writes.
- No implementation of the workflow/assembly wiring in this change.
- No Lark-specific behavior.

## Decisions

1. Put the Linear ToolBridge in `internal/toolbridge/linear`.
   - This keeps the feature in the compatibility shell where provider-specific behavior belongs.
   - Alternative considered: adding Linear write helpers to `internal/trackers/linear`. Rejected because that would entangle read and write concerns and make the core-facing tracker package the wrong ownership boundary.

2. Keep `linear_graphql` as a raw passthrough tool, not a family of provider-specific Codex tools.
   - Symphony already exposes the raw GraphQL surface, and it is the least opinionated way to preserve parity.
   - Alternative considered: add dedicated Codex tools for comment and issue-state mutations. Rejected because that would overfit the surface and duplicate a capability that raw GraphQL already covers.

3. Extend `internal/codex` only at the generic protocol boundary.
   - `ToolCall.Arguments` needs to accept either object-shaped or raw string input.
   - `ToolResult` needs top-level `contentItems` so the JSON shape matches Symphony's app-server payloads.
   - Alternative considered: keep `Arguments` as a map and encode raw string inputs inside a wrapper object. Rejected because it would lose the original protocol shape and make raw string preservation impossible.

4. Model unsupported tools as a bridge-local failure payload that includes `supportedTools`.
   - This preserves the legacy dynamic-tool response contract while keeping the bridge responsible for compatibility behavior.
   - Alternative considered: route unsupported tools through a generic `codex.ErrUnsupportedTool`. Rejected because it would collapse the Symphony-specific payload shape the task is meant to preserve.

5. Keep provider-specific write helpers as ordinary bridge methods.
   - `CreateComment` and `UpdateIssueState` are useful for later workflow/runtime assembly without becoming part of Codex tool advertisement.
   - Alternative considered: expose them as separate Codex tools. Rejected because the task scope explicitly keeps them out of the tool list and avoids expanding the user-facing tool surface.

## Risks / Trade-offs

- [Risk] The Codex boundary changes touch protocol serialization. → Mitigation: add regression tests that assert raw string arguments and top-level `contentItems` serialize exactly as expected.
- [Risk] The bridge could drift into a second tracker API by accident. → Mitigation: keep the package dependency graph narrow and add explicit tests that `internal/toolbridge/linear` does not depend on `internal/tracker`, `internal/orchestrator`, or `internal/domain`.
- [Risk] Failure payload compatibility may diverge from Symphony if the unsupported-tool shape is underspecified. → Mitigation: pin the exact `supportedTools` payload in tests and spec text before implementation.
- [Risk] The assembly layer needed to wire the bridge may land in a later task. → Mitigation: keep this change limited to the bridge, protocol boundary, and compatibility-shell contract so it remains apply-ready even before wiring exists.
