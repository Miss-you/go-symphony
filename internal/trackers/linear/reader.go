package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/domain"
)

const (
	defaultEndpoint = "https://api.linear.app/graphql"
	pageSize        = 50
)

var (
	ErrMissingAPIToken       = errors.New("linear api token missing")
	ErrMissingProjectSlug    = errors.New("linear project slug missing")
	ErrMissingViewerIdentity = errors.New("linear viewer identity missing")
	ErrUnknownPayload        = errors.New("linear unknown payload")
	ErrMissingEndCursor      = errors.New("linear missing end cursor")
)

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
	Body   any
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("linear api status %d", e.Status)
}

type GraphQLErrorsError struct {
	Errors any
}

func (e *GraphQLErrorsError) Error() string {
	return fmt.Sprintf("linear graphql errors: %v", e.Errors)
}

type Client interface {
	GraphQL(context.Context, string, map[string]any) (map[string]any, error)
}

type Reader struct {
	settings config.ProviderSettings
	client   Client
}

func NewReader(settings config.ProviderSettings, client Client) (*Reader, error) {
	settings.Endpoint = firstNonEmpty(strings.TrimSpace(settings.Endpoint), defaultEndpoint)
	settings.APIKey = strings.TrimSpace(settings.APIKey)
	settings.Project = strings.TrimSpace(settings.Project)
	settings.Assignee = strings.TrimSpace(settings.Assignee)

	if settings.APIKey == "" {
		return nil, ErrMissingAPIToken
	}
	if settings.Project == "" {
		return nil, ErrMissingProjectSlug
	}
	if client == nil {
		client = &HTTPClient{
			Endpoint: settings.Endpoint,
			APIKey:   settings.APIKey,
		}
	}

	return &Reader{settings: settings, client: client}, nil
}

func (r *Reader) ListCandidates(ctx context.Context) ([]domain.WorkItem, error) {
	states := normalizeStateNames(r.settings.ActiveStates)
	if len(states) == 0 {
		return []domain.WorkItem{}, nil
	}
	filter, err := r.assigneeFilter(ctx)
	if err != nil {
		return nil, err
	}
	return r.listByStatesPage(ctx, candidateQuery, states, filter, true)
}

func (r *Reader) ListByStates(ctx context.Context, states []string) ([]domain.WorkItem, error) {
	normalizedStates := normalizeStateNames(states)
	if len(normalizedStates) == 0 {
		return []domain.WorkItem{}, nil
	}
	return r.listByStatesPage(ctx, issuesByStatesQuery, normalizedStates, nil, false)
}

func (r *Reader) RefreshByIDs(ctx context.Context, ids []string) ([]domain.WorkItem, error) {
	normalizedIDs := normalizeIDs(ids)
	if len(normalizedIDs) == 0 {
		return []domain.WorkItem{}, nil
	}

	filter, err := r.assigneeFilter(ctx)
	if err != nil {
		return nil, err
	}

	order := make(map[string]int, len(normalizedIDs))
	for i, id := range normalizedIDs {
		order[id] = i
	}

	var refreshed []domain.WorkItem
	for start := 0; start < len(normalizedIDs); start += pageSize {
		end := start + pageSize
		if end > len(normalizedIDs) {
			end = len(normalizedIDs)
		}
		batch := normalizedIDs[start:end]
		body, err := r.graphql(ctx, issuesByIDsQuery, map[string]any{
			"ids":           batch,
			"first":         len(batch),
			"relationFirst": pageSize,
		})
		if err != nil {
			return nil, err
		}
		items, err := decodeIssues(body, filter, true)
		if err != nil {
			return nil, err
		}
		refreshed = append(refreshed, items...)
	}

	sort.SliceStable(refreshed, func(i, j int) bool {
		return orderIndex(refreshed[i].ID, order) < orderIndex(refreshed[j].ID, order)
	})
	return refreshed, nil
}

func (r *Reader) listByStatesPage(ctx context.Context, query string, states []string, filter *assigneeFilter, setRoutable bool) ([]domain.WorkItem, error) {
	var items []domain.WorkItem
	var after any
	for {
		body, err := r.graphql(ctx, query, map[string]any{
			"projectSlug":   r.settings.Project,
			"stateNames":    states,
			"first":         pageSize,
			"relationFirst": pageSize,
			"after":         after,
		})
		if err != nil {
			return nil, err
		}
		pageItems, page, err := decodeIssuesPage(body, filter, setRoutable)
		if err != nil {
			return nil, err
		}
		items = append(items, pageItems...)
		if !page.hasNext {
			return items, nil
		}
		if strings.TrimSpace(page.endCursor) == "" {
			return nil, ErrMissingEndCursor
		}
		after = page.endCursor
	}
}

func (r *Reader) graphql(ctx context.Context, query string, variables map[string]any) (map[string]any, error) {
	body, err := r.client.GraphQL(ctx, query, variables)
	if err != nil {
		return nil, err
	}
	if err := decodeGraphQLErrors(body); err != nil {
		return nil, err
	}
	return body, nil
}

func (r *Reader) assigneeFilter(ctx context.Context) (*assigneeFilter, error) {
	assignee := strings.TrimSpace(r.settings.Assignee)
	if assignee == "" {
		return nil, nil
	}
	if assignee != "me" {
		return &assigneeFilter{id: assignee}, nil
	}

	body, err := r.graphql(ctx, viewerQuery, map[string]any{})
	if err != nil {
		return nil, err
	}
	viewer, ok := nestedMap(body, "data", "viewer")
	if !ok {
		return nil, ErrMissingViewerIdentity
	}
	viewerID := strings.TrimSpace(stringValue(viewer["id"]))
	if viewerID == "" {
		return nil, ErrMissingViewerIdentity
	}
	return &assigneeFilter{id: viewerID}, nil
}

type assigneeFilter struct {
	id string
}

type pageInfo struct {
	hasNext   bool
	endCursor string
}

func decodeIssuesPage(body map[string]any, filter *assigneeFilter, setRoutable bool) ([]domain.WorkItem, pageInfo, error) {
	issues, ok := nestedMap(body, "data", "issues")
	if !ok {
		return nil, pageInfo{}, ErrUnknownPayload
	}
	items, err := normalizeIssueNodes(issues["nodes"], filter, setRoutable)
	if err != nil {
		return nil, pageInfo{}, err
	}
	pageMap, ok := asMap(issues["pageInfo"])
	if !ok {
		return nil, pageInfo{}, ErrUnknownPayload
	}
	page := pageInfo{
		hasNext:   boolValue(pageMap["hasNextPage"]),
		endCursor: stringValue(pageMap["endCursor"]),
	}
	return items, page, nil
}

func decodeIssues(body map[string]any, filter *assigneeFilter, setRoutable bool) ([]domain.WorkItem, error) {
	issues, ok := nestedMap(body, "data", "issues")
	if !ok {
		return nil, ErrUnknownPayload
	}
	return normalizeIssueNodes(issues["nodes"], filter, setRoutable)
}

func normalizeIssueNodes(nodes any, filter *assigneeFilter, setRoutable bool) ([]domain.WorkItem, error) {
	nodeList, ok := asSlice(nodes)
	if !ok {
		return nil, ErrUnknownPayload
	}
	items := make([]domain.WorkItem, 0, len(nodeList))
	for _, raw := range nodeList {
		issue, ok := asMap(raw)
		if !ok {
			continue
		}
		items = append(items, normalizeIssue(issue, filter, setRoutable))
	}
	return items, nil
}

func normalizeIssue(issue map[string]any, filter *assigneeFilter, setRoutable bool) domain.WorkItem {
	item := domain.WorkItem{
		ID:          stringValue(issue["id"]),
		Identifier:  stringValue(issue["identifier"]),
		Title:       stringValue(issue["title"]),
		Description: stringValue(issue["description"]),
		State:       nestedString(issue, "state", "name"),
		Priority:    intPtr(parsePriority(issue["priority"])),
		BranchName:  stringValue(issue["branchName"]),
		URL:         stringValue(issue["url"]),
		AssigneeID:  nestedString(issue, "assignee", "id"),
		Labels:      extractLabels(issue),
		BlockedBy:   extractBlockers(issue),
		CreatedAt:   timePtr(parseTime(issue["createdAt"])),
		UpdatedAt:   timePtr(parseTime(issue["updatedAt"])),
	}
	if setRoutable {
		routable := filter == nil || (item.AssigneeID != "" && item.AssigneeID == filter.id)
		item.Routable = &routable
	}
	return item
}

func decodeGraphQLErrors(body map[string]any) error {
	if body == nil {
		return ErrUnknownPayload
	}
	if errorsPayload, ok := body["errors"]; ok {
		return &GraphQLErrorsError{Errors: errorsPayload}
	}
	return nil
}

func extractLabels(issue map[string]any) []string {
	labels, ok := nestedMap(issue, "labels")
	if !ok {
		return []string{}
	}
	nodes, ok := asSlice(labels["nodes"])
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(nodes))
	for _, node := range nodes {
		label, ok := asMap(node)
		if !ok {
			continue
		}
		name := stringValue(label["name"])
		if name == "" {
			continue
		}
		result = append(result, strings.ToLower(name))
	}
	return result
}

func extractBlockers(issue map[string]any) []domain.Blocker {
	relations, ok := nestedMap(issue, "inverseRelations")
	if !ok {
		return []domain.Blocker{}
	}
	nodes, ok := asSlice(relations["nodes"])
	if !ok {
		return []domain.Blocker{}
	}
	blockers := make([]domain.Blocker, 0, len(nodes))
	for _, node := range nodes {
		relation, ok := asMap(node)
		if !ok || strings.ToLower(strings.TrimSpace(stringValue(relation["type"]))) != "blocks" {
			continue
		}
		blockerIssue, ok := asMap(relation["issue"])
		if !ok {
			continue
		}
		blockers = append(blockers, domain.Blocker{
			ID:         stringValue(blockerIssue["id"]),
			Identifier: stringValue(blockerIssue["identifier"]),
			State:      nestedString(blockerIssue, "state", "name"),
		})
	}
	return blockers
}

func normalizeStateNames(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	return normalized
}

func normalizeIDs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		id := strings.TrimSpace(value)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized
}

func orderIndex(id string, order map[string]int) int {
	if index, ok := order[id]; ok {
		return index
	}
	return len(order)
}

func nestedMap(root map[string]any, keys ...string) (map[string]any, bool) {
	current := root
	for i, key := range keys {
		next, ok := current[key]
		if !ok {
			return nil, false
		}
		if i == len(keys)-1 {
			return asMap(next)
		}
		current, ok = asMap(next)
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func nestedString(root map[string]any, keys ...string) string {
	value, ok := nestedAny(root, keys...)
	if !ok {
		return ""
	}
	return stringValue(value)
}

func nestedAny(root map[string]any, keys ...string) (any, bool) {
	var current any = root
	for _, key := range keys {
		currentMap, ok := asMap(current)
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

func asMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	default:
		return nil, false
	}
}

func asSlice(value any) ([]any, bool) {
	switch typed := value.(type) {
	case []any:
		return typed, true
	case []map[string]any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, item)
		}
		return result, true
	default:
		return nil, false
	}
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func boolValue(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}

func parsePriority(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		if math.Trunc(typed) == typed {
			return int(typed), true
		}
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return int(parsed), true
		}
	}
	return 0, false
}

func parseTime(value any) (time.Time, bool) {
	raw := stringValue(value)
	if raw == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func intPtr(value int, ok bool) *int {
	if !ok {
		return nil
	}
	return &value
}

func timePtr(value time.Time, ok bool) *time.Time {
	if !ok {
		return nil
	}
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

type HTTPClient struct {
	Endpoint string
	APIKey   string
	Client   *http.Client
}

func (c *HTTPClient) GraphQL(ctx context.Context, query string, variables map[string]any) (map[string]any, error) {
	payload := map[string]any{
		"query":     query,
		"variables": variables,
	}
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
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, &RequestError{Err: err}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &StatusError{Status: resp.StatusCode, Body: summarizeBody(body)}
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var decoded map[string]any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnknownPayload, err)
	}
	return decoded, nil
}

func summarizeBody(body []byte) string {
	const max = 1000
	summary := strings.TrimSpace(strings.Join(strings.Fields(string(body)), " "))
	if len(summary) > max {
		return summary[:max] + "...<truncated>"
	}
	return summary
}

const candidateQuery = `
query SymphonyLinearPoll($projectSlug: String!, $stateNames: [String!]!, $first: Int!, $relationFirst: Int!, $after: String) {
  issues(filter: {project: {slugId: {eq: $projectSlug}}, state: {name: {in: $stateNames}}}, first: $first, after: $after) {
    nodes {
      id
      identifier
      title
      description
      priority
      state { name }
      branchName
      url
      assignee { id }
      labels { nodes { name } }
      inverseRelations(first: $relationFirst) {
        nodes {
          type
          issue { id identifier state { name } }
        }
      }
      createdAt
      updatedAt
    }
    pageInfo { hasNextPage endCursor }
  }
}
`

const issuesByStatesQuery = `
query SymphonyLinearByStates($projectSlug: String!, $stateNames: [String!]!, $first: Int!, $relationFirst: Int!, $after: String) {
  issues(filter: {project: {slugId: {eq: $projectSlug}}, state: {name: {in: $stateNames}}}, first: $first, after: $after) {
    nodes {
      id
      identifier
      title
      description
      priority
      state { name }
      branchName
      url
      assignee { id }
      labels { nodes { name } }
      inverseRelations(first: $relationFirst) {
        nodes {
          type
          issue { id identifier state { name } }
        }
      }
      createdAt
      updatedAt
    }
    pageInfo { hasNextPage endCursor }
  }
}
`

const issuesByIDsQuery = `
query SymphonyLinearIssuesById($ids: [ID!]!, $first: Int!, $relationFirst: Int!) {
  issues(filter: {id: {in: $ids}}, first: $first) {
    nodes {
      id
      identifier
      title
      description
      priority
      state { name }
      branchName
      url
      assignee { id }
      labels { nodes { name } }
      inverseRelations(first: $relationFirst) {
        nodes {
          type
          issue { id identifier state { name } }
        }
      }
      createdAt
      updatedAt
    }
  }
}
`

const viewerQuery = `
query SymphonyLinearViewer {
  viewer { id }
}
`
