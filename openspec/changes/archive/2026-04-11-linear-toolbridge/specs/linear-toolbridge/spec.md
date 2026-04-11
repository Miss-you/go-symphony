## ADDED Requirements

### Requirement: Linear ToolBridge exposes the `linear_graphql` dynamic tool
The compatibility shell SHALL expose exactly one Codex dynamic tool for this slice named `linear_graphql`. The tool spec MUST include the Symphony-compatible description and an input schema requiring a non-empty `query` string with optional `variables` that are either a JSON object or null. The tool spec MUST NOT add dedicated comment, state, workpad, or provider-neutral write tool names.

#### Scenario: Tool specs advertise only linear_graphql
- **WHEN** a caller asks the Linear ToolBridge for dynamic tool specs
- **THEN** the returned specs include exactly one tool named `linear_graphql`
- **AND** the spec includes the required query and optional variables schema

### Requirement: Linear ToolBridge executes raw Linear GraphQL calls
The Linear ToolBridge SHALL dispatch `linear_graphql` calls to a narrow Linear GraphQL client interface. It MUST accept raw string arguments and object arguments with `query` and optional `variables`, trim query strings before execution, ignore legacy `operationName`, reject blank or missing queries before calling Linear, and reject non-object variables. Successful and failed GraphQL bodies MUST be returned as a single `inputText` content item with pretty JSON text when the payload is JSON-shaped.

#### Scenario: Raw query string is executed
- **WHEN** Codex calls `linear_graphql` with a raw string GraphQL query argument
- **THEN** the bridge trims the query
- **AND** calls the Linear client with the trimmed query and empty variables
- **AND** returns a successful tool result with a top-level `contentItems` entry

#### Scenario: Object arguments are executed with variables
- **WHEN** Codex calls `linear_graphql` with an object containing `query` and JSON-object `variables`
- **THEN** the bridge forwards the trimmed query and variables to the Linear client
- **AND** ignores any provided `operationName`

#### Scenario: Invalid arguments fail before Linear is called
- **WHEN** Codex calls `linear_graphql` with a blank query, missing query, malformed argument type, or non-object variables
- **THEN** the bridge returns the Symphony-compatible failed content-item payload
- **AND** the Linear client is not called

#### Scenario: GraphQL errors preserve the response body
- **WHEN** Linear returns an HTTP-successful GraphQL response containing a non-empty `errors` list
- **THEN** the bridge marks the tool result unsuccessful
- **AND** preserves the GraphQL body text in the content item

### Requirement: Linear ToolBridge preserves Symphony dynamic-tool failure payloads
The Linear ToolBridge SHALL return structured failed tool results rather than raising for user-visible dynamic-tool failures. Unknown tool names MUST return a failed content-item payload whose JSON includes `supportedTools: ["linear_graphql"]`. Missing auth, non-200 HTTP status, request failure, invalid arguments, invalid variables, and unexpected client failures MUST retain distinct Symphony-compatible messages.

#### Scenario: Unknown tool returns supported tool list
- **WHEN** the bridge receives a tool call for an unknown tool name
- **THEN** it returns `success: false`
- **AND** the content-item JSON names `linear_graphql` under `supportedTools`

#### Scenario: Linear status and request failures are distinct
- **WHEN** the Linear client reports a non-200 status or transport/request failure
- **THEN** the bridge returns a failed content-item payload with the corresponding Symphony-compatible error message

### Requirement: Linear ToolBridge owns provider-specific write helpers
The compatibility shell SHALL provide Linear-specific helper methods for comment creation and issue state updates without adding write methods to `internal/tracker`, provider-specific fields to `internal/domain`, or Linear write logic to `internal/orchestrator`.

#### Scenario: Comment helper runs commentCreate
- **WHEN** compatibility-shell runtime code asks the bridge to create a Linear comment
- **THEN** the bridge runs the `commentCreate` mutation with the issue id and body
- **AND** succeeds only when Linear returns `data.commentCreate.success == true`

#### Scenario: State helper resolves state before issueUpdate
- **WHEN** compatibility-shell runtime code asks the bridge to move a Linear issue to a named state
- **THEN** the bridge resolves the state id through the issue team state query
- **AND** runs `issueUpdate` with the resolved state id

#### Scenario: Write helper failures remain provider-specific
- **WHEN** comment creation fails, state lookup returns no state, or issue update fails
- **THEN** the bridge returns a Linear ToolBridge error category
- **AND** no provider-neutral tracker write API is required

### Requirement: Linear ToolBridge remains a compatibility-shell leaf
The Linear ToolBridge SHALL depend only on the generic Codex tool boundary, config/provider settings, and a narrow Linear GraphQL client interface. It MUST NOT import or require `internal/tracker`, `internal/domain`, or `internal/orchestrator`.

#### Scenario: Dependency guard rejects core write leakage
- **WHEN** the bridge package dependency graph is inspected
- **THEN** it does not include provider-neutral tracker, domain, or orchestrator packages
- **AND** Linear write behavior remains owned by the compatibility shell
