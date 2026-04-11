package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Miss-you/go-symphony/internal/config"
)

const (
	initializeID  = 1
	threadStartID = 2
	turnStartID   = 3

	NonInteractiveInputAnswer = "This is a non-interactive session. Operator input is unavailable."
)

var (
	ErrReadTimeout      = errors.New("codex read timeout")
	ErrTurnTimeout      = errors.New("codex turn timeout")
	ErrApprovalRequired = errors.New("codex approval required")
	ErrUnsupportedTool  = errors.New("unsupported tool")
)

type Config struct {
	Command           string
	WorkspaceRoot     string
	ApprovalPolicy    any
	ThreadSandbox     string
	TurnSandboxPolicy map[string]any
	DynamicTools      []ToolSpec
	ReadTimeout       time.Duration
	TurnTimeout       time.Duration
}

type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type TransportRequest struct {
	Command string
	Dir     string
}

type Transport interface {
	ReadLine(context.Context) ([]byte, error)
	WriteJSON(context.Context, map[string]any) error
	Close() error
}

type TransportFactory func(context.Context, TransportRequest) (Transport, error)

type EventKind string

const (
	EventSessionStarted      EventKind = "session_started"
	EventTurnCompleted       EventKind = "turn_completed"
	EventTurnFailed          EventKind = "turn_failed"
	EventTurnCancelled       EventKind = "turn_cancelled"
	EventToolCallCompleted   EventKind = "tool_call_completed"
	EventToolCallFailed      EventKind = "tool_call_failed"
	EventUnsupportedToolCall EventKind = "unsupported_tool_call"
	EventApprovalAnswered    EventKind = "approval_answered"
	EventToolInputAnswered   EventKind = "tool_input_answered"
	EventMalformedMessage    EventKind = "malformed_message"
	EventUnknownMessage      EventKind = "unknown_message"
)

type Event struct {
	Kind      EventKind
	Method    string
	Raw       string
	SessionID string
	ThreadID  string
	TurnID    string
	Message   string
	Payload   map[string]any
	Err       error
}

type EventSink func(Event)

type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}

type ToolResult struct {
	Success bool   `json:"success"`
	Result  any    `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ToolHandler interface {
	HandleTool(context.Context, ToolCall) (ToolResult, error)
}

type ToolHandlerFunc func(context.Context, ToolCall) (ToolResult, error)

func (f ToolHandlerFunc) HandleTool(ctx context.Context, call ToolCall) (ToolResult, error) {
	return f(ctx, call)
}

type TurnStatus string

const (
	TurnCompleted TurnStatus = "completed"
	TurnFailed    TurnStatus = "failed"
	TurnCancelled TurnStatus = "cancelled"
)

type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

type TurnRequest struct {
	Prompt string
	Title  string
}

type TurnResult struct {
	Status TurnStatus
	TurnID string
	Usage  TokenUsage
}

type SessionOptions struct {
	Config            Config
	WorkspacePath     string
	TransportFactory  TransportFactory
	ToolHandler       ToolHandler
	EventSink         EventSink
	NonInteractive    bool
	TurnSandboxPolicy map[string]any
}

type WorkspaceErrorKind string

const (
	ErrWorkspacePathUnreadable WorkspaceErrorKind = "workspace_path_unreadable"
	ErrWorkspaceEqualsRoot     WorkspaceErrorKind = "workspace_equals_root"
	ErrWorkspaceOutsideRoot    WorkspaceErrorKind = "workspace_outside_root"
	ErrWorkspaceSymlinkEscape  WorkspaceErrorKind = "workspace_symlink_escape"
)

type WorkspaceError struct {
	Kind WorkspaceErrorKind
	Path string
	Root string
	Err  error
}

func (e *WorkspaceError) Error() string {
	if e == nil {
		return "<nil>"
	}
	parts := []string{string(e.Kind)}
	if e.Path != "" {
		parts = append(parts, "path="+e.Path)
	}
	if e.Root != "" {
		parts = append(parts, "root="+e.Root)
	}
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	return strings.Join(parts, " ")
}

func (e *WorkspaceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type ProtocolError struct {
	Kind    string
	Message string
	Payload map[string]any
	Err     error
}

func (e *ProtocolError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Message != "" {
		return e.Kind + ": " + e.Message
	}
	if e.Err != nil {
		return e.Kind + ": " + e.Err.Error()
	}
	return e.Kind
}

func (e *ProtocolError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type Session struct {
	mu     sync.Mutex
	turnMu sync.Mutex

	cfg               Config
	workspacePath     string
	transport         Transport
	toolHandler       ToolHandler
	eventSink         EventSink
	nonInteractive    bool
	turnSandboxPolicy map[string]any

	threadID string
	closed   bool
}

func ConfigFromSettings(settings config.Settings) Config {
	return Config{
		Command:           settings.Codex.Command,
		WorkspaceRoot:     settings.Workspace.Root,
		ApprovalPolicy:    cloneValue(settings.Codex.ApprovalPolicy),
		ThreadSandbox:     settings.Codex.ThreadSandbox,
		TurnSandboxPolicy: cloneMap(settings.Codex.TurnSandboxPolicy),
		ReadTimeout:       time.Duration(settings.Codex.ReadTimeoutMS) * time.Millisecond,
		TurnTimeout:       time.Duration(settings.Codex.TurnTimeoutMS) * time.Millisecond,
	}
}

func StartSession(ctx context.Context, opts SessionOptions) (*Session, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := normalizeConfig(opts.Config)
	workspacePath, err := validateWorkspacePath(cfg.WorkspaceRoot, opts.WorkspacePath)
	if err != nil {
		return nil, err
	}
	factory := opts.TransportFactory
	if factory == nil {
		factory = StartProcessTransport
	}
	transport, err := factory(ctx, TransportRequest{Command: cfg.Command, Dir: workspacePath})
	if err != nil {
		return nil, err
	}
	session := &Session{
		cfg:               cfg,
		workspacePath:     workspacePath,
		transport:         transport,
		toolHandler:       opts.ToolHandler,
		eventSink:         opts.EventSink,
		nonInteractive:    opts.NonInteractive,
		turnSandboxPolicy: cloneMap(firstMap(opts.TurnSandboxPolicy, cfg.TurnSandboxPolicy)),
	}
	if session.toolHandler == nil {
		session.toolHandler = ToolHandlerFunc(func(context.Context, ToolCall) (ToolResult, error) {
			return ToolResult{}, ErrUnsupportedTool
		})
	}
	if err := session.bootstrap(ctx); err != nil {
		_ = transport.Close()
		return nil, err
	}
	return session, nil
}

func (s *Session) ThreadID() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.threadID
}

func (s *Session) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.transport == nil {
		return nil
	}
	return s.transport.Close()
}

func (s *Session) RunTurn(ctx context.Context, req TurnRequest) (TurnResult, error) {
	if s == nil {
		return TurnResult{}, errors.New("codex session is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s.turnMu.Lock()
	defer s.turnMu.Unlock()

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return TurnResult{}, errors.New("codex session is closed")
	}
	s.mu.Unlock()

	if err := s.write(ctx, map[string]any{
		"method": "turn/start",
		"id":     turnStartID,
		"params": map[string]any{
			"threadId":       s.threadID,
			"input":          []map[string]any{{"type": "text", "text": req.Prompt}},
			"cwd":            s.workspacePath,
			"title":          req.Title,
			"approvalPolicy": s.cfg.ApprovalPolicy,
			"sandboxPolicy":  cloneMap(s.turnSandboxPolicy),
		},
	}); err != nil {
		return TurnResult{}, err
	}
	result, err := s.awaitResponse(ctx, turnStartID)
	if err != nil {
		return TurnResult{}, err
	}
	turnID := nestedString(result, "turn", "id")
	if turnID == "" {
		return TurnResult{}, &ProtocolError{Kind: "invalid_turn_payload", Payload: result}
	}

	turnCtx, cancel := context.WithTimeout(ctx, s.cfg.TurnTimeout)
	defer cancel()
	for {
		line, err := s.transport.ReadLine(turnCtx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(turnCtx.Err(), context.DeadlineExceeded) {
				return TurnResult{}, fmt.Errorf("%w", ErrTurnTimeout)
			}
			return TurnResult{}, err
		}
		payload, raw, ok := decodeLine(line)
		if !ok {
			s.emit(Event{Kind: EventMalformedMessage, Raw: raw, ThreadID: s.threadID, TurnID: turnID})
			continue
		}
		method := stringValue(payload["method"])
		switch method {
		case "turn/completed":
			usage := usageFromPayload(payload)
			s.emit(Event{Kind: EventTurnCompleted, Method: method, Raw: raw, ThreadID: s.threadID, TurnID: turnID, Payload: payload})
			return TurnResult{Status: TurnCompleted, TurnID: turnID, Usage: usage}, nil
		case "turn/failed":
			s.emit(Event{Kind: EventTurnFailed, Method: method, Raw: raw, ThreadID: s.threadID, TurnID: turnID, Payload: payload})
			return TurnResult{Status: TurnFailed, TurnID: turnID}, nil
		case "turn/cancelled":
			s.emit(Event{Kind: EventTurnCancelled, Method: method, Raw: raw, ThreadID: s.threadID, TurnID: turnID, Payload: payload})
			return TurnResult{Status: TurnCancelled, TurnID: turnID}, nil
		case "item/commandExecution/requestApproval", "execCommandApproval", "applyPatchApproval", "item/fileChange/requestApproval":
			if err := s.handleApproval(ctx, payload, raw, turnID); err != nil {
				return TurnResult{}, err
			}
		case "item/tool/requestUserInput":
			if err := s.handleUserInput(ctx, payload, raw, turnID); err != nil {
				return TurnResult{}, err
			}
		case "item/tool/call":
			if err := s.handleToolCall(ctx, payload, raw, turnID); err != nil {
				return TurnResult{}, err
			}
		default:
			s.emit(Event{Kind: EventUnknownMessage, Method: method, Raw: raw, ThreadID: s.threadID, TurnID: turnID, Payload: payload})
		}
	}
}

func (s *Session) bootstrap(ctx context.Context) error {
	if err := s.write(ctx, map[string]any{
		"method": "initialize",
		"id":     initializeID,
		"params": map[string]any{
			"capabilities": map[string]any{"experimentalApi": true},
			"clientInfo": map[string]any{
				"name":    "symphony-orchestrator",
				"title":   "Symphony Orchestrator",
				"version": "0.1.0",
			},
		},
	}); err != nil {
		return err
	}
	if _, err := s.awaitResponse(ctx, initializeID); err != nil {
		return err
	}
	if err := s.write(ctx, map[string]any{"method": "initialized", "params": map[string]any{}}); err != nil {
		return err
	}
	if err := s.write(ctx, map[string]any{
		"method": "thread/start",
		"id":     threadStartID,
		"params": map[string]any{
			"approvalPolicy": s.cfg.ApprovalPolicy,
			"sandbox":        s.cfg.ThreadSandbox,
			"cwd":            s.workspacePath,
			"dynamicTools":   s.cfg.DynamicTools,
		},
	}); err != nil {
		return err
	}
	result, err := s.awaitResponse(ctx, threadStartID)
	if err != nil {
		return err
	}
	threadID := nestedString(result, "thread", "id")
	if threadID == "" {
		return &ProtocolError{Kind: "invalid_thread_payload", Payload: result}
	}
	s.threadID = threadID
	s.emit(Event{Kind: EventSessionStarted, ThreadID: threadID})
	return nil
}

func (s *Session) awaitResponse(ctx context.Context, id any) (map[string]any, error) {
	readCtx, cancel := context.WithTimeout(ctx, s.cfg.ReadTimeout)
	defer cancel()
	for {
		line, err := s.transport.ReadLine(readCtx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(readCtx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("%w", ErrReadTimeout)
			}
			return nil, err
		}
		payload, raw, ok := decodeLine(line)
		if !ok {
			s.emit(Event{Kind: EventMalformedMessage, Raw: raw, ThreadID: s.threadID})
			continue
		}
		if !sameID(payload["id"], id) {
			s.emit(Event{Kind: EventUnknownMessage, Method: stringValue(payload["method"]), Raw: raw, ThreadID: s.threadID, Payload: payload})
			continue
		}
		if responseError, ok := payload["error"]; ok {
			return nil, &ProtocolError{Kind: "response_error", Message: fmt.Sprint(responseError), Payload: payload}
		}
		result, ok := payload["result"].(map[string]any)
		if !ok {
			return nil, &ProtocolError{Kind: "response_error", Payload: payload}
		}
		return result, nil
	}
}

func (s *Session) handleApproval(ctx context.Context, payload map[string]any, raw, turnID string) error {
	if !isNeverApprovalPolicy(s.cfg.ApprovalPolicy) {
		return ErrApprovalRequired
	}
	id := payload["id"]
	decision := approvalDecision(stringValue(payload["method"]))
	if decision == "" {
		return ErrApprovalRequired
	}
	if err := s.write(ctx, map[string]any{"id": id, "result": map[string]any{"decision": decision}}); err != nil {
		return err
	}
	s.emit(Event{Kind: EventApprovalAnswered, Method: stringValue(payload["method"]), Raw: raw, ThreadID: s.threadID, TurnID: turnID, Payload: payload})
	return nil
}

func (s *Session) handleUserInput(ctx context.Context, payload map[string]any, raw, turnID string) error {
	answers, ok := approvalAnswers(payload)
	if !ok {
		answers, ok = unavailableAnswers(payload)
	}
	if !ok {
		return ErrApprovalRequired
	}
	if err := s.write(ctx, map[string]any{"id": payload["id"], "result": map[string]any{"answers": answers}}); err != nil {
		return err
	}
	s.emit(Event{Kind: EventToolInputAnswered, Method: stringValue(payload["method"]), Raw: raw, ThreadID: s.threadID, TurnID: turnID, Payload: payload})
	return nil
}

func (s *Session) handleToolCall(ctx context.Context, payload map[string]any, raw, turnID string) error {
	call := ToolCall{
		ID:        idString(payload["id"]),
		Name:      toolCallName(payload),
		Arguments: toolCallArguments(payload),
	}
	eventKind := EventToolCallCompleted
	result := ToolResult{}
	err := error(nil)
	if call.Name == "" {
		err = ErrUnsupportedTool
	} else {
		result, err = s.toolHandler.HandleTool(ctx, call)
	}
	if err != nil {
		if errors.Is(err, ErrUnsupportedTool) {
			result = ToolResult{Success: false, Error: "unsupported_tool_call"}
			eventKind = EventUnsupportedToolCall
		} else {
			result = ToolResult{Success: false, Error: err.Error()}
			eventKind = EventToolCallFailed
		}
	} else if !result.Success {
		eventKind = EventToolCallFailed
	}
	if err := s.write(ctx, map[string]any{"id": payload["id"], "result": result}); err != nil {
		return err
	}
	s.emit(Event{Kind: eventKind, Method: stringValue(payload["method"]), Raw: raw, ThreadID: s.threadID, TurnID: turnID, Payload: payload})
	return nil
}

func (s *Session) write(ctx context.Context, payload map[string]any) error {
	return s.transport.WriteJSON(ctx, payload)
}

func (s *Session) emit(event Event) {
	if s.eventSink != nil {
		s.eventSink(event)
	}
}

func StartProcessTransport(ctx context.Context, req TransportRequest) (Transport, error) {
	command := strings.TrimSpace(req.Command)
	if command == "" {
		command = "codex app-server"
	}
	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Dir = req.Dir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	transport := &processTransport{
		cmd:   cmd,
		stdin: stdin,
		reads: make(chan readResult, 128),
	}
	go transport.scan(stdout)
	go func() {
		_, _ = io.Copy(io.Discard, stderr)
	}()
	return transport, nil
}

type readResult struct {
	line []byte
	err  error
}

type processTransport struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser

	writeMu sync.Mutex
	closeMu sync.Mutex
	closed  bool

	reads chan readResult
}

func (t *processTransport) ReadLine(ctx context.Context) ([]byte, error) {
	select {
	case result := <-t.reads:
		return result.line, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (t *processTransport) WriteJSON(_ context.Context, payload map[string]any) error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	content, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	content = append(content, '\n')
	_, err = t.stdin.Write(content)
	return err
}

func (t *processTransport) Close() error {
	t.closeMu.Lock()
	defer t.closeMu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true
	_ = t.stdin.Close()
	if t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
	}
	_ = t.cmd.Wait()
	return nil
}

func (t *processTransport) scan(reader io.Reader) {
	buffered := bufio.NewReader(reader)
	for {
		line, err := buffered.ReadBytes('\n')
		if len(line) > 0 {
			line = bytes.TrimRight(line, "\r\n")
			t.reads <- readResult{line: append([]byte(nil), line...)}
		}
		if err != nil {
			t.reads <- readResult{err: err}
			return
		}
	}
}

func normalizeConfig(cfg Config) Config {
	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = 5 * time.Second
	}
	if cfg.TurnTimeout <= 0 {
		cfg.TurnTimeout = time.Hour
	}
	cfg.TurnSandboxPolicy = cloneMap(cfg.TurnSandboxPolicy)
	cfg.WorkspaceRoot = strings.TrimSpace(cfg.WorkspaceRoot)
	cfg.Command = strings.TrimSpace(cfg.Command)
	return cfg
}

func validateWorkspacePath(root, workspacePath string) (string, error) {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(workspacePath) == "" {
		return "", &WorkspaceError{Kind: ErrWorkspacePathUnreadable, Path: workspacePath, Root: root}
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", &WorkspaceError{Kind: ErrWorkspacePathUnreadable, Path: root, Root: root, Err: err}
	}
	workspaceAbs, err := filepath.Abs(workspacePath)
	if err != nil {
		return "", &WorkspaceError{Kind: ErrWorkspacePathUnreadable, Path: workspacePath, Root: root, Err: err}
	}
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", &WorkspaceError{Kind: ErrWorkspacePathUnreadable, Path: rootAbs, Root: rootAbs, Err: err}
	}
	workspaceReal, err := filepath.EvalSymlinks(workspaceAbs)
	if err != nil {
		return "", &WorkspaceError{Kind: ErrWorkspacePathUnreadable, Path: workspaceAbs, Root: rootReal, Err: err}
	}
	rootReal = filepath.Clean(rootReal)
	workspaceReal = filepath.Clean(workspaceReal)
	workspaceAbs = filepath.Clean(workspaceAbs)
	if workspaceReal == rootReal {
		return "", &WorkspaceError{Kind: ErrWorkspaceEqualsRoot, Path: workspaceReal, Root: rootReal}
	}
	if !pathWithin(rootReal, workspaceReal) {
		kind := ErrWorkspaceOutsideRoot
		if pathWithin(filepath.Clean(rootAbs), workspaceAbs) {
			kind = ErrWorkspaceSymlinkEscape
		}
		return "", &WorkspaceError{Kind: kind, Path: workspaceReal, Root: rootReal}
	}
	return workspaceReal, nil
}

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func decodeLine(line []byte) (map[string]any, string, bool) {
	raw := string(line)
	var payload map[string]any
	if err := json.Unmarshal(line, &payload); err != nil {
		return nil, raw, false
	}
	return payload, raw, true
}

func usageFromPayload(payload map[string]any) TokenUsage {
	params, _ := payload["params"].(map[string]any)
	usage, _ := params["usage"].(map[string]any)
	return TokenUsage{
		InputTokens:  intValue(firstPresent(usage, "input_tokens", "inputTokens")),
		OutputTokens: intValue(firstPresent(usage, "output_tokens", "outputTokens")),
		TotalTokens:  intValue(firstPresent(usage, "total_tokens", "totalTokens")),
	}
}

func approvalDecision(method string) string {
	switch method {
	case "item/commandExecution/requestApproval", "item/fileChange/requestApproval":
		return "acceptForSession"
	case "execCommandApproval", "applyPatchApproval":
		return "approved_for_session"
	default:
		return ""
	}
}

func approvalAnswers(payload map[string]any) (map[string]any, bool) {
	questions := payloadQuestions(payload)
	if len(questions) == 0 {
		return nil, false
	}
	answers := make(map[string]any, len(questions))
	for _, question := range questions {
		id := stringValue(question["id"])
		if id == "" {
			return nil, false
		}
		label := approvalOptionLabel(question)
		if label == "" {
			return nil, false
		}
		answers[id] = map[string]any{"answers": []string{label}}
	}
	return answers, true
}

func unavailableAnswers(payload map[string]any) (map[string]any, bool) {
	questions := payloadQuestions(payload)
	if len(questions) == 0 {
		return nil, false
	}
	answers := make(map[string]any, len(questions))
	for _, question := range questions {
		id := stringValue(question["id"])
		if id == "" {
			return nil, false
		}
		answers[id] = map[string]any{"answers": []string{NonInteractiveInputAnswer}}
	}
	return answers, true
}

func payloadQuestions(payload map[string]any) []map[string]any {
	params, _ := payload["params"].(map[string]any)
	rawQuestions, _ := params["questions"].([]any)
	questions := make([]map[string]any, 0, len(rawQuestions))
	for _, rawQuestion := range rawQuestions {
		if question, ok := rawQuestion.(map[string]any); ok {
			questions = append(questions, question)
		}
	}
	return questions
}

func approvalOptionLabel(question map[string]any) string {
	rawOptions, _ := question["options"].([]any)
	var fallback string
	for _, rawOption := range rawOptions {
		option, _ := rawOption.(map[string]any)
		label := stringValue(option["label"])
		normalized := strings.ToLower(strings.TrimSpace(label))
		if label == "Approve this Session" {
			return label
		}
		if label == "Approve Once" {
			return label
		}
		if fallback == "" && (strings.HasPrefix(normalized, "approve") || strings.HasPrefix(normalized, "allow")) {
			fallback = label
		}
	}
	return fallback
}

func toolCallName(payload map[string]any) string {
	params, _ := payload["params"].(map[string]any)
	if name := stringValue(params["tool"]); name != "" {
		return name
	}
	if name := stringValue(params["name"]); name != "" {
		return name
	}
	if name := stringValue(params["toolName"]); name != "" {
		return name
	}
	if tool, ok := params["tool"].(map[string]any); ok {
		return stringValue(firstPresent(tool, "name", "toolName"))
	}
	return ""
}

func toolCallArguments(payload map[string]any) map[string]any {
	params, _ := payload["params"].(map[string]any)
	if args, ok := params["arguments"].(map[string]any); ok {
		return cloneMap(args)
	}
	if args, ok := params["args"].(map[string]any); ok {
		return cloneMap(args)
	}
	return map[string]any{}
}

func nestedString(payload map[string]any, keys ...string) string {
	var current any = payload
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[key]
	}
	return stringValue(current)
}

func firstMap(values ...map[string]any) map[string]any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstPresent(m map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value
		}
	}
	return nil
}

func sameID(got, want any) bool {
	return idString(got) == idString(want)
}

func idString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case json.Number:
		return v.String()
	default:
		return fmt.Sprint(v)
	}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}

func intValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}

func isNeverApprovalPolicy(value any) bool {
	policy, ok := value.(string)
	return ok && strings.EqualFold(strings.TrimSpace(policy), "never")
}

func cloneValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return cloneMap(v)
	case []any:
		output := make([]any, len(v))
		for i, item := range v {
			output[i] = cloneValue(item)
		}
		return output
	default:
		return v
	}
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = cloneValue(value)
	}
	return output
}
