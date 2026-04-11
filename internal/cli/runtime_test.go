package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Miss-you/go-symphony/internal/codex"
	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/trackers/memory"
)

func TestMemoryRuntimeRunsWithoutLinearToolsAndSchedulesContinuationRetry(t *testing.T) {
	root := t.TempDir()
	store := newTestStore(t, root, "memory", "")
	reader := memory.NewReader([]domain.WorkItem{testWorkItem("item-1", "MT-1", "In Progress")})
	transports := newRuntimeTransportFactory(newRuntimeTransport(
		`{"id":1,"result":{}}`,
		`{"id":2,"result":{"thread":{"id":"thread-memory"}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-1"}}}`,
		`{"method":"turn/completed","params":{"usage":{"input_tokens":5,"output_tokens":7,"total_tokens":12}}}`,
	))

	rt, err := StartRuntime(context.Background(), RuntimeOptions{
		Store:            store,
		Reader:           reader,
		TransportFactory: transports.Start,
	})
	if err != nil {
		t.Fatalf("StartRuntime returned error: %v", err)
	}
	defer func() { _ = rt.Close() }()

	waitFor(t, time.Second, func() bool {
		return len(rt.Snapshot().Retrying) == 1
	})

	writes := transports.transport(0).writes()
	threadStart := findWriteByMethod(t, writes, "thread/start")
	params := threadStart["params"].(map[string]any)
	if tools, ok := params["dynamicTools"].([]any); ok && len(tools) != 0 {
		t.Fatalf("memory dynamicTools = %#v, want none", tools)
	}
	if strings.Contains(transports.joinedWrites(), "linear_graphql") {
		t.Fatalf("memory runtime advertised linear_graphql: %s", transports.joinedWrites())
	}
	if len(transports.usedTransports()) != 1 {
		t.Fatalf("transport sessions = %d, want 1 before continuation retry fires", len(transports.usedTransports()))
	}

	snapshot := rt.Snapshot()
	if got := snapshot.Retrying[0].Attempt; got != 1 {
		t.Fatalf("continuation retry attempt = %d, want 1", got)
	}
	if got := snapshot.CodexTotals.TotalTokens; got != 12 {
		t.Fatalf("codex total tokens = %d, want 12", got)
	}
	if _, err := os.Stat(filepath.Join(root, "workspaces", "MT-1")); err != nil {
		t.Fatalf("workspace not created: %v", err)
	}
}

func TestLinearRuntimeAdvertisesWorkflowSelectedTool(t *testing.T) {
	root := t.TempDir()
	store := newTestStore(t, root, "linear", "api_key: test-token\n  project_slug: TEST")
	reader := memory.NewReader([]domain.WorkItem{testWorkItem("item-2", "MT-2", "In Progress")})
	transports := newRuntimeTransportFactory(newRuntimeTransport(
		`{"id":1,"result":{}}`,
		`{"id":2,"result":{"thread":{"id":"thread-linear"}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-1"}}}`,
		`{"method":"turn/completed","params":{"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}}}`,
	))

	rt, err := StartRuntime(context.Background(), RuntimeOptions{
		Store:            store,
		Reader:           reader,
		TransportFactory: transports.Start,
	})
	if err != nil {
		t.Fatalf("StartRuntime returned error: %v", err)
	}
	defer func() { _ = rt.Close() }()

	waitFor(t, time.Second, func() bool {
		transport := transports.transport(0)
		return transport != nil && findWriteByMethodOrNil(transport.writes(), "thread/start") != nil
	})

	threadStart := findWriteByMethod(t, transports.transport(0).writes(), "thread/start")
	params := threadStart["params"].(map[string]any)
	tools, ok := params["dynamicTools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("dynamicTools = %#v, want one linear tool", params["dynamicTools"])
	}
	tool := tools[0].(map[string]any)
	if tool["name"] != "linear_graphql" {
		t.Fatalf("tool name = %#v, want linear_graphql", tool["name"])
	}
}

func TestStartupCleanupRunsBeforeDispatch(t *testing.T) {
	root := t.TempDir()
	store := newTestStore(t, root, "memory", "")
	terminalPath := filepath.Join(root, "workspaces", "MT-DONE")
	if err := os.MkdirAll(terminalPath, 0o755); err != nil {
		t.Fatalf("mkdir terminal workspace: %v", err)
	}
	reader := memory.NewReader([]domain.WorkItem{
		testWorkItem("done", "MT-DONE", "Done"),
		testWorkItem("active", "MT-ACTIVE", "In Progress"),
	})
	transports := newRuntimeTransportFactory(newRuntimeTransport(
		`{"id":1,"result":{}}`,
		`{"id":2,"result":{"thread":{"id":"thread-active"}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-1"}}}`,
		`{"method":"turn/completed","params":{}}`,
	))

	rt, err := StartRuntime(context.Background(), RuntimeOptions{
		Store:            store,
		Reader:           reader,
		TransportFactory: transports.Start,
	})
	if err != nil {
		t.Fatalf("StartRuntime returned error: %v", err)
	}
	defer func() { _ = rt.Close() }()

	waitFor(t, time.Second, func() bool {
		transport := transports.transport(0)
		return transport != nil && findWriteByMethodOrNil(transport.writes(), "thread/start") != nil
	})
	if _, err := os.Stat(terminalPath); !os.IsNotExist(err) {
		t.Fatalf("terminal workspace still exists or unexpected stat err: %v", err)
	}
}

func TestWorkerRefreshesAfterTurnsAndMaxTurnsExitIsNormalCompletion(t *testing.T) {
	root := t.TempDir()
	store := newTestStoreWithMaxTurns(t, root, "memory", "", 2)
	reader := &recordingReader{items: []domain.WorkItem{testWorkItem("item-3", "MT-3", "In Progress")}}
	transports := newRuntimeTransportFactory(newRuntimeTransport(
		`{"id":1,"result":{}}`,
		`{"id":2,"result":{"thread":{"id":"thread-refresh"}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-1"}}}`,
		`{"method":"turn/completed","params":{"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-2"}}}`,
		`{"method":"turn/completed","params":{"usage":{"input_tokens":2,"output_tokens":2,"total_tokens":4}}}`,
	))

	rt, err := StartRuntime(context.Background(), RuntimeOptions{
		Store:            store,
		Reader:           reader,
		TransportFactory: transports.Start,
	})
	if err != nil {
		t.Fatalf("StartRuntime returned error: %v", err)
	}
	defer func() { _ = rt.Close() }()

	waitFor(t, time.Second, func() bool {
		return len(rt.Snapshot().Retrying) == 1
	})

	if got := reader.refreshCount(); got != 3 {
		t.Fatalf("refresh count = %d, want dispatch revalidation plus one refresh after each completed turn", got)
	}
	writes := transports.transport(0).writes()
	turnStarts := writesByMethod(writes, "turn/start")
	if len(turnStarts) != 2 {
		t.Fatalf("turn/start count = %d, want 2", len(turnStarts))
	}
	secondParams := turnStarts[1]["params"].(map[string]any)
	input := secondParams["input"].([]any)
	text := input[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "Continuation guidance") {
		t.Fatalf("second turn prompt = %q, want continuation guidance", text)
	}
	retry := rt.Snapshot().Retrying[0]
	if retry.Attempt != 1 || retry.LastError != "" {
		t.Fatalf("retry after max turns = %+v, want normal continuation attempt", retry)
	}
}

func TestTurnCancelledNormalizesAsFailureRetry(t *testing.T) {
	root := t.TempDir()
	store := newTestStore(t, root, "memory", "")
	reader := memory.NewReader([]domain.WorkItem{testWorkItem("item-4", "MT-4", "In Progress")})
	transports := newRuntimeTransportFactory(newRuntimeTransport(
		`{"id":1,"result":{}}`,
		`{"id":2,"result":{"thread":{"id":"thread-cancel"}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-1"}}}`,
		`{"method":"turn/cancelled","params":{"reason":"test"}}`,
	))

	rt, err := StartRuntime(context.Background(), RuntimeOptions{
		Store:            store,
		Reader:           reader,
		TransportFactory: transports.Start,
	})
	if err != nil {
		t.Fatalf("StartRuntime returned error: %v", err)
	}
	defer func() { _ = rt.Close() }()

	waitFor(t, time.Second, func() bool {
		snapshot := rt.Snapshot()
		return len(snapshot.Retrying) == 1 && strings.Contains(snapshot.Retrying[0].LastError, "cancelled")
	})

	retry := rt.Snapshot().Retrying[0]
	if retry.Attempt != 1 {
		t.Fatalf("failure retry attempt = %d, want 1", retry.Attempt)
	}
}

func TestRenderPromptHandlesDefaultDescriptionConditional(t *testing.T) {
	template := config.EffectivePromptTemplate(config.Workflow{})
	withDescription := renderPrompt(template, testWorkItem("item-5", "MT-5", "In Progress"))
	if strings.Contains(withDescription, "{%") || strings.Contains(withDescription, "%}") {
		t.Fatalf("prompt leaked template markers: %q", withDescription)
	}
	if !strings.Contains(withDescription, "Body MT-5") {
		t.Fatalf("prompt = %q, want issue description", withDescription)
	}
	if strings.Contains(withDescription, "No description provided.") {
		t.Fatalf("prompt = %q, want description branch without fallback", withDescription)
	}

	item := testWorkItem("item-6", "MT-6", "In Progress")
	item.Description = ""
	withoutDescription := renderPrompt(template, item)
	if strings.Contains(withoutDescription, "{%") || strings.Contains(withoutDescription, "%}") {
		t.Fatalf("prompt leaked template markers: %q", withoutDescription)
	}
	if !strings.Contains(withoutDescription, "No description provided.") {
		t.Fatalf("prompt = %q, want fallback description branch", withoutDescription)
	}
}

func TestRuntimeUsesStartupWorkflowSnapshotForWorkerPrompts(t *testing.T) {
	root := t.TempDir()
	store, workflowPath := newTestStoreWithPrompt(t, root, "memory", "", 1, "Initial {{ issue.identifier }}")
	reader := newBlockingReader([]domain.WorkItem{testWorkItem("item-7", "MT-7", "In Progress")})
	transports := newRuntimeTransportFactory(newRuntimeTransport(
		`{"id":1,"result":{}}`,
		`{"id":2,"result":{"thread":{"id":"thread-snapshot"}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-1"}}}`,
		`{"method":"turn/completed","params":{}}`,
	))

	rt, err := StartRuntime(context.Background(), RuntimeOptions{
		Store:            store,
		Reader:           reader,
		TransportFactory: transports.Start,
	})
	if err != nil {
		t.Fatalf("StartRuntime returned error: %v", err)
	}
	defer func() { _ = rt.Close() }()

	if err := os.WriteFile(workflowPath, []byte(testWorkflowContent(root, "memory", "", 1, "Mutated {{ issue.identifier }}")), 0o644); err != nil {
		t.Fatalf("rewrite workflow: %v", err)
	}
	reader.releaseCandidates()

	waitFor(t, time.Second, func() bool {
		return len(rt.Snapshot().Retrying) == 1
	})

	turnStart := findWriteByMethod(t, transports.transport(0).writes(), "turn/start")
	params := turnStart["params"].(map[string]any)
	input := params["input"].([]any)
	text := input[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "Initial MT-7") {
		t.Fatalf("turn prompt = %q, want startup snapshot prompt", text)
	}
	if strings.Contains(text, "Mutated MT-7") {
		t.Fatalf("turn prompt = %q, want no mutated prompt", text)
	}
}

func newTestStore(t *testing.T, root, provider, providerExtra string) *config.Store {
	return newTestStoreWithMaxTurns(t, root, provider, providerExtra, 1)
}

func newTestStoreWithMaxTurns(t *testing.T, root, provider, providerExtra string, maxTurns int) *config.Store {
	t.Helper()
	store, _ := newTestStoreWithPrompt(t, root, provider, providerExtra, maxTurns, "Work on {{ issue.identifier }}: {{ issue.title }}")
	return store
}

func newTestStoreWithPrompt(t *testing.T, root, provider, providerExtra string, maxTurns int, prompt string) (*config.Store, string) {
	t.Helper()
	workflowPath := filepath.Join(root, "WORKFLOW.md")
	if err := os.WriteFile(workflowPath, []byte(testWorkflowContent(root, provider, providerExtra, maxTurns, prompt)), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	store, err := config.NewStore(config.WithWorkflowPath(workflowPath))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, workflowPath
}

func testWorkflowContent(root, provider, providerExtra string, maxTurns int, prompt string) string {
	return `---
tracker:
  kind: ` + provider + `
  ` + providerExtra + `
  active_states: ["Todo", "In Progress"]
  terminal_states: ["Done", "Canceled"]
workspace:
  root: ` + filepath.ToSlash(filepath.Join(root, "workspaces")) + `
agent:
  max_concurrent_agents: 1
  max_turns: ` + fmt.Sprint(maxTurns) + `
  max_retry_backoff_ms: 300000
codex:
  command: codex app-server
  approval_policy: never
  thread_sandbox: workspace-write
  turn_timeout_ms: 1000
  read_timeout_ms: 1000
  stall_timeout_ms: 0
hooks:
  timeout_ms: 1000
observability:
  refresh_ms: 1000
  render_interval_ms: 16
---
` + prompt + `
`
}

func testWorkItem(id, identifier, state string) domain.WorkItem {
	return domain.WorkItem{
		ID:          id,
		Identifier:  identifier,
		Title:       "Test " + identifier,
		Description: "Body " + identifier,
		State:       state,
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

type runtimeTransportFactory struct {
	mu    sync.Mutex
	queue []*runtimeTransport
	used  []*runtimeTransport
}

func newRuntimeTransportFactory(transports ...*runtimeTransport) *runtimeTransportFactory {
	return &runtimeTransportFactory{queue: transports}
}

func (f *runtimeTransportFactory) Start(context.Context, codex.TransportRequest) (codex.Transport, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.queue) == 0 {
		transport := newRuntimeTransport()
		f.used = append(f.used, transport)
		return transport, nil
	}
	transport := f.queue[0]
	f.queue = f.queue[1:]
	f.used = append(f.used, transport)
	return transport, nil
}

func (f *runtimeTransportFactory) transport(index int) *runtimeTransport {
	f.mu.Lock()
	defer f.mu.Unlock()
	if index >= len(f.used) {
		return nil
	}
	return f.used[index]
}

func (f *runtimeTransportFactory) joinedWrites() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var all []string
	for _, transport := range f.used {
		all = append(all, transport.joinedWrites())
	}
	return strings.Join(all, "\n")
}

func (f *runtimeTransportFactory) usedTransports() []*runtimeTransport {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*runtimeTransport(nil), f.used...)
}

type runtimeTransport struct {
	mu      sync.Mutex
	lines   [][]byte
	written []map[string]any
	closed  bool
}

func newRuntimeTransport(lines ...string) *runtimeTransport {
	transport := &runtimeTransport{}
	for _, line := range lines {
		transport.lines = append(transport.lines, []byte(line))
	}
	return transport
}

func (t *runtimeTransport) ReadLine(ctx context.Context) ([]byte, error) {
	for {
		t.mu.Lock()
		if len(t.lines) > 0 {
			line := t.lines[0]
			t.lines = t.lines[1:]
			t.mu.Unlock()
			return line, nil
		}
		if t.closed {
			t.mu.Unlock()
			return nil, context.Canceled
		}
		t.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Millisecond):
		}
	}
}

func (t *runtimeTransport) WriteJSON(_ context.Context, payload map[string]any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.written = append(t.written, decoded)
	return nil
}

func (t *runtimeTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	return nil
}

func (t *runtimeTransport) writesCopy() []map[string]any {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]map[string]any(nil), t.written...)
}

func (t *runtimeTransport) writes() []map[string]any { return t.writesCopy() }

func (t *runtimeTransport) joinedWrites() string {
	writes := t.writesCopy()
	var encoded []string
	for _, write := range writes {
		content, _ := json.Marshal(write)
		encoded = append(encoded, string(content))
	}
	return strings.Join(encoded, "\n")
}

func findWriteByMethod(t *testing.T, writes []map[string]any, method string) map[string]any {
	t.Helper()
	if write := findWriteByMethodOrNil(writes, method); write != nil {
		return write
	}
	t.Fatalf("method %q not found in writes %#v", method, writes)
	return nil
}

func findWriteByMethodOrNil(writes []map[string]any, method string) map[string]any {
	for _, write := range writes {
		if write["method"] == method {
			return write
		}
	}
	return nil
}

func writesByMethod(writes []map[string]any, method string) []map[string]any {
	var matched []map[string]any
	for _, write := range writes {
		if write["method"] == method {
			matched = append(matched, write)
		}
	}
	return matched
}

type recordingReader struct {
	mu      sync.Mutex
	items   []domain.WorkItem
	refresh int
}

func (r *recordingReader) ListCandidates(context.Context) ([]domain.WorkItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.WorkItem(nil), r.items...), nil
}

func (r *recordingReader) ListByStates(_ context.Context, states []string) ([]domain.WorkItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var matched []domain.WorkItem
	for _, item := range r.items {
		for _, state := range states {
			if strings.EqualFold(strings.TrimSpace(item.State), strings.TrimSpace(state)) {
				matched = append(matched, item)
			}
		}
	}
	return matched, nil
}

func (r *recordingReader) RefreshByIDs(_ context.Context, ids []string) ([]domain.WorkItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refresh++
	var matched []domain.WorkItem
	for _, id := range ids {
		for _, item := range r.items {
			if item.ID == id {
				matched = append(matched, item)
			}
		}
	}
	return matched, nil
}

func (r *recordingReader) refreshCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.refresh
}

type blockingReader struct {
	*recordingReader
	release chan struct{}
	once    sync.Once
}

func newBlockingReader(items []domain.WorkItem) *blockingReader {
	return &blockingReader{
		recordingReader: &recordingReader{items: items},
		release:         make(chan struct{}),
	}
}

func (r *blockingReader) releaseCandidates() {
	r.once.Do(func() { close(r.release) })
}

func (r *blockingReader) ListCandidates(ctx context.Context) ([]domain.WorkItem, error) {
	select {
	case <-r.release:
		return r.recordingReader.ListCandidates(ctx)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
