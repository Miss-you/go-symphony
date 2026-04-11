package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type CommandRequest struct {
	Host    string
	Dir     string
	Command string
	Timeout time.Duration
}

type CommandResult struct {
	Output string
	Status int
}

type Executor interface {
	RunCommand(context.Context, CommandRequest) (CommandResult, error)
}

type ProcessStarter func(ctx context.Context, name string, args []string, dir string) ([]byte, error)

type ExecutorOption func(*CommandExecutor)

type CommandExecutor struct {
	start     ProcessStarter
	lookupEnv func(string) (string, bool)
}

type ExitError struct {
	Status int
	Err    error
}

func (e *ExitError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "process exited with status " + strconv.Itoa(e.Status)
}

func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewExecutor(options ...ExecutorOption) *CommandExecutor {
	executor := &CommandExecutor{
		start:     defaultProcessStarter,
		lookupEnv: os.LookupEnv,
	}
	for _, option := range options {
		option(executor)
	}
	return executor
}

func WithProcessStarter(starter ProcessStarter) ExecutorOption {
	return func(executor *CommandExecutor) {
		if starter != nil {
			executor.start = starter
		}
	}
}

func WithLookupEnv(lookup func(string) (string, bool)) ExecutorOption {
	return func(executor *CommandExecutor) {
		if lookup != nil {
			executor.lookupEnv = lookup
		}
	}
}

func (e *CommandExecutor) RunCommand(ctx context.Context, req CommandRequest) (CommandResult, error) {
	if e == nil {
		e = NewExecutor()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	name := "sh"
	args := []string{"-lc", req.Command}
	dir := req.Dir
	if strings.TrimSpace(req.Host) != "" {
		name = "ssh"
		args = buildSSHArgs(req.Host, req.Dir, req.Command, e.lookupEnv)
		dir = ""
	}

	output, err := e.start(ctx, name, args, dir)
	if ctx.Err() != nil {
		return CommandResult{}, ctx.Err()
	}
	if err == nil {
		return CommandResult{Output: string(output), Status: 0}, nil
	}

	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return CommandResult{Output: string(output), Status: exitErr.Status}, nil
	}
	return CommandResult{}, err
}

func defaultProcessStarter(ctx context.Context, name string, args []string, dir string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err == nil {
		return output, nil
	}
	var execExitErr *exec.ExitError
	if errors.As(err, &execExitErr) {
		return output, &ExitError{Status: execExitErr.ExitCode(), Err: err}
	}
	return output, err
}

func buildSSHArgs(host, dir, command string, lookupEnv func(string) (string, bool)) []string {
	target, port := splitHostPort(strings.TrimSpace(host))
	args := []string{"-T"}
	if lookupEnv != nil {
		if configPath, ok := lookupEnv("SYMPHONY_SSH_CONFIG"); ok && strings.TrimSpace(configPath) != "" {
			args = append(args, "-F", configPath)
		}
	}
	if port != "" {
		args = append(args, "-p", port)
	}
	remoteCommand := command
	if strings.TrimSpace(dir) != "" {
		remoteCommand = "cd " + shellQuote(dir) + " && " + command
	}
	args = append(args, target, "bash -lc "+shellQuote(remoteCommand))
	return args
}

func splitHostPort(host string) (string, string) {
	prefix := ""
	destination := host
	if at := strings.LastIndex(destination, "@"); at >= 0 {
		prefix = destination[:at+1]
		destination = destination[at+1:]
	}

	if strings.HasPrefix(destination, "[") {
		end := strings.Index(destination, "]")
		if end > 0 && len(destination) > end+1 && destination[end+1] == ':' {
			port := destination[end+2:]
			if isPort(port) {
				return prefix + destination[:end+1], port
			}
		}
		return host, ""
	}

	colon := strings.LastIndex(destination, ":")
	if colon <= 0 || colon == len(destination)-1 {
		return host, ""
	}
	if strings.Count(destination, ":") > 1 {
		return host, ""
	}
	port := destination[colon+1:]
	if !isPort(port) {
		return host, ""
	}
	return prefix + destination[:colon], port
}

func isPort(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if strings.IndexFunc(value, unsafeShellRune) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func unsafeShellRune(r rune) bool {
	return !safeShellRune(r)
}

func safeShellRune(r rune) bool {
	return r == '/' || r == '.' || r == '_' || r == '-' || ('0' <= r && r <= '9') || ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z')
}
