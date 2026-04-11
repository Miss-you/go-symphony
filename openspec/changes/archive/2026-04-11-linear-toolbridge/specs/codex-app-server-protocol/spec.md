## MODIFIED Requirements

### Requirement: Dynamic tool calls are dispatched through an injected handler
The system MUST dispatch `item/tool/call` requests through an injected tool handler.

The system MUST keep the dispatch boundary generic so compatibility-shell code can provide provider-specific behavior later.

The system MUST preserve object-shaped and raw string tool arguments when forwarding a tool call to the injected handler.

The system MUST allow handler results to include top-level `contentItems` inside the JSON-RPC `result` object so compatibility-shell tools can return Symphony-style dynamic-tool output.

When the session receives a tool call for an unsupported tool and the injected handler reports it as unsupported, it MUST return a structured failure result instead of stalling the turn.

#### Scenario: Tool call is dispatched to handler
- **WHEN** the app-server issues an `item/tool/call`
- **THEN** the session forwards the call to the injected tool handler
- **AND** returns the handler result to the protocol stream

#### Scenario: Raw string arguments are preserved
- **WHEN** the app-server issues an `item/tool/call` whose `params.arguments` value is a raw string
- **THEN** the injected handler receives that raw string value
- **AND** the session does not replace it with an empty object

#### Scenario: Tool content items are top-level within the tool result
- **WHEN** the injected handler returns a tool result with content items
- **THEN** the session writes a JSON-RPC response whose `result.contentItems` contains those content items
- **AND** the content items are not nested under `result.result.contentItems`

#### Scenario: Unsupported tool returns failure
- **WHEN** the app-server requests a tool that the injected handler does not support
- **THEN** the session returns a structured failure result
- **AND** the turn does not hang waiting for an implementation that does not exist
