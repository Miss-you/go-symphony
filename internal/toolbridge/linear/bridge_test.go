package linear

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/Miss-you/go-symphony/internal/codex"
	"github.com/Miss-you/go-symphony/internal/config"
	lineartracker "github.com/Miss-you/go-symphony/internal/trackers/linear"
)

func TestNewConstructsBridgeAndToolSpecsExposeLinearGraphQL(t *testing.T) {
	t.Parallel()

	var _ Client = (*lineartracker.HTTPClient)(nil)
	var _ codex.ToolHandler = (*Bridge)(nil)

	_, err := New(config.ProviderSettings{}, nil)
	if !errors.Is(err, ErrMissingAPIToken) {
		t.Fatalf("New without client/token error = %v, want %v", err, ErrMissingAPIToken)
	}

	bridge := mustBridge(t, &fakeClient{})
	specs := bridge.ToolSpecs()
	if len(specs) != 1 {
		t.Fatalf("ToolSpecs len = %d, want 1", len(specs))
	}
	spec := specs[0]
	if spec.Name != "linear_graphql" {
		t.Fatalf("tool name = %q, want linear_graphql", spec.Name)
	}
	if !strings.Contains(spec.Description, "Linear") {
		t.Fatalf("description = %q, want Linear context", spec.Description)
	}
	schema := spec.InputSchema
	if schema["type"] != "object" || schema["additionalProperties"] != false {
		t.Fatalf("schema basics = %#v, want object with additionalProperties=false", schema)
	}
	if !reflect.DeepEqual(schema["required"], []any{"query"}) {
		t.Fatalf("required = %#v, want query", schema["required"])
	}
	properties := schema["properties"].(map[string]any)
	if properties["query"] == nil || properties["variables"] == nil {
		t.Fatalf("properties = %#v, want query and variables", properties)
	}

	specs[0].Name = "mutated"
	if got := bridge.ToolSpecs()[0].Name; got != "linear_graphql" {
		t.Fatalf("ToolSpecs returned shared slice, got name %q", got)
	}
}

func TestHandleToolDispatchesLinearGraphQL(t *testing.T) {
	t.Parallel()

	wantBody := map[string]any{"data": map[string]any{"viewer": map[string]any{"id": "usr_123"}}}
	client := &fakeClient{responses: []fakeResponse{{body: wantBody}}}
	bridge := mustBridge(t, client)

	result, err := bridge.HandleTool(context.Background(), codex.ToolCall{
		Name:      "linear_graphql",
		Arguments: "  query Viewer { viewer { id } }  ",
	})
	if err != nil {
		t.Fatalf("HandleTool returned error: %v", err)
	}
	if !result.Success {
		t.Fatalf("result success = false, want true: %#v", result)
	}
	if len(client.calls) != 1 {
		t.Fatalf("client call count = %d, want 1", len(client.calls))
	}
	if client.calls[0].query != "query Viewer { viewer { id } }" {
		t.Fatalf("query = %q, want trimmed query", client.calls[0].query)
	}
	if len(client.calls[0].variables) != 0 {
		t.Fatalf("variables = %#v, want empty object", client.calls[0].variables)
	}
	if got := decodeContentText(t, result); !reflect.DeepEqual(got, wantBody) {
		t.Fatalf("content text = %#v, want GraphQL body", got)
	}

	client = &fakeClient{responses: []fakeResponse{{body: map[string]any{"data": map[string]any{"ok": true}}}}}
	bridge = mustBridge(t, client)
	result, err = bridge.HandleTool(context.Background(), codex.ToolCall{
		Name: "linear_graphql",
		Arguments: map[string]any{
			"query":         " query Viewer { viewer { id } } ",
			"variables":     map[string]any{"includeTeams": false},
			"operationName": "Ignored",
		},
	})
	if err != nil {
		t.Fatalf("HandleTool object returned error: %v", err)
	}
	if !result.Success {
		t.Fatalf("object result success = false, want true: %#v", result)
	}
	assertVariable(t, client.calls[0].variables, "includeTeams", false)
}

func TestHandleToolRejectsInvalidArgumentsBeforeLinearCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		arguments any
		message   string
	}{
		{name: "blank raw", arguments: "   ", message: "`linear_graphql` requires a non-empty `query` string."},
		{name: "missing query", arguments: map[string]any{"variables": map[string]any{"id": "issue-1"}}, message: "`linear_graphql` requires a non-empty `query` string."},
		{name: "invalid type", arguments: []any{"bad"}, message: "`linear_graphql` expects either a GraphQL query string or an object with `query` and optional `variables`."},
		{name: "invalid variables", arguments: map[string]any{"query": "query Viewer { viewer { id } }", "variables": []any{"bad"}}, message: "`linear_graphql.variables` must be a JSON object when provided."},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &fakeClient{}
			bridge := mustBridge(t, client)
			result, err := bridge.HandleTool(context.Background(), codex.ToolCall{Name: "linear_graphql", Arguments: tt.arguments})
			if err != nil {
				t.Fatalf("HandleTool returned error: %v", err)
			}
			if result.Success {
				t.Fatalf("result success = true, want false")
			}
			if len(client.calls) != 0 {
				t.Fatalf("client was called for invalid arguments: %#v", client.calls)
			}
			payload := decodeContentText(t, result)
			errorPayload := payload["error"].(map[string]any)
			if errorPayload["message"] != tt.message {
				t.Fatalf("message = %#v, want %q", errorPayload["message"], tt.message)
			}
		})
	}
}

func TestHandleToolReturnsSupportedToolsForUnknownTool(t *testing.T) {
	t.Parallel()

	bridge := mustBridge(t, &fakeClient{})
	result, err := bridge.HandleTool(context.Background(), codex.ToolCall{Name: "not_a_real_tool", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("HandleTool returned error: %v", err)
	}
	if result.Success {
		t.Fatalf("result success = true, want false")
	}
	payload := decodeContentText(t, result)
	errorPayload := payload["error"].(map[string]any)
	if errorPayload["message"] != `Unsupported dynamic tool: "not_a_real_tool".` {
		t.Fatalf("message = %#v, want unsupported tool message", errorPayload["message"])
	}
	if !reflect.DeepEqual(errorPayload["supportedTools"], []any{"linear_graphql"}) {
		t.Fatalf("supportedTools = %#v, want linear_graphql", errorPayload["supportedTools"])
	}
}

func TestHandleToolMapsTransportStatusAndGraphQLErrorDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		resp    fakeResponse
		success bool
		message string
	}{
		{
			name:    "status",
			resp:    fakeResponse{err: &lineartracker.StatusError{Status: 503}},
			success: false,
			message: "Linear GraphQL request failed with HTTP 503.",
		},
		{
			name:    "request",
			resp:    fakeResponse{err: &lineartracker.RequestError{Err: errors.New("timeout")}},
			success: false,
			message: "Linear GraphQL request failed before receiving a successful response.",
		},
		{
			name:    "unexpected",
			resp:    fakeResponse{err: errors.New("boom")},
			success: false,
			message: "Linear GraphQL tool execution failed.",
		},
		{
			name:    "graphql errors",
			resp:    fakeResponse{body: map[string]any{"data": nil, "errors": []any{map[string]any{"message": "Unknown field"}}}},
			success: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bridge := mustBridge(t, &fakeClient{responses: []fakeResponse{tt.resp}})
			result, err := bridge.HandleTool(context.Background(), codex.ToolCall{Name: "linear_graphql", Arguments: "query Viewer { viewer { id } }"})
			if err != nil {
				t.Fatalf("HandleTool returned error: %v", err)
			}
			if result.Success != tt.success {
				t.Fatalf("success = %v, want %v", result.Success, tt.success)
			}
			payload := decodeContentText(t, result)
			if tt.message != "" {
				errorPayload := payload["error"].(map[string]any)
				if errorPayload["message"] != tt.message {
					t.Fatalf("message = %#v, want %q", errorPayload["message"], tt.message)
				}
			} else if payload["errors"] == nil {
				t.Fatalf("payload = %#v, want preserved GraphQL errors", payload)
			}
		})
	}
}

func TestLinearWriteHelpers(t *testing.T) {
	t.Parallel()

	t.Run("create comment success and failure", func(t *testing.T) {
		t.Parallel()

		client := &fakeClient{responses: []fakeResponse{{body: commentCreateBody(true)}}}
		bridge := mustBridge(t, client)
		if err := bridge.CreateComment(context.Background(), "issue-1", "body"); err != nil {
			t.Fatalf("CreateComment returned error: %v", err)
		}
		if !strings.Contains(client.calls[0].query, "commentCreate") {
			t.Fatalf("query = %q, want commentCreate mutation", client.calls[0].query)
		}
		assertVariable(t, client.calls[0].variables, "issueId", "issue-1")
		assertVariable(t, client.calls[0].variables, "body", "body")

		bridge = mustBridge(t, &fakeClient{responses: []fakeResponse{{body: commentCreateBody(false)}}})
		if err := bridge.CreateComment(context.Background(), "issue-1", "body"); !errors.Is(err, ErrCommentCreateFailed) {
			t.Fatalf("CreateComment failure error = %v, want %v", err, ErrCommentCreateFailed)
		}
	})

	t.Run("update state resolves before mutation", func(t *testing.T) {
		t.Parallel()

		client := &fakeClient{responses: []fakeResponse{
			{body: stateLookupBody("state-1")},
			{body: issueUpdateBody(true)},
		}}
		bridge := mustBridge(t, client)
		if err := bridge.UpdateIssueState(context.Background(), "issue-1", "Done"); err != nil {
			t.Fatalf("UpdateIssueState returned error: %v", err)
		}
		if len(client.calls) != 2 {
			t.Fatalf("call count = %d, want lookup and update", len(client.calls))
		}
		if !strings.Contains(client.calls[0].query, "SymphonyResolveStateId") {
			t.Fatalf("first query = %q, want state lookup", client.calls[0].query)
		}
		assertVariable(t, client.calls[0].variables, "stateName", "Done")
		if !strings.Contains(client.calls[1].query, "issueUpdate") {
			t.Fatalf("second query = %q, want issueUpdate", client.calls[1].query)
		}
		assertVariable(t, client.calls[1].variables, "stateId", "state-1")

		bridge = mustBridge(t, &fakeClient{responses: []fakeResponse{{body: stateLookupBody("")}}})
		if err := bridge.UpdateIssueState(context.Background(), "issue-1", "Done"); !errors.Is(err, ErrStateNotFound) {
			t.Fatalf("UpdateIssueState missing state error = %v, want %v", err, ErrStateNotFound)
		}

		bridge = mustBridge(t, &fakeClient{responses: []fakeResponse{
			{body: stateLookupBody("state-1")},
			{body: issueUpdateBody(false)},
		}})
		if err := bridge.UpdateIssueState(context.Background(), "issue-1", "Done"); !errors.Is(err, ErrIssueUpdateFailed) {
			t.Fatalf("UpdateIssueState failed update error = %v, want %v", err, ErrIssueUpdateFailed)
		}
	})
}

func TestBridgeDoesNotDependOnCoreTrackerDomainOrOrchestrator(t *testing.T) {
	t.Parallel()

	output, err := exec.Command("go", "list", "-deps", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps failed: %v\n%s", err, output)
	}
	forbidden := []string{
		"github.com/Miss-you/go-symphony/internal/domain",
		"github.com/Miss-you/go-symphony/internal/orchestrator",
		"github.com/Miss-you/go-symphony/internal/tracker",
	}
	deps := strings.Split(string(output), "\n")
	for _, forbiddenImport := range forbidden {
		for _, dep := range deps {
			if dep == forbiddenImport {
				t.Fatalf("forbidden dependency %q found in deps:\n%s", forbiddenImport, output)
			}
		}
	}
}

func mustBridge(t *testing.T, client Client) *Bridge {
	t.Helper()
	bridge, err := New(config.ProviderSettings{}, client)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	return bridge
}

func decodeContentText(t *testing.T, result codex.ToolResult) map[string]any {
	t.Helper()
	if len(result.ContentItems) != 1 {
		t.Fatalf("content item count = %d, want 1: %#v", len(result.ContentItems), result)
	}
	if result.ContentItems[0].Type != "inputText" {
		t.Fatalf("content item type = %q, want inputText", result.ContentItems[0].Type)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.ContentItems[0].Text), &payload); err != nil {
		t.Fatalf("content text is not JSON: %v\n%s", err, result.ContentItems[0].Text)
	}
	return payload
}

func assertVariable(t *testing.T, variables map[string]any, key string, want any) {
	t.Helper()
	if !reflect.DeepEqual(variables[key], want) {
		t.Fatalf("variable %q = %#v, want %#v in %#v", key, variables[key], want, variables)
	}
}

func commentCreateBody(success bool) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"commentCreate": map[string]any{"success": success},
		},
	}
}

func issueUpdateBody(success bool) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issueUpdate": map[string]any{"success": success},
		},
	}
}

func stateLookupBody(stateID string) map[string]any {
	nodes := []any{}
	if stateID != "" {
		nodes = append(nodes, map[string]any{"id": stateID})
	}
	return map[string]any{
		"data": map[string]any{
			"issue": map[string]any{
				"team": map[string]any{
					"states": map[string]any{
						"nodes": nodes,
					},
				},
			},
		},
	}
}

type fakeClient struct {
	calls     []fakeCall
	responses []fakeResponse
}

type fakeCall struct {
	query     string
	variables map[string]any
}

type fakeResponse struct {
	body map[string]any
	err  error
}

func (c *fakeClient) GraphQL(_ context.Context, query string, variables map[string]any) (map[string]any, error) {
	c.calls = append(c.calls, fakeCall{query: query, variables: cloneMap(variables)})
	if len(c.responses) == 0 {
		return map[string]any{"data": map[string]any{}}, nil
	}
	resp := c.responses[0]
	c.responses = c.responses[1:]
	return resp.body, resp.err
}
