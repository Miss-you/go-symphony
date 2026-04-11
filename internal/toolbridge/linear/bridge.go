package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/Miss-you/go-symphony/internal/codex"
	"github.com/Miss-you/go-symphony/internal/config"
)

const (
	linearGraphQLTool      = "linear_graphql"
	defaultEndpoint        = "https://api.linear.app/graphql"
	linearGraphQLDesc      = "Execute a raw GraphQL query or mutation against Linear using Symphony's configured auth."
	missingAuthMessage     = "Symphony is missing Linear auth. Set `linear.api_key` in `WORKFLOW.md` or export `LINEAR_API_KEY`."
	missingQueryMessage    = "`linear_graphql` requires a non-empty `query` string."
	invalidArgsMessage     = "`linear_graphql` expects either a GraphQL query string or an object with `query` and optional `variables`."
	invalidVariablesMsg    = "`linear_graphql.variables` must be a JSON object when provided."
	requestFailedMessage   = "Linear GraphQL request failed before receiving a successful response."
	unexpectedToolErrorMsg = "Linear GraphQL tool execution failed."
)

var (
	ErrMissingAPIToken     = errors.New("linear api token missing")
	ErrCommentCreateFailed = errors.New("linear comment create failed")
	ErrStateNotFound       = errors.New("linear state not found")
	ErrIssueUpdateFailed   = errors.New("linear issue update failed")
)

type Client interface {
	GraphQL(context.Context, string, map[string]any) (map[string]any, error)
}

type Bridge struct {
	client Client
}

func New(settings config.ProviderSettings, client Client) (*Bridge, error) {
	if client == nil {
		apiKey := strings.TrimSpace(settings.APIKey)
		endpoint := strings.TrimSpace(settings.Endpoint)
		if endpoint == "" {
			endpoint = defaultEndpoint
		}
		client = &HTTPClient{Endpoint: endpoint, APIKey: apiKey}
	}
	return &Bridge{client: client}, nil
}

func (b *Bridge) ToolSpecs() []codex.ToolSpec {
	return []codex.ToolSpec{{
		Name:        linearGraphQLTool,
		Description: linearGraphQLDesc,
		InputSchema: linearGraphQLInputSchema(),
	}}
}

func (b *Bridge) HandleTool(ctx context.Context, call codex.ToolCall) (codex.ToolResult, error) {
	if strings.TrimSpace(call.Name) != linearGraphQLTool {
		return failureResponse(map[string]any{
			"error": map[string]any{
				"message":        fmt.Sprintf("Unsupported dynamic tool: %q.", call.Name),
				"supportedTools": []string{linearGraphQLTool},
			},
		}), nil
	}

	query, variables, err := normalizeArguments(call.Arguments)
	if err != nil {
		return failureResponse(toolErrorPayload(err)), nil
	}
	body, err := b.client.GraphQL(ctx, query, variables)
	if err != nil {
		return failureResponse(clientErrorPayload(err)), nil
	}
	return graphqlResponse(body), nil
}

func (b *Bridge) CreateComment(ctx context.Context, issueID, body string) error {
	response, err := b.client.GraphQL(ctx, createCommentMutation, map[string]any{
		"issueId": issueID,
		"body":    body,
	})
	if err != nil {
		return err
	}
	if nestedBool(response, "data", "commentCreate", "success") {
		return nil
	}
	return ErrCommentCreateFailed
}

func (b *Bridge) UpdateIssueState(ctx context.Context, issueID, stateName string) error {
	stateResponse, err := b.client.GraphQL(ctx, stateLookupQuery, map[string]any{
		"issueId":   issueID,
		"stateName": stateName,
	})
	if err != nil {
		return err
	}
	stateID := firstStateID(stateResponse)
	if stateID == "" {
		return ErrStateNotFound
	}

	response, err := b.client.GraphQL(ctx, updateStateMutation, map[string]any{
		"issueId": issueID,
		"stateId": stateID,
	})
	if err != nil {
		return err
	}
	if nestedBool(response, "data", "issueUpdate", "success") {
		return nil
	}
	return ErrIssueUpdateFailed
}

type HTTPClient struct {
	Endpoint string
	APIKey   string
	Client   *http.Client
}

func (c *HTTPClient) GraphQL(ctx context.Context, query string, variables map[string]any) (map[string]any, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, ErrMissingAPIToken
	}
	payload := map[string]any{"query": query, "variables": variables}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, &RequestError{Err: err}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, &RequestError{Err: err}
	}
	req.Header.Set("Authorization", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, &RequestError{Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, &RequestError{Err: err}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &StatusError{Status: resp.StatusCode}
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	var decoded map[string]any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

type RequestError struct {
	Err error
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("linear api request failed: %v", e.Err)
}

func (e *RequestError) Unwrap() error {
	return e.Err
}

type StatusError struct {
	Status int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("linear api status %d", e.Status)
}

var (
	errMissingQuery     = errors.New("missing query")
	errInvalidArgs      = errors.New("invalid arguments")
	errInvalidVariables = errors.New("invalid variables")
)

func normalizeArguments(arguments any) (string, map[string]any, error) {
	switch typed := arguments.(type) {
	case string:
		query := strings.TrimSpace(typed)
		if query == "" {
			return "", nil, errMissingQuery
		}
		return query, map[string]any{}, nil
	case map[string]any:
		rawQuery, _ := typed["query"].(string)
		query := strings.TrimSpace(rawQuery)
		if query == "" {
			return "", nil, errMissingQuery
		}
		variables, ok := normalizeVariables(typed)
		if !ok {
			return "", nil, errInvalidVariables
		}
		return query, variables, nil
	default:
		return "", nil, errInvalidArgs
	}
}

func normalizeVariables(arguments map[string]any) (map[string]any, bool) {
	raw, ok := arguments["variables"]
	if !ok || raw == nil {
		return map[string]any{}, true
	}
	variables, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	return cloneMap(variables), true
}

func graphqlResponse(body map[string]any) codex.ToolResult {
	return codex.ToolResult{
		Success:      !hasGraphQLErrors(body),
		ContentItems: []codex.ToolContentItem{textItem(encodePayload(body))},
	}
}

func failureResponse(payload map[string]any) codex.ToolResult {
	return codex.ToolResult{
		Success:      false,
		ContentItems: []codex.ToolContentItem{textItem(encodePayload(payload))},
	}
}

func textItem(text string) codex.ToolContentItem {
	return codex.ToolContentItem{Type: "inputText", Text: text}
}

func toolErrorPayload(err error) map[string]any {
	message := invalidArgsMessage
	switch {
	case errors.Is(err, errMissingQuery):
		message = missingQueryMessage
	case errors.Is(err, errInvalidVariables):
		message = invalidVariablesMsg
	}
	return map[string]any{"error": map[string]any{"message": message}}
}

func clientErrorPayload(err error) map[string]any {
	if errors.Is(err, ErrMissingAPIToken) {
		return map[string]any{"error": map[string]any{"message": missingAuthMessage}}
	}
	if status, ok := statusFromError(err); ok {
		return map[string]any{"error": map[string]any{"message": fmt.Sprintf("Linear GraphQL request failed with HTTP %d.", status), "status": status}}
	}
	if isRequestError(err) {
		return map[string]any{"error": map[string]any{"message": requestFailedMessage, "reason": err.Error()}}
	}
	return map[string]any{"error": map[string]any{"message": unexpectedToolErrorMsg, "reason": err.Error()}}
}

func statusFromError(err error) (int, bool) {
	for current := err; current != nil; current = errors.Unwrap(current) {
		value := reflect.Indirect(reflect.ValueOf(current))
		if value.IsValid() && value.Kind() == reflect.Struct {
			field := value.FieldByName("Status")
			if field.IsValid() && field.CanInt() {
				return int(field.Int()), true
			}
		}
	}
	return 0, false
}

func isRequestError(err error) bool {
	for current := err; current != nil; current = errors.Unwrap(current) {
		name := reflect.TypeOf(current).String()
		if strings.HasSuffix(name, ".RequestError") || strings.Contains(current.Error(), "linear api request failed") {
			return true
		}
	}
	return false
}

func hasGraphQLErrors(body map[string]any) bool {
	errorsValue, ok := body["errors"]
	if !ok {
		return false
	}
	switch typed := errorsValue.(type) {
	case []any:
		return len(typed) > 0
	case []map[string]any:
		return len(typed) > 0
	default:
		return false
	}
}

func encodePayload(payload any) string {
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Sprint(payload)
	}
	return string(encoded)
}

func linearGraphQLInputSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"query"},
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "GraphQL query or mutation document to execute against Linear.",
			},
			"variables": map[string]any{
				"type":                 []any{"object", "null"},
				"description":          "Optional GraphQL variables object.",
				"additionalProperties": true,
			},
		},
	}
}

func nestedBool(root map[string]any, keys ...string) bool {
	value, ok := nestedAny(root, keys...)
	if !ok {
		return false
	}
	typed, ok := value.(bool)
	return ok && typed
}

func firstStateID(root map[string]any) string {
	nodes, ok := nestedAny(root, "data", "issue", "team", "states", "nodes")
	if !ok {
		return ""
	}
	list, ok := nodes.([]any)
	if !ok || len(list) == 0 {
		return ""
	}
	first, ok := list[0].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := first["id"].(string)
	return strings.TrimSpace(id)
}

func nestedAny(root map[string]any, keys ...string) (any, bool) {
	var current any = root
	for _, key := range keys {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = currentMap[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

const createCommentMutation = `
mutation SymphonyCreateComment($issueId: ID!, $body: String!) {
  commentCreate(input: {issueId: $issueId, body: $body}) {
    success
  }
}`

const updateStateMutation = `
mutation SymphonyUpdateIssueState($issueId: ID!, $stateId: ID!) {
  issueUpdate(id: $issueId, input: {stateId: $stateId}) {
    success
  }
}`

const stateLookupQuery = `
query SymphonyResolveStateId($issueId: ID!, $stateName: String!) {
  issue(id: $issueId) {
    team {
      states(filter: {name: {eq: $stateName}}, first: 1) {
        nodes {
          id
        }
      }
    }
  }
}`
