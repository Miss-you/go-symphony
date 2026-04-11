package codex

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Miss-you/go-symphony/internal/config"
)

func TestConfigFromSettingsMapsCodexRuntimeSettings(t *testing.T) {
	t.Parallel()

	settings := config.Settings{
		Workspace: config.WorkspaceSettings{Root: "/tmp/workspaces"},
		Codex: config.CodexSettings{
			Command: "codex app-server",
			ApprovalPolicy: map[string]any{
				"reject": map[string]any{
					"sandbox_approval": true,
					"rules":            true,
				},
			},
			ThreadSandbox:     "workspace-write",
			TurnSandboxPolicy: map[string]any{"mode": "workspace-write"},
			ReadTimeoutMS:     1234,
			TurnTimeoutMS:     5678,
		},
	}

	got := ConfigFromSettings(settings)

	if got.Command != "codex app-server" {
		t.Fatalf("Command = %q, want %q", got.Command, "codex app-server")
	}
	if got.WorkspaceRoot != "/tmp/workspaces" {
		t.Fatalf("WorkspaceRoot = %q, want %q", got.WorkspaceRoot, "/tmp/workspaces")
	}
	wantApproval := map[string]any{
		"reject": map[string]any{
			"sandbox_approval": true,
			"rules":            true,
		},
	}
	if !reflect.DeepEqual(got.ApprovalPolicy, wantApproval) {
		t.Fatalf("ApprovalPolicy = %#v, want %#v", got.ApprovalPolicy, wantApproval)
	}
	if got.ThreadSandbox != "workspace-write" {
		t.Fatalf("ThreadSandbox = %q, want workspace-write", got.ThreadSandbox)
	}
	if got.ReadTimeout != 1234*time.Millisecond {
		t.Fatalf("ReadTimeout = %s, want %s", got.ReadTimeout, 1234*time.Millisecond)
	}
	if got.TurnTimeout != 5678*time.Millisecond {
		t.Fatalf("TurnTimeout = %s, want %s", got.TurnTimeout, 5678*time.Millisecond)
	}
	if !reflect.DeepEqual(got.TurnSandboxPolicy, map[string]any{"mode": "workspace-write"}) {
		t.Fatalf("TurnSandboxPolicy = %#v, want mode policy", got.TurnSandboxPolicy)
	}
}

func TestStartSessionValidatesWorkspaceAndBootstrapsThread(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspacePath := filepath.Join(root, "MT-123")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	transport := newScriptedTransport(
		`{"id":1,"result":{"server":"ready"}}`,
		`{"id":2,"result":{"thread":{"id":"thread-1"}}}`,
	)
	factory := &recordingFactory{transport: transport}

	session, err := StartSession(context.Background(), SessionOptions{
		Config: Config{
			Command:        "codex app-server",
			WorkspaceRoot:  root,
			ReadTimeout:    time.Second,
			ApprovalPolicy: "never",
			ThreadSandbox:  "workspace-write",
			DynamicTools: []ToolSpec{
				{Name: "linear_graphql", Description: "Run Linear GraphQL"},
			},
		},
		WorkspacePath:     workspacePath,
		TransportFactory:  factory.Start,
		ToolHandler:       ToolHandlerFunc(func(context.Context, ToolCall) (ToolResult, error) { return ToolResult{}, ErrUnsupportedTool }),
		NonInteractive:    true,
		TurnSandboxPolicy: map[string]any{"mode": "workspace-write"},
	})
	if err != nil {
		t.Fatalf("StartSession returned error: %v", err)
	}
	defer func() { _ = session.Close() }()

	if session.ThreadID() != "thread-1" {
		t.Fatalf("ThreadID = %q, want thread-1", session.ThreadID())
	}
	if factory.request.Command != "codex app-server" {
		t.Fatalf("factory command = %q, want codex app-server", factory.request.Command)
	}
	resolvedWorkspacePath, err := filepath.EvalSymlinks(workspacePath)
	if err != nil {
		t.Fatalf("EvalSymlinks(workspacePath): %v", err)
	}
	if factory.request.Dir != resolvedWorkspacePath {
		t.Fatalf("factory dir = %q, want %q", factory.request.Dir, resolvedWorkspacePath)
	}

	writes := transport.writes()
	assertMethod(t, writes[0], "initialize")
	assertMethod(t, writes[1], "initialized")
	threadStart := assertMethod(t, writes[2], "thread/start")
	params := mapValue(t, threadStart, "params")
	if params["approvalPolicy"] != "never" {
		t.Fatalf("thread/start approvalPolicy = %#v, want never", params["approvalPolicy"])
	}
	if params["sandbox"] != "workspace-write" {
		t.Fatalf("thread/start sandbox = %#v, want workspace-write", params["sandbox"])
	}
	if params["cwd"] != resolvedWorkspacePath {
		t.Fatalf("thread/start cwd = %#v, want %q", params["cwd"], resolvedWorkspacePath)
	}
	dynamicTools, ok := params["dynamicTools"].([]any)
	if !ok || len(dynamicTools) != 1 {
		t.Fatalf("dynamicTools = %#v, want one advertised tool", params["dynamicTools"])
	}

	if err := session.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if !transport.closed() {
		t.Fatal("transport was not closed")
	}
}

func TestStartSessionRejectsInvalidWorkspacePaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	inside := filepath.Join(root, "inside")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatalf("mkdir inside: %v", err)
	}
	escape := filepath.Join(root, "escape")
	if err := os.Symlink(outside, escape); err != nil {
		t.Fatalf("symlink escape: %v", err)
	}

	tests := []struct {
		name          string
		workspacePath string
		want          WorkspaceErrorKind
	}{
		{name: "root", workspacePath: root, want: ErrWorkspaceEqualsRoot},
		{name: "outside", workspacePath: outside, want: ErrWorkspaceOutsideRoot},
		{name: "symlink_escape", workspacePath: escape, want: ErrWorkspaceSymlinkEscape},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			factory := &recordingFactory{transport: newScriptedTransport()}
			_, err := StartSession(context.Background(), SessionOptions{
				Config: Config{
					Command:       "codex app-server",
					WorkspaceRoot: root,
					ReadTimeout:   time.Second,
				},
				WorkspacePath:    tt.workspacePath,
				TransportFactory: factory.Start,
			})
			if err == nil {
				t.Fatal("StartSession returned nil error, want workspace validation error")
			}
			var workspaceErr *WorkspaceError
			if !errors.As(err, &workspaceErr) {
				t.Fatalf("error type = %T, want *WorkspaceError", err)
			}
			if workspaceErr.Kind != tt.want {
				t.Fatalf("workspace error kind = %q, want %q", workspaceErr.Kind, tt.want)
			}
			if factory.called {
				t.Fatal("transport factory was called for invalid workspace")
			}
		})
	}
}

func TestRunTurnReusesSessionAndEmitsProtocolEvents(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspacePath := filepath.Join(root, "MT-234")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	transport := newScriptedTransport(
		`{"id":1,"result":{}}`,
		`{"id":2,"result":{"thread":{"id":"thread-234"}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-1"}}}`,
		`{"method":"item/progress","params":{"message":"working"}}`,
		`not json`,
		`{"method":"turn/completed","params":{"usage":{"input_tokens":11,"output_tokens":7,"total_tokens":18}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-2"}}}`,
		`{"method":"turn/cancelled","params":{"reason":"operator"}}`,
	)
	var events []Event
	var session *Session
	session = startTestSession(t, root, workspacePath, transport, func(event Event) {
		if session != nil {
			_ = session.ThreadID()
		}
		events = append(events, event)
	})
	defer func() { _ = session.Close() }()

	result, err := session.RunTurn(context.Background(), TurnRequest{
		Prompt: "do the work",
		Title:  "MT-234: Do the work",
	})
	if err != nil {
		t.Fatalf("RunTurn returned error: %v", err)
	}
	if result.Status != TurnCompleted {
		t.Fatalf("turn status = %q, want %q", result.Status, TurnCompleted)
	}
	if result.TurnID != "turn-1" {
		t.Fatalf("TurnID = %q, want turn-1", result.TurnID)
	}

	second, err := session.RunTurn(context.Background(), TurnRequest{
		Prompt: "continue",
		Title:  "MT-234: Do the work",
	})
	if err != nil {
		t.Fatalf("second RunTurn returned error: %v", err)
	}
	if second.Status != TurnCancelled {
		t.Fatalf("second turn status = %q, want %q", second.Status, TurnCancelled)
	}

	writes := transport.writes()
	firstTurn := assertMethod(t, writes[3], "turn/start")
	firstParams := mapValue(t, firstTurn, "params")
	if firstParams["threadId"] != "thread-234" {
		t.Fatalf("first turn threadId = %#v, want thread-234", firstParams["threadId"])
	}
	input := firstParams["input"].([]any)[0].(map[string]any)
	if input["text"] != "do the work" {
		t.Fatalf("first turn input text = %#v, want prompt", input["text"])
	}
	secondTurn := assertMethod(t, writes[4], "turn/start")
	secondParams := mapValue(t, secondTurn, "params")
	if secondParams["threadId"] != "thread-234" {
		t.Fatalf("second turn threadId = %#v, want thread-234", secondParams["threadId"])
	}

	assertEventKinds(t, events,
		EventSessionStarted,
		EventUnknownMessage,
		EventMalformedMessage,
		EventTurnCompleted,
		EventTurnCancelled,
	)
	if result.Usage.InputTokens != 11 || result.Usage.OutputTokens != 7 || result.Usage.TotalTokens != 18 {
		t.Fatalf("usage = %#v, want 11/7/18", result.Usage)
	}
}

func TestRunTurnHandlesApprovalsUserInputAndTools(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspacePath := filepath.Join(root, "MT-345")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	transport := newScriptedTransport(
		`{"id":1,"result":{}}`,
		`{"id":2,"result":{"thread":{"id":"thread-345"}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-tools"}}}`,
		`{"id":"approval-1","method":"item/commandExecution/requestApproval","params":{}}`,
		`{"id":"input-1","method":"item/tool/requestUserInput","params":{"questions":[{"id":"q1","options":[{"label":"Approve this Session"}]}]}}`,
		`{"id":"input-2","method":"item/tool/requestUserInput","params":{"questions":[{"id":"q2"}]}}`,
		`{"id":"tool-1","method":"item/tool/call","params":{"tool":"echo","arguments":{"query":"viewer { id }"}}}`,
		`{"id":"tool-2","method":"item/tool/call","params":{"name":"missing","arguments":{}}}`,
		`{"method":"turn/completed","params":{}}`,
	)
	var events []Event
	session := startTestSessionWithTool(t, root, workspacePath, transport, func(event Event) {
		events = append(events, event)
	}, ToolHandlerFunc(func(_ context.Context, call ToolCall) (ToolResult, error) {
		if call.Name != "echo" {
			return ToolResult{}, ErrUnsupportedTool
		}
		return ToolResult{Success: true, Result: call.Arguments}, nil
	}))
	defer func() { _ = session.Close() }()

	if _, err := session.RunTurn(context.Background(), TurnRequest{Prompt: "use tools", Title: "MT-345"}); err != nil {
		t.Fatalf("RunTurn returned error: %v", err)
	}

	writes := transport.writes()
	approval := responseForID(t, writes, "approval-1")
	if decision := mapValue(t, approval, "result")["decision"]; decision != "acceptForSession" {
		t.Fatalf("approval decision = %#v, want acceptForSession", decision)
	}
	input := responseForID(t, writes, "input-1")
	answers := mapValue(t, input, "result")["answers"].(map[string]any)
	if got := answers["q1"].(map[string]any)["answers"].([]any)[0]; got != "Approve this Session" {
		t.Fatalf("q1 answer = %#v, want Approve this Session", got)
	}
	fallbackInput := responseForID(t, writes, "input-2")
	fallbackAnswers := mapValue(t, fallbackInput, "result")["answers"].(map[string]any)
	if got := fallbackAnswers["q2"].(map[string]any)["answers"].([]any)[0]; got != NonInteractiveInputAnswer {
		t.Fatalf("q2 answer = %#v, want non-interactive fallback", got)
	}
	toolOK := responseForID(t, writes, "tool-1")
	if success := mapValue(t, toolOK, "result")["success"]; success != true {
		t.Fatalf("tool-1 success = %#v, want true", success)
	}
	toolMissing := responseForID(t, writes, "tool-2")
	missingResult := mapValue(t, toolMissing, "result")
	if success := missingResult["success"]; success != false {
		t.Fatalf("tool-2 success = %#v, want false", success)
	}
	if missingResult["error"] != "unsupported_tool_call" {
		t.Fatalf("tool-2 error = %#v, want unsupported_tool_call", missingResult["error"])
	}

	assertEventKinds(t, events,
		EventSessionStarted,
		EventApprovalAnswered,
		EventToolInputAnswered,
		EventToolInputAnswered,
		EventToolCallCompleted,
		EventUnsupportedToolCall,
		EventTurnCompleted,
	)
}

func TestRunTurnPreservesRawStringToolArgumentsAndContentItems(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspacePath := filepath.Join(root, "MT-347")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	rawQuery := "  query Viewer { viewer { id } }  "
	transport := newScriptedTransport(
		`{"id":1,"result":{}}`,
		`{"id":2,"result":{"thread":{"id":"thread-347"}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-tools"}}}`,
		`{"id":"tool-raw","method":"item/tool/call","params":{"tool":"linear_graphql","arguments":"`+rawQuery+`"}}`,
		`{"method":"turn/completed","params":{}}`,
	)
	var gotArgs any
	session := startTestSessionWithTool(t, root, workspacePath, transport, nil, ToolHandlerFunc(func(_ context.Context, call ToolCall) (ToolResult, error) {
		gotArgs = call.Arguments
		return ToolResult{
			Success: true,
			ContentItems: []ToolContentItem{
				{Type: "inputText", Text: `{"data":{"viewer":{"id":"usr_123"}}}`},
			},
		}, nil
	}))
	defer func() { _ = session.Close() }()

	if _, err := session.RunTurn(context.Background(), TurnRequest{Prompt: "use raw tool", Title: "MT-347"}); err != nil {
		t.Fatalf("RunTurn returned error: %v", err)
	}
	if gotArgs != rawQuery {
		t.Fatalf("tool arguments = %#v, want raw string %q", gotArgs, rawQuery)
	}

	toolResponse := responseForID(t, transport.writes(), "tool-raw")
	result := mapValue(t, toolResponse, "result")
	items, ok := result["contentItems"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("result.contentItems = %#v, want one content item", result["contentItems"])
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("content item = %#v, want map", items[0])
	}
	if item["type"] != "inputText" || item["text"] == "" {
		t.Fatalf("content item = %#v, want inputText with text", item)
	}
	if nested, ok := result["result"].(map[string]any); ok {
		if _, exists := nested["contentItems"]; exists {
			t.Fatalf("contentItems nested under result.result: %#v", result)
		}
	}
}

func TestRunTurnRequiresNonInteractiveForUserInput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspacePath := filepath.Join(root, "MT-346")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	transport := newScriptedTransport(
		`{"id":1,"result":{}}`,
		`{"id":2,"result":{"thread":{"id":"thread-346"}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-input"}}}`,
		`{"id":"input-1","method":"item/tool/requestUserInput","params":{"questions":[{"id":"q1"}]}}`,
	)
	session, err := StartSession(context.Background(), SessionOptions{
		Config: Config{
			Command:        "codex app-server",
			WorkspaceRoot:  root,
			ReadTimeout:    time.Second,
			TurnTimeout:    time.Second,
			ApprovalPolicy: "never",
		},
		WorkspacePath:     workspacePath,
		TransportFactory:  (&recordingFactory{transport: transport}).Start,
		ToolHandler:       ToolHandlerFunc(func(context.Context, ToolCall) (ToolResult, error) { return ToolResult{}, ErrUnsupportedTool }),
		NonInteractive:    false,
		TurnSandboxPolicy: map[string]any{},
	})
	if err != nil {
		t.Fatalf("StartSession returned error: %v", err)
	}
	defer func() { _ = session.Close() }()

	_, err = session.RunTurn(context.Background(), TurnRequest{Prompt: "ask", Title: "MT-346"})
	if !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("RunTurn error = %v, want ErrApprovalRequired", err)
	}
	for _, write := range transport.writes() {
		if write["id"] == "input-1" {
			t.Fatalf("unexpected non-interactive response when disabled: %#v", write)
		}
	}
}

func TestTimeoutsAreClassified(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspacePath := filepath.Join(root, "MT-456")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	_, err := StartSession(context.Background(), SessionOptions{
		Config: Config{
			Command:        "codex app-server",
			WorkspaceRoot:  root,
			ReadTimeout:    time.Millisecond,
			ApprovalPolicy: "never",
		},
		WorkspacePath:     workspacePath,
		TransportFactory:  (&recordingFactory{transport: newScriptedTransport()}).Start,
		ToolHandler:       ToolHandlerFunc(func(context.Context, ToolCall) (ToolResult, error) { return ToolResult{}, ErrUnsupportedTool }),
		NonInteractive:    true,
		TurnSandboxPolicy: map[string]any{},
	})
	if !errors.Is(err, ErrReadTimeout) {
		t.Fatalf("StartSession error = %v, want ErrReadTimeout", err)
	}

	transport := newScriptedTransport(
		`{"id":1,"result":{}}`,
		`{"id":2,"result":{"thread":{"id":"thread-timeout"}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-timeout"}}}`,
	)
	session := startTestSessionWithConfig(t, Config{
		Command:        "codex app-server",
		WorkspaceRoot:  root,
		ReadTimeout:    time.Second,
		TurnTimeout:    time.Millisecond,
		ApprovalPolicy: "never",
	}, workspacePath, transport, nil)
	defer func() { _ = session.Close() }()

	_, err = session.RunTurn(context.Background(), TurnRequest{Prompt: "wait", Title: "MT-456"})
	if !errors.Is(err, ErrTurnTimeout) {
		t.Fatalf("RunTurn error = %v, want ErrTurnTimeout", err)
	}

	parentCtx, parentCancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer parentCancel()
	_, err = StartSession(parentCtx, SessionOptions{
		Config: Config{
			Command:        "codex app-server",
			WorkspaceRoot:  root,
			ReadTimeout:    time.Second,
			ApprovalPolicy: "never",
		},
		WorkspacePath:     workspacePath,
		TransportFactory:  (&recordingFactory{transport: newScriptedTransport()}).Start,
		ToolHandler:       ToolHandlerFunc(func(context.Context, ToolCall) (ToolResult, error) { return ToolResult{}, ErrUnsupportedTool }),
		NonInteractive:    true,
		TurnSandboxPolicy: map[string]any{},
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("StartSession error = %v, want parent context deadline", err)
	}
	if errors.Is(err, ErrReadTimeout) {
		t.Fatalf("StartSession error = %v, must not wrap ErrReadTimeout for parent deadline", err)
	}

	parentTransport := newScriptedTransport(
		`{"id":1,"result":{}}`,
		`{"id":2,"result":{"thread":{"id":"thread-parent-timeout"}}}`,
	)
	parentSession := startTestSessionWithConfig(t, Config{
		Command:        "codex app-server",
		WorkspaceRoot:  root,
		ReadTimeout:    time.Second,
		TurnTimeout:    time.Second,
		ApprovalPolicy: "never",
	}, workspacePath, parentTransport, nil)
	defer func() { _ = parentSession.Close() }()
	parentTransport.lines <- []byte(`{"id":3,"result":{"turn":{"id":"turn-parent-timeout"}}}`)

	turnCtx, turnCancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer turnCancel()
	_, err = parentSession.RunTurn(turnCtx, TurnRequest{Prompt: "wait", Title: "MT-456"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RunTurn error = %v, want parent context deadline", err)
	}
	if errors.Is(err, ErrTurnTimeout) {
		t.Fatalf("RunTurn error = %v, must not wrap ErrTurnTimeout for parent deadline", err)
	}
}

func TestProcessTransportDoesNotReadStderrAsProtocol(t *testing.T) {
	t.Parallel()

	transport, err := StartProcessTransport(context.Background(), TransportRequest{
		Command: `printf 'not-json\n' >&2; sleep 0.05; printf '{"id":1,"result":{}}\n'`,
		Dir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("StartProcessTransport returned error: %v", err)
	}
	defer func() { _ = transport.Close() }()

	line, err := transport.ReadLine(context.Background())
	if err != nil {
		t.Fatalf("ReadLine returned error: %v", err)
	}
	if string(line) != `{"id":1,"result":{}}` {
		t.Fatalf("ReadLine = %q, want stdout protocol JSON only", string(line))
	}
}

func TestProcessTransportReadsLongProtocolLine(t *testing.T) {
	t.Parallel()

	longMessage := `{"id":1,"result":{"text":"` + strings.Repeat("x", 70*1024) + `"}}`
	transport := &processTransport{
		reads: make(chan readResult, 2),
	}
	go transport.scan(strings.NewReader(longMessage + "\n"))

	line, err := transport.ReadLine(context.Background())
	if err != nil {
		t.Fatalf("ReadLine returned error: %v", err)
	}
	if string(line) != longMessage {
		t.Fatalf("long line length = %d, want %d", len(line), len(longMessage))
	}
}

func startTestSession(t *testing.T, root, workspacePath string, transport *scriptedTransport, sink EventSink) *Session {
	t.Helper()
	return startTestSessionWithTool(t, root, workspacePath, transport, sink, ToolHandlerFunc(func(context.Context, ToolCall) (ToolResult, error) {
		return ToolResult{}, ErrUnsupportedTool
	}))
}

func startTestSessionWithTool(t *testing.T, root, workspacePath string, transport *scriptedTransport, sink EventSink, handler ToolHandler) *Session {
	t.Helper()
	return startTestSessionWithConfig(t, Config{
		Command:        "codex app-server",
		WorkspaceRoot:  root,
		ReadTimeout:    time.Second,
		TurnTimeout:    time.Second,
		ApprovalPolicy: "never",
		ThreadSandbox:  "workspace-write",
		DynamicTools:   []ToolSpec{{Name: "linear_graphql", Description: "Run Linear GraphQL"}},
	}, workspacePath, transport, sink, handler)
}

func startTestSessionWithConfig(t *testing.T, cfg Config, workspacePath string, transport *scriptedTransport, sink EventSink, handlers ...ToolHandler) *Session {
	t.Helper()
	handler := ToolHandler(ToolHandlerFunc(func(context.Context, ToolCall) (ToolResult, error) {
		return ToolResult{}, ErrUnsupportedTool
	}))
	if len(handlers) > 0 && handlers[0] != nil {
		handler = handlers[0]
	}
	session, err := StartSession(context.Background(), SessionOptions{
		Config:            cfg,
		WorkspacePath:     workspacePath,
		TransportFactory:  (&recordingFactory{transport: transport}).Start,
		ToolHandler:       handler,
		EventSink:         sink,
		NonInteractive:    true,
		TurnSandboxPolicy: map[string]any{"mode": "workspace-write"},
	})
	if err != nil {
		t.Fatalf("StartSession returned error: %v", err)
	}
	return session
}

func assertMethod(t *testing.T, payload map[string]any, want string) map[string]any {
	t.Helper()
	if payload["method"] != want {
		t.Fatalf("method = %#v, want %q in payload %#v", payload["method"], want, payload)
	}
	return payload
}

func responseForID(t *testing.T, writes []map[string]any, id string) map[string]any {
	t.Helper()
	for _, write := range writes {
		if write["id"] == id {
			return write
		}
	}
	t.Fatalf("no response for id %q in %#v", id, writes)
	return nil
}

func mapValue(t *testing.T, payload map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := payload[key].(map[string]any)
	if !ok {
		t.Fatalf("payload[%q] = %#v, want map", key, payload[key])
	}
	return value
}

func assertEventKinds(t *testing.T, events []Event, want ...EventKind) {
	t.Helper()
	var got []EventKind
	for _, event := range events {
		got = append(got, event.Kind)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("event kinds = %#v, want %#v", got, want)
	}
}

type recordingFactory struct {
	transport *scriptedTransport
	request   TransportRequest
	called    bool
}

func (f *recordingFactory) Start(_ context.Context, req TransportRequest) (Transport, error) {
	f.called = true
	f.request = req
	return f.transport, nil
}

type scriptedTransport struct {
	lines      chan []byte
	out        []map[string]any
	closedFlag bool
}

func newScriptedTransport(lines ...string) *scriptedTransport {
	ch := make(chan []byte, len(lines))
	for _, line := range lines {
		ch <- []byte(line)
	}
	return &scriptedTransport{lines: ch}
}

func (t *scriptedTransport) ReadLine(ctx context.Context) ([]byte, error) {
	select {
	case line := <-t.lines:
		return line, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (t *scriptedTransport) WriteJSON(_ context.Context, payload map[string]any) error {
	t.out = append(t.out, clonePayload(payload))
	return nil
}

func (t *scriptedTransport) Close() error {
	t.closedFlag = true
	return nil
}

func (t *scriptedTransport) writes() []map[string]any {
	return append([]map[string]any(nil), t.out...)
}

func (t *scriptedTransport) closed() bool {
	return t.closedFlag
}

func clonePayload(payload map[string]any) map[string]any {
	content, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	var cloned map[string]any
	if err := json.Unmarshal(content, &cloned); err != nil {
		panic(err)
	}
	return cloned
}
