package main

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

var errUsage = errors.New("usage")

type linearArgs struct {
	workflowPath string
	limit        int
	refreshIDs   []string
	onlyIssue    string
}

type runArgs struct {
	workflowPath string
	ack          bool
	onlyIssue    string
	portOverride *int
	timeout      time.Duration
}

func parseLinearArgs(args []string) (linearArgs, error) {
	parsed := linearArgs{workflowPath: "WORKFLOW.md", limit: 10}
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--limit":
			value, ok := nextValue(args, &i)
			if !ok {
				return linearArgs{}, errUsage
			}
			limit, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || limit < 0 {
				return linearArgs{}, errUsage
			}
			parsed.limit = limit
		case strings.HasPrefix(arg, "--limit="):
			limit, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(arg, "--limit=")))
			if err != nil || limit < 0 {
				return linearArgs{}, errUsage
			}
			parsed.limit = limit
		case arg == "--refresh-id":
			value, ok := nextValue(args, &i)
			if !ok || strings.TrimSpace(value) == "" || strings.HasPrefix(value, "--") {
				return linearArgs{}, errUsage
			}
			parsed.refreshIDs = append(parsed.refreshIDs, strings.TrimSpace(value))
		case strings.HasPrefix(arg, "--refresh-id="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--refresh-id="))
			if value == "" {
				return linearArgs{}, errUsage
			}
			parsed.refreshIDs = append(parsed.refreshIDs, value)
		case arg == "--only-issue":
			value, ok := nextValue(args, &i)
			if !ok || strings.TrimSpace(value) == "" || strings.HasPrefix(value, "--") {
				return linearArgs{}, errUsage
			}
			parsed.onlyIssue = strings.TrimSpace(value)
		case strings.HasPrefix(arg, "--only-issue="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--only-issue="))
			if value == "" {
				return linearArgs{}, errUsage
			}
			parsed.onlyIssue = value
		case strings.HasPrefix(arg, "--"):
			return linearArgs{}, errUsage
		default:
			positionals = append(positionals, arg)
		}
	}
	if len(positionals) > 1 {
		return linearArgs{}, errUsage
	}
	if len(positionals) == 1 {
		parsed.workflowPath = positionals[0]
	}
	return parsed, nil
}

func parseRunArgs(args []string) (runArgs, error) {
	parsed := runArgs{workflowPath: "WORKFLOW.md", timeout: 10 * time.Minute}
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == ackFlag:
			parsed.ack = true
		case arg == "--only-issue":
			value, ok := nextValue(args, &i)
			if !ok || strings.TrimSpace(value) == "" || strings.HasPrefix(value, "--") {
				return runArgs{}, errUsage
			}
			parsed.onlyIssue = strings.TrimSpace(value)
		case strings.HasPrefix(arg, "--only-issue="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--only-issue="))
			if value == "" {
				return runArgs{}, errUsage
			}
			parsed.onlyIssue = value
		case arg == "--port":
			value, ok := nextValue(args, &i)
			if !ok {
				return runArgs{}, errUsage
			}
			port, err := parsePort(value)
			if err != nil {
				return runArgs{}, errUsage
			}
			parsed.portOverride = &port
		case strings.HasPrefix(arg, "--port="):
			port, err := parsePort(strings.TrimPrefix(arg, "--port="))
			if err != nil {
				return runArgs{}, errUsage
			}
			parsed.portOverride = &port
		case arg == "--timeout":
			value, ok := nextValue(args, &i)
			if !ok {
				return runArgs{}, errUsage
			}
			timeout, err := parseTimeout(value)
			if err != nil {
				return runArgs{}, errUsage
			}
			parsed.timeout = timeout
		case strings.HasPrefix(arg, "--timeout="):
			timeout, err := parseTimeout(strings.TrimPrefix(arg, "--timeout="))
			if err != nil {
				return runArgs{}, errUsage
			}
			parsed.timeout = timeout
		case strings.HasPrefix(arg, "--"):
			return runArgs{}, errUsage
		default:
			positionals = append(positionals, arg)
		}
	}
	if len(positionals) > 1 {
		return runArgs{}, errUsage
	}
	if len(positionals) == 1 {
		parsed.workflowPath = positionals[0]
	}
	return parsed, nil
}

func nextValue(args []string, index *int) (string, bool) {
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

func parseTimeout(value string) (time.Duration, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "0" {
		return 0, nil
	}
	timeout, err := time.ParseDuration(trimmed)
	if err != nil || timeout < 0 {
		return 0, errUsage
	}
	return timeout, nil
}
