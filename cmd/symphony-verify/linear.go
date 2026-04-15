package main

import (
	"context"
	"fmt"

	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/tracker"
	"github.com/Miss-you/go-symphony/internal/verify/linearprobe"
)

func runLinearCommand(ctx context.Context, args []string, deps verifyDeps) int {
	parsed, err := parseLinearArgs(args)
	if err != nil {
		_, _ = fmt.Fprintln(deps.stderr, usage())
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
		_, _ = fmt.Fprintln(deps.stderr, "linear probe requires tracker.kind=linear")
		return 1
	}
	reader, err := deps.newLinearReader(settings.Provider)
	if err != nil {
		_, _ = fmt.Fprintf(deps.stderr, "failed to create Linear reader: %v\n", err)
		return 1
	}
	if parsed.onlyIssue != "" {
		reader = tracker.NewFilteredReader(reader, parsed.onlyIssue)
	}
	result, err := linearprobe.Run(ctx, settings, reader, parsed.refreshIDs)
	if err != nil {
		_, _ = fmt.Fprintf(deps.stderr, "linear probe failed: %v\n", err)
		return 1
	}
	linearprobe.Render(deps.stdout, result, parsed.limit)
	return 0
}
