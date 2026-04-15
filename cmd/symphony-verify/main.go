package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/Miss-you/go-symphony/internal/cli"
	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/tracker"
	lineartracker "github.com/Miss-you/go-symphony/internal/trackers/linear"
)

const ackFlag = "--i-understand-that-this-will-be-running-without-the-usual-guardrails"

type runtimeHandle interface {
	Close() error
	DashboardURL() string
	Snapshot() domain.Snapshot
}

type verifyDeps struct {
	stdout          io.Writer
	stderr          io.Writer
	openStore       func(string) (*config.Store, error)
	newLinearReader func(config.ProviderSettings) (tracker.TrackerReader, error)
	startRuntime    func(context.Context, cli.RuntimeOptions) (runtimeHandle, error)
}

func main() {
	os.Exit(mainWithDeps(context.Background(), os.Args[1:], realDeps()))
}

func realDeps() verifyDeps {
	return verifyDeps{
		stdout: os.Stdout,
		stderr: os.Stderr,
		openStore: func(path string) (*config.Store, error) {
			return config.NewStore(config.WithWorkflowPath(path))
		},
		newLinearReader: func(settings config.ProviderSettings) (tracker.TrackerReader, error) {
			return lineartracker.NewReader(settings, nil)
		},
		startRuntime: func(ctx context.Context, opts cli.RuntimeOptions) (runtimeHandle, error) {
			return cli.StartRuntime(ctx, opts)
		},
	}
}

func mainWithDeps(ctx context.Context, args []string, deps verifyDeps) int {
	deps = normalizeVerifyDeps(deps)
	if len(args) == 0 {
		_, _ = fmt.Fprintln(deps.stderr, usage())
		return 1
	}
	switch args[0] {
	case "linear":
		return runLinearCommand(ctx, args[1:], deps)
	case "run":
		return runRuntimeCommand(ctx, args[1:], deps)
	default:
		_, _ = fmt.Fprintln(deps.stderr, usage())
		return 1
	}
}

func normalizeVerifyDeps(deps verifyDeps) verifyDeps {
	if deps.stdout == nil {
		deps.stdout = io.Discard
	}
	if deps.stderr == nil {
		deps.stderr = io.Discard
	}
	if deps.openStore == nil {
		deps.openStore = realDeps().openStore
	}
	if deps.newLinearReader == nil {
		deps.newLinearReader = realDeps().newLinearReader
	}
	if deps.startRuntime == nil {
		deps.startRuntime = realDeps().startRuntime
	}
	return deps
}

func usage() string {
	return "Usage: symphony-verify <linear|run> [options] [path-to-WORKFLOW.md]"
}
