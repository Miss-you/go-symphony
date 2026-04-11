package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Miss-you/go-symphony/internal/codex"
	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/httpapi"
	"github.com/Miss-you/go-symphony/internal/observability"
	"github.com/Miss-you/go-symphony/internal/orchestrator"
	"github.com/Miss-you/go-symphony/internal/tracker"
	lineartracker "github.com/Miss-you/go-symphony/internal/trackers/linear"
	"github.com/Miss-you/go-symphony/internal/trackers/memory"
	"github.com/Miss-you/go-symphony/internal/web"
	"github.com/Miss-you/go-symphony/internal/workflow"
	"github.com/Miss-you/go-symphony/internal/workspace"
)

type RuntimeOptions struct {
	WorkflowPath     string
	Store            *config.Store
	Reader           tracker.TrackerReader
	Workspace        WorkspaceController
	TransportFactory codex.TransportFactory
	BundleFactory    BundleFactory
	MemoryItems      []domain.WorkItem
}

type BundleFactory func(config.Workflow, config.Settings) (RuntimeBundle, error)

type RuntimeBundle struct {
	PromptTemplate string
	DynamicTools   []codex.ToolSpec
	ToolHandler    codex.ToolHandler
}

type WorkspaceController interface {
	Create(identifier, workerHost string) (workspace.CreateResult, error)
	RunWithHooks(workspacePath, identifier, workerHost string, run func() error) error
	Remove(workspacePath, identifier, workerHost string) error
	RemoveIssueWorkspaces(identifier, workerHost string) error
}

type Runtime struct {
	store      *config.Store
	ownStore   bool
	service    *orchestrator.Service
	workers    *workerManager
	httpServer *http.Server
	dashboard  string
	closeOnce  sync.Once
	closeError error
}

func StartRuntime(ctx context.Context, opts RuntimeOptions) (*Runtime, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	store, ownStore, err := runtimeStore(opts)
	if err != nil {
		return nil, err
	}
	settings, err := store.CurrentSettings()
	if err != nil {
		if ownStore {
			_ = store.Close()
		}
		return nil, err
	}
	rawWorkflow, err := store.Current()
	if err != nil {
		if ownStore {
			_ = store.Close()
		}
		return nil, err
	}
	reader, err := runtimeReader(settings, opts)
	if err != nil {
		if ownStore {
			_ = store.Close()
		}
		return nil, err
	}
	workspaceManager := opts.Workspace
	if workspaceManager == nil {
		workspaceManager = workspace.NewManager(settings)
	}
	if err := startupCleanup(ctx, reader, workspaceManager, settings); err != nil {
		if ownStore {
			_ = store.Close()
		}
		return nil, err
	}

	workerRoot, cancelWorkers := context.WithCancel(ctx)
	workers := newWorkerManager(workerRoot, cancelWorkers, rawWorkflow, settings, reader, workspaceManager, opts)

	runtime := &Runtime{
		store:    store,
		ownStore: ownStore,
		workers:  workers,
	}
	svc := orchestrator.Start(settings, orchestrator.Dependencies{
		ListCandidates: reader.ListCandidates,
		RefreshItems:   reader.RefreshByIDs,
		StartRun:       workers.StartRun,
		StopRun:        workers.StopRun,
	})
	workers.emit = svc.ApplyRunEvent
	runtime.service = svc
	if err := runtime.startHTTPServer(settings); err != nil {
		_ = runtime.Close()
		return nil, err
	}
	return runtime, nil
}

func (r *Runtime) Snapshot() domain.Snapshot {
	if r == nil || r.service == nil {
		return domain.Snapshot{}
	}
	return r.service.Snapshot()
}

func (r *Runtime) DashboardURL() string {
	if r == nil {
		return ""
	}
	return r.dashboard
}

func (r *Runtime) Close() error {
	if r == nil {
		return nil
	}
	r.closeOnce.Do(func() {
		var errs []error
		if r.httpServer != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			errs = append(errs, r.httpServer.Shutdown(shutdownCtx))
			cancel()
		}
		if r.service != nil {
			errs = append(errs, r.service.Close())
		}
		if r.workers != nil {
			errs = append(errs, r.workers.Close())
		}
		if r.ownStore && r.store != nil {
			errs = append(errs, r.store.Close())
		}
		r.closeError = errors.Join(errs...)
	})
	return r.closeError
}

func (r *Runtime) startHTTPServer(settings config.Settings) error {
	if settings.Server.Port == nil {
		return nil
	}
	host := strings.TrimSpace(settings.Server.Host)
	if host == "" {
		host = "127.0.0.1"
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(host, fmt.Sprint(*settings.Server.Port)))
	if err != nil {
		return err
	}
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		_ = listener.Close()
		return err
	}
	r.dashboard = "http://" + net.JoinHostPort(host, port) + "/"
	server := &http.Server{
		Handler: web.NewHandler(web.Options{
			Snapshot: func(context.Context) (domain.Snapshot, error) {
				return r.Snapshot(), nil
			},
			Refresh: func(context.Context) (httpapi.RefreshResult, error) {
				if r.service == nil {
					return httpapi.RefreshResult{}, httpapi.ErrRefreshUnavailable
				}
				result := r.service.RequestRefresh()
				return httpapi.RefreshResult{Queued: result.Queued, Coalesced: result.Coalesced}, nil
			},
			WorkspaceRoot: settings.Workspace.Root,
			Now:           func() time.Time { return time.Now().UTC() },
			MaxAgents:     settings.Agent.MaxConcurrentAgents,
			DashboardURL:  r.dashboard,
			ProjectURL:    runtimeProjectURL(settings),
		}),
	}
	r.httpServer = server
	go func() {
		_ = server.Serve(listener)
	}()
	return nil
}

func runtimeProjectURL(settings config.Settings) string {
	if settings.Provider.Kind != config.ProviderLinear || strings.TrimSpace(settings.Provider.Project) == "" {
		return ""
	}
	return "https://linear.app/project/" + strings.TrimSpace(settings.Provider.Project) + "/issues"
}

func runtimeStore(opts RuntimeOptions) (*config.Store, bool, error) {
	if opts.Store != nil {
		return opts.Store, false, nil
	}
	store, err := config.NewStore(config.WithWorkflowPath(opts.WorkflowPath))
	if err != nil {
		return nil, false, err
	}
	return store, true, nil
}

func runtimeReader(settings config.Settings, opts RuntimeOptions) (tracker.TrackerReader, error) {
	if opts.Reader != nil {
		return opts.Reader, nil
	}
	switch settings.Provider.Kind {
	case config.ProviderLinear:
		return lineartracker.NewReader(settings.Provider, nil)
	case config.ProviderMemory:
		return memory.NewReader(opts.MemoryItems), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", settings.Provider.Kind)
	}
}

func startupCleanup(ctx context.Context, reader tracker.TrackerReader, manager WorkspaceController, settings config.Settings) error {
	items, err := reader.ListByStates(ctx, settings.Provider.TerminalStates)
	if err != nil {
		return err
	}
	for _, item := range items {
		if strings.TrimSpace(item.Identifier) == "" {
			continue
		}
		if err := manager.RemoveIssueWorkspaces(item.Identifier, ""); err != nil {
			return err
		}
	}
	return nil
}

func defaultBundleFactory(raw config.Workflow, settings config.Settings) (RuntimeBundle, error) {
	switch settings.Provider.Kind {
	case config.ProviderLinear:
		bundle, err := workflow.Select(raw, settings)
		if err != nil {
			return RuntimeBundle{}, err
		}
		return RuntimeBundle{
			PromptTemplate: bundle.PromptTemplate,
			DynamicTools:   bundle.DynamicTools,
			ToolHandler:    bundle.ToolHandler,
		}, nil
	case config.ProviderMemory:
		return RuntimeBundle{
			PromptTemplate: config.EffectivePromptTemplate(raw),
			ToolHandler: codex.ToolHandlerFunc(func(context.Context, codex.ToolCall) (codex.ToolResult, error) {
				return codex.ToolResult{}, codex.ErrUnsupportedTool
			}),
		}, nil
	default:
		return RuntimeBundle{}, fmt.Errorf("unsupported provider: %s", settings.Provider.Kind)
	}
}

type workerManager struct {
	rootCtx    context.Context
	cancelRoot context.CancelFunc
	workflow   config.Workflow
	settings   config.Settings
	reader     tracker.TrackerReader
	workspace  WorkspaceController
	transport  codex.TransportFactory
	bundles    BundleFactory
	emit       func(domain.RunEvent)
	mu         sync.Mutex
	handles    map[string]*runHandle
	closeOnce  sync.Once
	closed     bool
	closeWait  sync.WaitGroup
}

func newWorkerManager(ctx context.Context, cancel context.CancelFunc, rawWorkflow config.Workflow, settings config.Settings, reader tracker.TrackerReader, workspaceManager WorkspaceController, opts RuntimeOptions) *workerManager {
	transportFactory := opts.TransportFactory
	if transportFactory == nil {
		transportFactory = codex.StartProcessTransport
	}
	bundleFactory := opts.BundleFactory
	if bundleFactory == nil {
		bundleFactory = defaultBundleFactory
	}
	return &workerManager{
		rootCtx:    ctx,
		cancelRoot: cancel,
		workflow:   rawWorkflow,
		settings:   settings,
		reader:     reader,
		workspace:  workspaceManager,
		transport:  transportFactory,
		bundles:    bundleFactory,
		handles:    make(map[string]*runHandle),
	}
}

func (m *workerManager) StartRun(_ context.Context, req orchestrator.StartRunRequest) (orchestrator.StartRunResult, error) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return orchestrator.StartRunResult{}, context.Canceled
	}
	runCtx, cancel := context.WithCancel(m.rootCtx)
	handle := &runHandle{
		itemID:     req.Item.ID,
		identifier: req.Item.Identifier,
		workerHost: req.PreferredHost,
		cancel:     cancel,
		done:       make(chan struct{}),
	}
	m.handles[req.Item.ID] = handle
	m.closeWait.Add(1)
	m.mu.Unlock()

	go func() {
		defer m.closeWait.Done()
		defer close(handle.done)
		defer m.unregister(req.Item.ID)
		m.runWorker(runCtx, handle, req)
	}()

	return orchestrator.StartRunResult{Handle: handle, WorkerHost: req.PreferredHost}, nil
}

func (m *workerManager) StopRun(_ context.Context, req orchestrator.StopRunRequest) error {
	handle, _ := req.Handle.(*runHandle)
	if handle == nil {
		if req.CleanupWorkspace {
			return m.workspace.RemoveIssueWorkspaces(req.ItemIdentifier, "")
		}
		return nil
	}
	handle.stop()
	if req.CleanupWorkspace {
		path := handle.workspace()
		if path != "" {
			return m.workspace.Remove(path, req.ItemIdentifier, handle.workerHost)
		}
		return m.workspace.RemoveIssueWorkspaces(req.ItemIdentifier, handle.workerHost)
	}
	return nil
}

func (m *workerManager) Close() error {
	m.closeOnce.Do(func() {
		m.mu.Lock()
		m.closed = true
		var handles []*runHandle
		for _, handle := range m.handles {
			handles = append(handles, handle)
		}
		m.mu.Unlock()
		m.cancelRoot()
		for _, handle := range handles {
			handle.stop()
		}
		m.closeWait.Wait()
	})
	return nil
}

func (m *workerManager) unregister(itemID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.handles, itemID)
}

func (m *workerManager) runWorker(ctx context.Context, handle *runHandle, req orchestrator.StartRunRequest) {
	item := req.Item
	attempt := req.Attempt
	if attempt <= 0 {
		attempt = 0
	}
	m.emitEvent(domain.RunEvent{
		Kind:           domain.RunEventRunnerHostSelected,
		ItemID:         item.ID,
		ItemIdentifier: item.Identifier,
		At:             time.Now().UTC(),
		Attempt:        attempt,
		WorkerHost:     req.PreferredHost,
	})

	created, err := m.workspace.Create(item.Identifier, req.PreferredHost)
	if err != nil {
		m.emitFailure(item, attempt, req.PreferredHost, "", err)
		return
	}
	handle.setWorkspace(created.Path)
	m.emitEvent(domain.RunEvent{
		Kind:           domain.RunEventWorkspaceCreated,
		ItemID:         item.ID,
		ItemIdentifier: item.Identifier,
		At:             time.Now().UTC(),
		Attempt:        attempt,
		WorkerHost:     req.PreferredHost,
		WorkspacePath:  created.Path,
		Message:        workspaceMessage(created.Created),
	})
	m.emitEvent(domain.RunEvent{
		Kind:           domain.RunEventWorkspacePathDiscovered,
		ItemID:         item.ID,
		ItemIdentifier: item.Identifier,
		At:             time.Now().UTC(),
		Attempt:        attempt,
		WorkerHost:     req.PreferredHost,
		WorkspacePath:  created.Path,
	})

	err = m.workspace.RunWithHooks(created.Path, item.Identifier, req.PreferredHost, func() error {
		return m.runCodexLoop(ctx, handle, item, attempt, req.PreferredHost, created.Path)
	})
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		m.emitFailure(item, attempt, req.PreferredHost, created.Path, err)
		return
	}
	if ctx.Err() != nil {
		return
	}
	m.emitEvent(domain.RunEvent{
		Kind:           domain.RunEventRunCompleted,
		ItemID:         item.ID,
		ItemIdentifier: item.Identifier,
		At:             time.Now().UTC(),
		Attempt:        attempt,
		WorkerHost:     req.PreferredHost,
		WorkspacePath:  created.Path,
	})
}

func (m *workerManager) runCodexLoop(ctx context.Context, handle *runHandle, item domain.WorkItem, attempt int, workerHost, workspacePath string) error {
	bundle, err := m.bundles(m.workflow, m.settings)
	if err != nil {
		return err
	}
	codexConfig := codex.ConfigFromSettings(m.settings)
	codexConfig.DynamicTools = append([]codex.ToolSpec(nil), bundle.DynamicTools...)
	session, err := codex.StartSession(ctx, codex.SessionOptions{
		Config:           codexConfig,
		WorkspacePath:    workspacePath,
		TransportFactory: m.transport,
		ToolHandler:      firstToolHandler(bundle.ToolHandler),
		EventSink: func(event codex.Event) {
			m.emitCodexEvent(item, attempt, workerHost, workspacePath, event)
		},
		NonInteractive: true,
	})
	if err != nil {
		return err
	}
	handle.setSession(session)
	defer func() {
		_ = session.Close()
		handle.clearSession(session)
	}()

	current := item
	maxTurns := m.settings.Agent.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 1
	}
	for turn := 1; turn <= maxTurns; turn++ {
		prompt := continuationPrompt(turn, maxTurns)
		if turn == 1 {
			prompt = renderPrompt(bundle.PromptTemplate, current)
		}
		result, err := session.RunTurn(ctx, codex.TurnRequest{
			Prompt: prompt,
			Title:  current.Identifier + ": " + current.Title,
		})
		if err != nil {
			return err
		}
		switch result.Status {
		case codex.TurnCompleted:
			m.emitEvent(domain.RunEvent{
				Kind:           domain.RunEventTurnCompleted,
				ItemID:         item.ID,
				ItemIdentifier: item.Identifier,
				At:             time.Now().UTC(),
				Attempt:        attempt,
				WorkerHost:     workerHost,
				WorkspacePath:  workspacePath,
				SessionID:      session.ThreadID(),
				Message:        turnCompletedEventMessage(result.Usage),
				CodexTotals: domain.CodexTotals{
					InputTokens:  result.Usage.InputTokens,
					OutputTokens: result.Usage.OutputTokens,
					TotalTokens:  result.Usage.TotalTokens,
				},
			})
		case codex.TurnFailed, codex.TurnCancelled:
			return fmt.Errorf("codex turn %s", result.Status)
		default:
			return fmt.Errorf("unknown codex turn status %q", result.Status)
		}

		refreshed, active, err := m.refreshForContinuation(ctx, current.ID, m.settings)
		if err != nil {
			return err
		}
		if !active {
			return nil
		}
		current = refreshed
	}
	return nil
}

func (m *workerManager) refreshForContinuation(ctx context.Context, itemID string, settings config.Settings) (domain.WorkItem, bool, error) {
	items, err := m.reader.RefreshByIDs(ctx, []string{itemID})
	if err != nil {
		return domain.WorkItem{}, false, err
	}
	if len(items) == 0 {
		return domain.WorkItem{}, false, nil
	}
	item := items[0]
	if item.Routable != nil && !*item.Routable {
		return item, false, nil
	}
	if !stateIn(item.State, settings.Provider.ActiveStates) || stateIn(item.State, settings.Provider.TerminalStates) {
		return item, false, nil
	}
	return item, true, nil
}

func (m *workerManager) emitFailure(item domain.WorkItem, attempt int, workerHost, workspacePath string, err error) {
	message := ""
	if err != nil {
		message = err.Error()
	}
	m.emitEvent(domain.RunEvent{
		Kind:           domain.RunEventRunFailed,
		ItemID:         item.ID,
		ItemIdentifier: item.Identifier,
		At:             time.Now().UTC(),
		Attempt:        attempt,
		WorkerHost:     workerHost,
		WorkspacePath:  workspacePath,
		Message:        message,
		Err:            err,
	})
}

func (m *workerManager) emitCodexEvent(item domain.WorkItem, attempt int, workerHost, workspacePath string, event codex.Event) {
	message := observability.SummarizeCodexEvent(event)
	if message == "" {
		message = string(event.Kind)
		if event.Method != "" {
			message += ":" + event.Method
		}
	}
	sessionID := event.SessionID
	if sessionID == "" {
		sessionID = event.ThreadID
	}
	m.emitEvent(domain.RunEvent{
		Kind:           domain.RunEventCodexEventReceived,
		ItemID:         item.ID,
		ItemIdentifier: item.Identifier,
		At:             time.Now().UTC(),
		Attempt:        attempt,
		WorkerHost:     workerHost,
		WorkspacePath:  workspacePath,
		SessionID:      sessionID,
		Message:        message,
	})
}

func (m *workerManager) emitEvent(event domain.RunEvent) {
	if m.emit != nil {
		m.emit(event)
	}
}

func turnCompletedEventMessage(usage codex.TokenUsage) string {
	payload := map[string]any{
		"params": map[string]any{
			"usage": map[string]any{
				"input_tokens":  usage.InputTokens,
				"output_tokens": usage.OutputTokens,
				"total_tokens":  usage.TotalTokens,
			},
		},
	}
	return observability.SummarizeCodexEvent(codex.Event{
		Kind:    codex.EventTurnCompleted,
		Method:  "turn/completed",
		Payload: payload,
	})
}

type runHandle struct {
	mu            sync.Mutex
	itemID        string
	identifier    string
	workerHost    string
	workspacePath string
	cancel        context.CancelFunc
	session       *codex.Session
	done          chan struct{}
	stopOnce      sync.Once
}

func (h *runHandle) setWorkspace(path string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.workspacePath = path
}

func (h *runHandle) workspace() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.workspacePath
}

func (h *runHandle) setSession(session *codex.Session) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.session = session
}

func (h *runHandle) clearSession(session *codex.Session) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.session == session {
		h.session = nil
	}
}

func (h *runHandle) stop() {
	h.stopOnce.Do(func() {
		if h.cancel != nil {
			h.cancel()
		}
		h.mu.Lock()
		session := h.session
		h.mu.Unlock()
		if session != nil {
			_ = session.Close()
		}
	})
}

func firstToolHandler(handler codex.ToolHandler) codex.ToolHandler {
	if handler != nil {
		return handler
	}
	return codex.ToolHandlerFunc(func(context.Context, codex.ToolCall) (codex.ToolResult, error) {
		return codex.ToolResult{}, codex.ErrUnsupportedTool
	})
}

func workspaceMessage(created bool) string {
	if created {
		return "created"
	}
	return "reused"
}

func stateIn(state string, states []string) bool {
	normalized := strings.ToLower(strings.TrimSpace(state))
	for _, candidate := range states {
		if strings.ToLower(strings.TrimSpace(candidate)) == normalized {
			return true
		}
	}
	return false
}

func renderPrompt(template string, item domain.WorkItem) string {
	if strings.TrimSpace(template) == "" {
		template = config.EffectivePromptTemplate(config.Workflow{})
	}
	description := strings.TrimSpace(item.Description)
	if strings.Contains(template, "{% if issue.description %}") {
		template = renderIssueDescriptionConditional(template, description)
	}
	replacements := map[string]string{
		"issue.id":          item.ID,
		"issue.identifier":  item.Identifier,
		"issue.title":       item.Title,
		"issue.description": item.Description,
		"issue.state":       item.State,
		"issue.url":         item.URL,
		"issue.branch_name": item.BranchName,
		"issue.branchName":  item.BranchName,
	}
	for key, value := range replacements {
		template = strings.ReplaceAll(template, "{{ "+key+" }}", value)
		template = strings.ReplaceAll(template, "{{"+key+"}}", value)
	}
	return strings.TrimSpace(template)
}

func renderIssueDescriptionConditional(template, description string) string {
	const (
		ifMarker    = "{% if issue.description %}"
		elseMarker  = "{% else %}"
		endifMarker = "{% endif %}"
	)

	startIndex := strings.Index(template, ifMarker)
	if startIndex < 0 {
		return template
	}
	thenStart := startIndex + len(ifMarker)
	elseOffset := strings.Index(template[thenStart:], elseMarker)
	if elseOffset < 0 {
		return template
	}
	elseIndex := thenStart + elseOffset
	elseStart := elseIndex + len(elseMarker)
	endifOffset := strings.Index(template[elseStart:], endifMarker)
	if endifOffset < 0 {
		return template
	}
	endifIndex := elseStart + endifOffset
	endifEnd := endifIndex + len(endifMarker)

	selected := template[thenStart:elseIndex]
	if description == "" {
		selected = template[elseStart:endifIndex]
	}
	return template[:startIndex] + selected + template[endifEnd:]
}

func continuationPrompt(turn, maxTurns int) string {
	return fmt.Sprintf(`Continuation guidance:

- The previous Codex turn completed normally, but the work item is still in an active state.
- This is continuation turn #%d of %d for the current agent run.
- Resume from the current workspace and thread context instead of restarting from scratch.
- Focus on the remaining ticket work.`, turn, maxTurns)
}
