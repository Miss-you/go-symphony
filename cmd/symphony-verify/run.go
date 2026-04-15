package main

import (
	"context"
	"fmt"

	"github.com/Miss-you/go-symphony/internal/cli"
	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/tracker"
)

func runRuntimeCommand(ctx context.Context, args []string, deps verifyDeps) int {
	parsed, err := parseRunArgs(args)
	if err != nil {
		_, _ = fmt.Fprintln(deps.stderr, usage())
		return 1
	}
	if !parsed.ack {
		_, _ = fmt.Fprintf(deps.stderr, "runtime smoke requires %s\n", ackFlag)
		return 1
	}
	if parsed.onlyIssue == "" {
		_, _ = fmt.Fprintln(deps.stderr, "runtime smoke requires --only-issue <id-or-identifier>")
		return 1
	}
	store, err := deps.openStore(parsed.workflowPath)
	if err != nil {
		_, _ = fmt.Fprintf(deps.stderr, "failed to load workflow: %v\n", err)
		return 1
	}
	defer func() { _ = store.Close() }()
	settings, err := store.CurrentSettings()
	if err != nil {
		_, _ = fmt.Fprintf(deps.stderr, "failed to load settings: %v\n", err)
		return 1
	}
	if settings.Provider.Kind != config.ProviderLinear {
		_, _ = fmt.Fprintln(deps.stderr, "runtime smoke requires tracker.kind=linear")
		return 1
	}
	reader, err := deps.newLinearReader(settings.Provider)
	if err != nil {
		_, _ = fmt.Fprintf(deps.stderr, "failed to create Linear reader: %v\n", err)
		return 1
	}
	filtered := tracker.NewFilteredReader(reader, parsed.onlyIssue)
	runtime, err := deps.startRuntime(ctx, cli.RuntimeOptions{
		Store:              store,
		Reader:             filtered,
		ServerPortOverride: parsed.portOverride,
	})
	if err != nil {
		_, _ = fmt.Fprintf(deps.stderr, "failed to start verification runtime: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintln(deps.stdout, "Verification runtime started")
	_, _ = fmt.Fprintf(deps.stdout, "only_issue: %s\n", parsed.onlyIssue)
	if dashboardURL := runtime.DashboardURL(); dashboardURL != "" {
		_, _ = fmt.Fprintf(deps.stdout, "dashboard: %s\n", dashboardURL)
	}

	waitCtx := ctx
	cancel := func() {}
	if parsed.timeout > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, parsed.timeout)
	}
	<-waitCtx.Done()
	cancel()

	snapshot := runtime.Snapshot()
	if err := runtime.Close(); err != nil {
		_, _ = fmt.Fprintf(deps.stderr, "failed to close verification runtime: %v\n", err)
		return 1
	}
	renderSnapshotSummary(deps.stdout, snapshot)
	return 0
}

func renderSnapshotSummary(w interface{ Write([]byte) (int, error) }, snapshot domain.Snapshot) {
	_, _ = fmt.Fprintln(w, "Final snapshot")
	_, _ = fmt.Fprintf(w, "running: %d\n", len(snapshot.Running))
	_, _ = fmt.Fprintf(w, "retrying: %d\n", len(snapshot.Retrying))
	_, _ = fmt.Fprintf(w, "total_tokens: %d\n", snapshot.CodexTotals.TotalTokens)
}
