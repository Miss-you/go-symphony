package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Miss-you/go-symphony/internal/dashboard"
)

const acknowledgementFlag = "--i-understand-that-this-will-be-running-without-the-usual-guardrails"

var errUsage = errors.New("usage")

type cliArgs struct {
	acknowledged bool
	workflowPath string
	logsRoot     string
	portOverride *int
}

type runtimeHandle interface {
	Close() error
}

type mainDeps struct {
	stdout           io.Writer
	stderr           io.Writer
	fileRegular      func(string) bool
	configureLogFile func(string) (func() error, string, error)
	startRuntime     func(context.Context, RuntimeOptions) (runtimeHandle, error)
}

func Main(ctx context.Context, args []string) int {
	return mainWithDeps(ctx, args, mainDeps{
		stdout:           os.Stdout,
		stderr:           os.Stderr,
		fileRegular:      fileRegular,
		configureLogFile: configureLogFile,
		startRuntime: func(ctx context.Context, opts RuntimeOptions) (runtimeHandle, error) {
			return StartRuntime(ctx, opts)
		},
	})
}

func mainWithDeps(ctx context.Context, args []string, deps mainDeps) int {
	if ctx == nil {
		ctx = context.Background()
	}
	deps = normalizeMainDeps(deps)
	parsed, err := parseCLIArgs(args)
	if errors.Is(err, errUsage) {
		_, _ = fmt.Fprintln(deps.stderr, usageMessage())
		return 1
	}
	if err != nil {
		_, _ = fmt.Fprintln(deps.stderr, err)
		return 1
	}
	if !parsed.acknowledged {
		_, _ = fmt.Fprintln(deps.stderr, acknowledgementBanner())
		return 1
	}

	workflowPath, err := filepath.Abs(parsed.workflowPath)
	if err != nil {
		_, _ = fmt.Fprintln(deps.stderr, err)
		return 1
	}
	if !deps.fileRegular(workflowPath) {
		_, _ = fmt.Fprintf(deps.stderr, "Workflow file not found: %s\n", workflowPath)
		return 1
	}

	logsRoot, err := expandedLogsRoot(parsed.logsRoot)
	if err != nil {
		_, _ = fmt.Fprintln(deps.stderr, err)
		return 1
	}
	restoreLog, _, err := deps.configureLogFile(logsRoot)
	if err != nil {
		_, _ = fmt.Fprintln(deps.stderr, err)
		return 1
	}
	if restoreLog != nil {
		defer func() { _ = restoreLog() }()
	}

	runtime, err := deps.startRuntime(ctx, RuntimeOptions{
		WorkflowPath:       workflowPath,
		ServerPortOverride: parsed.portOverride,
	})
	if err != nil {
		_, _ = fmt.Fprintf(deps.stderr, "Failed to start Symphony with workflow %s: %v\n", workflowPath, err)
		return 1
	}
	<-ctx.Done()
	if err := runtime.Close(); err != nil {
		_, _ = fmt.Fprintln(deps.stderr, err)
		return 1
	}
	_, _ = fmt.Fprintln(deps.stdout, dashboard.RenderOffline())
	return 0
}

func normalizeMainDeps(deps mainDeps) mainDeps {
	if deps.stdout == nil {
		deps.stdout = io.Discard
	}
	if deps.stderr == nil {
		deps.stderr = io.Discard
	}
	if deps.fileRegular == nil {
		deps.fileRegular = fileRegular
	}
	if deps.configureLogFile == nil {
		deps.configureLogFile = configureLogFile
	}
	if deps.startRuntime == nil {
		deps.startRuntime = func(ctx context.Context, opts RuntimeOptions) (runtimeHandle, error) {
			return StartRuntime(ctx, opts)
		}
	}
	return deps
}

func parseCLIArgs(args []string) (cliArgs, error) {
	parsed := cliArgs{workflowPath: "WORKFLOW.md"}
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == acknowledgementFlag:
			parsed.acknowledged = true
		case arg == "--logs-root":
			value, ok := nextFlagValue(args, &i)
			if !ok || strings.TrimSpace(value) == "" || strings.HasPrefix(value, "--") {
				return cliArgs{}, errUsage
			}
			parsed.logsRoot = strings.TrimSpace(value)
		case strings.HasPrefix(arg, "--logs-root="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--logs-root="))
			if value == "" {
				return cliArgs{}, errUsage
			}
			parsed.logsRoot = value
		case arg == "--port":
			value, ok := nextFlagValue(args, &i)
			if !ok {
				return cliArgs{}, errUsage
			}
			port, err := parsePort(value)
			if err != nil {
				return cliArgs{}, errUsage
			}
			parsed.portOverride = &port
		case strings.HasPrefix(arg, "--port="):
			port, err := parsePort(strings.TrimPrefix(arg, "--port="))
			if err != nil {
				return cliArgs{}, errUsage
			}
			parsed.portOverride = &port
		case strings.HasPrefix(arg, "--"):
			return cliArgs{}, errUsage
		default:
			positionals = append(positionals, arg)
		}
	}
	if len(positionals) > 1 {
		return cliArgs{}, errUsage
	}
	if len(positionals) == 1 {
		parsed.workflowPath = positionals[0]
	}
	return parsed, nil
}

func nextFlagValue(args []string, index *int) (string, bool) {
	if *index+1 >= len(args) {
		return "", false
	}
	*index++
	return args[*index], true
}

func parsePort(value string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || port < 0 {
		return 0, errUsage
	}
	return port, nil
}

func expandedLogsRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return os.Getwd()
	}
	return filepath.Abs(root)
}

func fileRegular(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func usageMessage() string {
	return "Usage: symphony [--logs-root <path>] [--port <port>] [path-to-WORKFLOW.md]"
}

func acknowledgementBanner() string {
	lines := []string{
		"This Symphony implementation is a low key engineering preview.",
		"Codex will run without any guardrails.",
		"SymphonyElixir is not a supported product and is presented as-is.",
		"To proceed, start with `" + acknowledgementFlag + "` CLI argument",
	}
	width := 0
	for _, line := range lines {
		width = max(width, len(line))
	}
	border := strings.Repeat("─", width+2)
	var content []string
	content = append(content, "╭"+border+"╮")
	content = append(content, "│ "+strings.Repeat(" ", width)+" │")
	for _, line := range lines {
		content = append(content, "│ "+line+strings.Repeat(" ", width-len(line))+" │")
	}
	content = append(content, "│ "+strings.Repeat(" ", width)+" │")
	content = append(content, "╰"+border+"╯")
	return "\x1b[31m\x1b[1m" + strings.Join(content, "\n") + "\x1b[0m"
}
