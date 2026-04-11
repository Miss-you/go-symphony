package linear

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/tracker"
)

func TestReaderImplementsTrackerReaderAndValidatesConfig(t *testing.T) {
	t.Parallel()

	var _ tracker.TrackerReader = (*Reader)(nil)

	_, err := NewReader(config.ProviderSettings{APIKey: "", Project: "project"}, &fakeClient{})
	if !errors.Is(err, ErrMissingAPIToken) {
		t.Fatalf("NewReader missing token error = %v, want %v", err, ErrMissingAPIToken)
	}

	_, err = NewReader(config.ProviderSettings{APIKey: "token", Project: " "}, &fakeClient{})
	if !errors.Is(err, ErrMissingProjectSlug) {
		t.Fatalf("NewReader missing project error = %v, want %v", err, ErrMissingProjectSlug)
	}
}

func TestListCandidatesPaginatesNormalizesAndPreservesOrder(t *testing.T) {
	t.Parallel()

	client := &fakeClient{responses: []fakeResponse{
		{body: issuesPage(
			[]map[string]any{linearIssue("issue-1", "MT-1", "Todo",
				withPriority(2),
				withAssignee("user-1"),
				withLabels("Backend", "Infra"),
				withBlocker("blocker-1", "MT-2", "In Progress"),
				withCreatedAt("2026-04-10T00:00:00Z"),
				withUpdatedAt("2026-04-11T00:00:00Z"),
			)},
			true,
			"cursor-1",
		)},
		{body: issuesPage(
			[]map[string]any{linearIssue("issue-2", "MT-3", "In Progress")},
			false,
			"",
		)},
	}}
	reader := mustReader(t, config.ProviderSettings{
		APIKey:       "token",
		Project:      "project",
		ActiveStates: []string{"Todo", "In Progress"},
	}, client)

	got, err := reader.ListCandidates(context.Background())
	if err != nil {
		t.Fatalf("ListCandidates error = %v, want nil", err)
	}
	if identifiers := identifiers(got); !slices.Equal(identifiers, []string{"MT-1", "MT-3"}) {
		t.Fatalf("candidate identifiers = %#v, want page order", identifiers)
	}
	if len(client.calls) != 2 {
		t.Fatalf("GraphQL call count = %d, want 2", len(client.calls))
	}
	if !contains(client.calls[0].query, "SymphonyLinearPoll") {
		t.Fatalf("candidate query = %q, want SymphonyLinearPoll", client.calls[0].query)
	}
	assertVariable(t, client.calls[0].variables, "projectSlug", "project")
	assertVariable(t, client.calls[0].variables, "stateNames", []string{"Todo", "In Progress"})
	assertVariable(t, client.calls[0].variables, "first", 50)
	assertVariable(t, client.calls[0].variables, "after", nil)
	assertVariable(t, client.calls[1].variables, "after", "cursor-1")

	item := got[0]
	if item.ID != "issue-1" || item.Title != "Title MT-1" || item.Description != "Description MT-1" || item.State != "Todo" {
		t.Fatalf("normalized identity fields mismatch: %#v", item)
	}
	if item.Priority == nil || *item.Priority != 2 {
		t.Fatalf("priority = %v, want 2", item.Priority)
	}
	if item.BranchName != "mt-1-branch" || item.URL != "https://linear.test/MT-1" {
		t.Fatalf("branch/url mismatch: %#v", item)
	}
	if item.AssigneeID != "user-1" {
		t.Fatalf("assignee id = %q, want user-1", item.AssigneeID)
	}
	if !slices.Equal(item.Labels, []string{"backend", "infra"}) {
		t.Fatalf("labels = %#v, want lowercase labels", item.Labels)
	}
	if len(item.BlockedBy) != 1 || item.BlockedBy[0].ID != "blocker-1" || item.BlockedBy[0].State != "In Progress" {
		t.Fatalf("blockers = %#v, want blocks relation only", item.BlockedBy)
	}
	if item.Routable == nil || !*item.Routable {
		t.Fatalf("Routable = %v, want true for no assignee filter", item.Routable)
	}
	assertTime(t, item.CreatedAt, time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC))
	assertTime(t, item.UpdatedAt, time.Date(2026, time.April, 11, 0, 0, 0, 0, time.UTC))
}

func TestListByStatesIsNoopForEmptyInputAndDoesNotRoute(t *testing.T) {
	t.Parallel()

	client := &fakeClient{responses: []fakeResponse{
		{body: issuesPage([]map[string]any{
			linearIssue("issue-1", "MT-1", "Done", withAssignee("someone-else")),
		}, false, "")},
	}}
	reader := mustReader(t, config.ProviderSettings{
		APIKey:   "token",
		Project:  "project",
		Assignee: "me",
	}, client)

	empty, err := reader.ListByStates(context.Background(), []string{" ", "\t"})
	if err != nil {
		t.Fatalf("ListByStates empty error = %v, want nil", err)
	}
	if len(empty) != 0 {
		t.Fatalf("ListByStates empty len = %d, want 0", len(empty))
	}
	if len(client.calls) != 0 {
		t.Fatalf("ListByStates empty made %d calls, want 0", len(client.calls))
	}

	got, err := reader.ListByStates(context.Background(), []string{" Done ", "Done", ""})
	if err != nil {
		t.Fatalf("ListByStates error = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListByStates len = %d, want 1", len(got))
	}
	if got[0].Routable != nil {
		t.Fatalf("ListByStates Routable = %v, want nil because cleanup reads do not route", got[0].Routable)
	}
	if len(client.calls) != 1 {
		t.Fatalf("ListByStates call count = %d, want 1 without viewer lookup", len(client.calls))
	}
	if !contains(client.calls[0].query, "SymphonyLinearByStates") {
		t.Fatalf("state query = %q, want SymphonyLinearByStates", client.calls[0].query)
	}
	assertVariable(t, client.calls[0].variables, "projectSlug", "project")
	assertVariable(t, client.calls[0].variables, "stateNames", []string{"Done"})
}

func TestRefreshByIDsBatchesAndRestoresRequestOrder(t *testing.T) {
	t.Parallel()

	ids := make([]string, 0, 51)
	for i := 1; i <= 51; i++ {
		ids = append(ids, fmt.Sprintf("issue-%d", i))
	}

	firstBatchNodes := []map[string]any{
		linearIssue("issue-50", "MT-50", "In Progress"),
		linearIssue("issue-1", "MT-1", "In Progress"),
	}
	secondBatchNodes := []map[string]any{
		linearIssue("issue-51", "MT-51", "In Progress"),
	}
	client := &fakeClient{responses: []fakeResponse{
		{body: issuesResponse(firstBatchNodes)},
		{body: issuesResponse(secondBatchNodes)},
	}}
	reader := mustReader(t, config.ProviderSettings{
		APIKey:  "token",
		Project: "project",
	}, client)

	got, err := reader.RefreshByIDs(context.Background(), ids)
	if err != nil {
		t.Fatalf("RefreshByIDs error = %v, want nil", err)
	}
	if identifiers := identifiers(got); !slices.Equal(identifiers, []string{"MT-1", "MT-50", "MT-51"}) {
		t.Fatalf("refresh identifiers = %#v, want request-visible order", identifiers)
	}
	if len(client.calls) != 2 {
		t.Fatalf("GraphQL call count = %d, want 2", len(client.calls))
	}
	if got := client.calls[0].variables["ids"].([]string); len(got) != 50 {
		t.Fatalf("first batch len = %d, want 50", len(got))
	}
	if got := client.calls[1].variables["ids"].([]string); !slices.Equal(got, []string{"issue-51"}) {
		t.Fatalf("second batch ids = %#v, want issue-51", got)
	}
}

func TestRefreshByIDsEmptyInputDoesNotCallClient(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	reader := mustReader(t, config.ProviderSettings{APIKey: "token", Project: "project", Assignee: "me"}, client)

	got, err := reader.RefreshByIDs(context.Background(), []string{" ", ""})
	if err != nil {
		t.Fatalf("RefreshByIDs empty error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("RefreshByIDs empty len = %d, want 0", len(got))
	}
	if len(client.calls) != 0 {
		t.Fatalf("RefreshByIDs empty made %d calls, want 0", len(client.calls))
	}
}

func TestRoutingExactMatchMismatchAndMe(t *testing.T) {
	t.Parallel()

	t.Run("exact match and mismatch", func(t *testing.T) {
		t.Parallel()

		client := &fakeClient{responses: []fakeResponse{
			{body: issuesResponse([]map[string]any{
				linearIssue("issue-1", "MT-1", "Todo", withAssignee("user-1")),
				linearIssue("issue-2", "MT-2", "Todo", withAssignee("user-2")),
				linearIssue("issue-3", "MT-3", "Todo"),
			})},
		}}
		reader := mustReader(t, config.ProviderSettings{APIKey: "token", Project: "project", Assignee: "user-1"}, client)

		got, err := reader.RefreshByIDs(context.Background(), []string{"issue-1", "issue-2", "issue-3"})
		if err != nil {
			t.Fatalf("RefreshByIDs error = %v, want nil", err)
		}
		assertRoutable(t, got[0], true)
		assertRoutable(t, got[1], false)
		assertRoutable(t, got[2], false)
	})

	t.Run("me resolves viewer", func(t *testing.T) {
		t.Parallel()

		client := &fakeClient{responses: []fakeResponse{
			{body: map[string]any{"data": map[string]any{"viewer": map[string]any{"id": "viewer-1"}}}},
			{body: issuesResponse([]map[string]any{
				linearIssue("issue-1", "MT-1", "Todo", withAssignee("viewer-1")),
				linearIssue("issue-2", "MT-2", "Todo", withAssignee("other")),
			})},
		}}
		reader := mustReader(t, config.ProviderSettings{APIKey: "token", Project: "project", ActiveStates: []string{"Todo"}, Assignee: "me"}, client)

		got, err := reader.RefreshByIDs(context.Background(), []string{"issue-1", "issue-2"})
		if err != nil {
			t.Fatalf("RefreshByIDs error = %v, want nil", err)
		}
		if !contains(client.calls[0].query, "SymphonyLinearViewer") {
			t.Fatalf("first query = %q, want viewer lookup", client.calls[0].query)
		}
		assertRoutable(t, got[0], true)
		assertRoutable(t, got[1], false)
	})

	t.Run("me missing viewer identity", func(t *testing.T) {
		t.Parallel()

		client := &fakeClient{responses: []fakeResponse{
			{body: map[string]any{"data": map[string]any{"viewer": map[string]any{}}}},
		}}
		reader := mustReader(t, config.ProviderSettings{APIKey: "token", Project: "project", ActiveStates: []string{"Todo"}, Assignee: "me"}, client)

		_, err := reader.ListCandidates(context.Background())
		if !errors.Is(err, ErrMissingViewerIdentity) {
			t.Fatalf("ListCandidates error = %v, want %v", err, ErrMissingViewerIdentity)
		}
	})
}

func TestLinearErrorClassificationAndContextCancellation(t *testing.T) {
	t.Parallel()

	t.Run("graphql errors", func(t *testing.T) {
		t.Parallel()

		client := &fakeClient{responses: []fakeResponse{
			{body: map[string]any{"errors": []any{map[string]any{"message": "bad input"}}}},
		}}
		reader := mustReader(t, config.ProviderSettings{APIKey: "token", Project: "project", ActiveStates: []string{"Todo"}}, client)

		_, err := reader.ListCandidates(context.Background())
		var graphErr *GraphQLErrorsError
		if !errors.As(err, &graphErr) {
			t.Fatalf("ListCandidates error = %T %[1]v, want GraphQLErrorsError", err)
		}
	})

	t.Run("unknown payload", func(t *testing.T) {
		t.Parallel()

		client := &fakeClient{responses: []fakeResponse{{body: map[string]any{"data": map[string]any{}}}}}
		reader := mustReader(t, config.ProviderSettings{APIKey: "token", Project: "project", ActiveStates: []string{"Todo"}}, client)

		_, err := reader.ListCandidates(context.Background())
		if !errors.Is(err, ErrUnknownPayload) {
			t.Fatalf("ListCandidates error = %v, want %v", err, ErrUnknownPayload)
		}
	})

	t.Run("missing cursor", func(t *testing.T) {
		t.Parallel()

		client := &fakeClient{responses: []fakeResponse{
			{body: issuesPage([]map[string]any{linearIssue("issue-1", "MT-1", "Todo")}, true, "")},
		}}
		reader := mustReader(t, config.ProviderSettings{APIKey: "token", Project: "project", ActiveStates: []string{"Todo"}}, client)

		_, err := reader.ListCandidates(context.Background())
		if !errors.Is(err, ErrMissingEndCursor) {
			t.Fatalf("ListCandidates error = %v, want %v", err, ErrMissingEndCursor)
		}
	})

	t.Run("transport and status errors stay classifiable", func(t *testing.T) {
		t.Parallel()

		transportCause := errors.New("timeout")
		client := &fakeClient{responses: []fakeResponse{{err: &RequestError{Err: transportCause}}}}
		reader := mustReader(t, config.ProviderSettings{APIKey: "token", Project: "project", ActiveStates: []string{"Todo"}}, client)
		_, err := reader.ListCandidates(context.Background())
		var requestErr *RequestError
		if !errors.As(err, &requestErr) || !errors.Is(err, transportCause) {
			t.Fatalf("transport error = %T %[1]v, want RequestError wrapping cause", err)
		}

		client = &fakeClient{responses: []fakeResponse{{err: &StatusError{Status: 503, Body: "unavailable"}}}}
		reader = mustReader(t, config.ProviderSettings{APIKey: "token", Project: "project", ActiveStates: []string{"Todo"}}, client)
		_, err = reader.ListCandidates(context.Background())
		var statusErr *StatusError
		if !errors.As(err, &statusErr) || statusErr.Status != 503 {
			t.Fatalf("status error = %T %[1]v, want 503 StatusError", err)
		}
	})

	t.Run("context cancellation propagates", func(t *testing.T) {
		t.Parallel()

		client := &fakeClient{responses: []fakeResponse{{err: context.Canceled}}}
		reader := mustReader(t, config.ProviderSettings{APIKey: "token", Project: "project", ActiveStates: []string{"Todo"}}, client)

		_, err := reader.ListCandidates(context.Background())
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ListCandidates error = %v, want context.Canceled", err)
		}
	})
}

func TestHTTPClientErrorClassification(t *testing.T) {
	t.Parallel()

	t.Run("body read cancellation propagates as context error", func(t *testing.T) {
		t.Parallel()

		client := &HTTPClient{
			Endpoint: "https://linear.test/graphql",
			APIKey:   "token",
			Client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       errReadCloser{err: context.Canceled},
				}, nil
			})},
		}

		_, err := client.GraphQL(context.Background(), "query Test { viewer { id } }", map[string]any{})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("GraphQL body read error = %T %[1]v, want context.Canceled", err)
		}
		var requestErr *RequestError
		if errors.As(err, &requestErr) {
			t.Fatalf("GraphQL body read error = %T %[1]v, want raw context error", err)
		}
	})

	t.Run("invalid json is payload classified", func(t *testing.T) {
		t.Parallel()

		client := &HTTPClient{
			Endpoint: "https://linear.test/graphql",
			APIKey:   "token",
			Client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("{")),
				}, nil
			})},
		}

		_, err := client.GraphQL(context.Background(), "query Test { viewer { id } }", map[string]any{})
		if !errors.Is(err, ErrUnknownPayload) {
			t.Fatalf("GraphQL decode error = %T %[1]v, want %v", err, ErrUnknownPayload)
		}
	})
}

func mustReader(t *testing.T, settings config.ProviderSettings, client Client) *Reader {
	t.Helper()
	reader, err := NewReader(settings, client)
	if err != nil {
		t.Fatalf("NewReader error = %v, want nil", err)
	}
	return reader
}

type fakeClient struct {
	responses []fakeResponse
	calls     []graphQLCall
}

type fakeResponse struct {
	body map[string]any
	err  error
}

type graphQLCall struct {
	query     string
	variables map[string]any
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type errReadCloser struct {
	err error
}

func (r errReadCloser) Read([]byte) (int, error) {
	return 0, r.err
}

func (r errReadCloser) Close() error {
	return nil
}

func (c *fakeClient) GraphQL(ctx context.Context, query string, variables map[string]any) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.calls = append(c.calls, graphQLCall{query: query, variables: cloneVariables(variables)})
	if len(c.responses) == 0 {
		return nil, fmt.Errorf("unexpected GraphQL call %q with variables %#v", query, variables)
	}
	response := c.responses[0]
	c.responses = c.responses[1:]
	if response.err != nil {
		return nil, response.err
	}
	return response.body, nil
}

func cloneVariables(variables map[string]any) map[string]any {
	cloned := make(map[string]any, len(variables))
	for key, value := range variables {
		switch typed := value.(type) {
		case []string:
			cloned[key] = append([]string(nil), typed...)
		default:
			cloned[key] = typed
		}
	}
	return cloned
}

func issuesPage(nodes []map[string]any, hasNext bool, endCursor string) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issues": map[string]any{
				"nodes": nodes,
				"pageInfo": map[string]any{
					"hasNextPage": hasNext,
					"endCursor":   endCursor,
				},
			},
		},
	}
}

func issuesResponse(nodes []map[string]any) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"issues": map[string]any{
				"nodes": nodes,
			},
		},
	}
}

type issueOption func(map[string]any)

func linearIssue(id, identifier, state string, opts ...issueOption) map[string]any {
	issue := map[string]any{
		"id":          id,
		"identifier":  identifier,
		"title":       "Title " + identifier,
		"description": "Description " + identifier,
		"state":       map[string]any{"name": state},
		"branchName":  lower(identifier) + "-branch",
		"url":         "https://linear.test/" + identifier,
		"labels":      map[string]any{"nodes": []any{}},
		"inverseRelations": map[string]any{
			"nodes": []any{},
		},
	}
	for _, opt := range opts {
		opt(issue)
	}
	return issue
}

func withPriority(priority int) issueOption {
	return func(issue map[string]any) { issue["priority"] = priority }
}

func withAssignee(id string) issueOption {
	return func(issue map[string]any) {
		issue["assignee"] = map[string]any{"id": id}
	}
}

func withLabels(labels ...string) issueOption {
	return func(issue map[string]any) {
		nodes := make([]any, 0, len(labels))
		for _, label := range labels {
			nodes = append(nodes, map[string]any{"name": label})
		}
		issue["labels"] = map[string]any{"nodes": nodes}
	}
}

func withBlocker(id, identifier, state string) issueOption {
	return func(issue map[string]any) {
		issue["inverseRelations"] = map[string]any{
			"nodes": []any{
				map[string]any{
					"type": " blocks ",
					"issue": map[string]any{
						"id":         id,
						"identifier": identifier,
						"state":      map[string]any{"name": state},
					},
				},
				map[string]any{
					"type":  "relatesTo",
					"issue": map[string]any{"id": "unrelated"},
				},
			},
		}
	}
}

func withCreatedAt(value string) issueOption {
	return func(issue map[string]any) { issue["createdAt"] = value }
}

func withUpdatedAt(value string) issueOption {
	return func(issue map[string]any) { issue["updatedAt"] = value }
}

func assertVariable(t *testing.T, variables map[string]any, key string, want any) {
	t.Helper()
	if got := variables[key]; !reflect.DeepEqual(got, want) {
		t.Fatalf("variable %s = %#v, want %#v", key, got, want)
	}
}

func identifiers(items []domain.WorkItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.Identifier)
	}
	return ids
}

func assertRoutable(t *testing.T, item domain.WorkItem, want bool) {
	t.Helper()
	if item.Routable == nil || *item.Routable != want {
		t.Fatalf("Routable = %v, want %v", item.Routable, want)
	}
}

func assertTime(t *testing.T, got *time.Time, want time.Time) {
	t.Helper()
	if got == nil || !got.Equal(want) {
		t.Fatalf("time = %v, want %v", got, want)
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

func lower(value string) string {
	return strings.ToLower(value)
}
