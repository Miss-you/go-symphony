package runner

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestExecutorRunsLocalCommandInRequestedDirectory(t *testing.T) {
	t.Parallel()

	var calls []processCall
	executor := NewExecutor(WithProcessStarter(func(_ context.Context, name string, args []string, dir string) ([]byte, error) {
		calls = append(calls, processCall{name: name, args: args, dir: dir})
		return []byte("ok\n"), nil
	}))

	result, err := executor.RunCommand(context.Background(), CommandRequest{
		Dir:     "/tmp/work",
		Command: "pwd",
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("RunCommand returned error: %v", err)
	}
	if result.Output != "ok\n" || result.Status != 0 {
		t.Fatalf("result = %+v, want output ok and status 0", result)
	}
	want := processCall{name: "sh", args: []string{"-lc", "pwd"}, dir: "/tmp/work"}
	if !reflect.DeepEqual(calls, []processCall{want}) {
		t.Fatalf("process calls = %+v, want %+v", calls, []processCall{want})
	}
}

func TestExecutorNormalizesExitStatusAndTimeout(t *testing.T) {
	t.Parallel()

	exitErr := &ExitError{Status: 17}
	executor := NewExecutor(WithProcessStarter(func(_ context.Context, _ string, _ []string, _ string) ([]byte, error) {
		return []byte("failed"), exitErr
	}))

	result, err := executor.RunCommand(context.Background(), CommandRequest{Command: "exit 17", Timeout: time.Second})
	if err != nil {
		t.Fatalf("RunCommand returned error for process exit: %v", err)
	}
	if result.Status != 17 || result.Output != "failed" {
		t.Fatalf("result = %+v, want status 17 with output", result)
	}

	timeoutExecutor := NewExecutor(WithProcessStarter(func(ctx context.Context, _ string, _ []string, _ string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}))
	_, err = timeoutExecutor.RunCommand(context.Background(), CommandRequest{Command: "sleep", Timeout: time.Nanosecond})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout error = %v, want context deadline exceeded", err)
	}
}

func TestSSHCommandArguments(t *testing.T) {
	t.Setenv("SYMPHONY_SSH_CONFIG", "/tmp/ssh_config")

	args := buildSSHArgs("user@worker.example:2222", "/tmp/work", "printf ok", os.LookupEnv)

	want := []string{
		"-T",
		"-F", "/tmp/ssh_config",
		"-p", "2222",
		"user@worker.example",
		"bash -lc 'cd /tmp/work && printf ok'",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("ssh args = %#v, want %#v", args, want)
	}
}

func TestSSHCommandArgumentsHandleBracketedIPv6(t *testing.T) {
	t.Setenv("SYMPHONY_SSH_CONFIG", "")

	args := buildSSHArgs("user@[2001:db8::1]:2200", "", "printf ok", os.LookupEnv)

	got := strings.Join(args, " ")
	if !strings.Contains(got, "-p 2200") {
		t.Fatalf("ssh args = %#v, want parsed port", args)
	}
	if !strings.Contains(got, "user@[2001:db8::1]") {
		t.Fatalf("ssh args = %#v, want user-prefixed bracketed IPv6 host preserved", args)
	}
}

func TestHostSelection(t *testing.T) {
	t.Parallel()

	capOne := 1
	cases := []struct {
		name      string
		policy    HostSelection
		preferred string
		loads     []HostLoad
		wantHost  string
		wantOK    bool
	}{
		{
			name:     "local when no hosts configured",
			policy:   HostSelection{},
			wantHost: "",
			wantOK:   true,
		},
		{
			name:      "eligible preferred host wins",
			policy:    HostSelection{Hosts: []string{"worker-a", "worker-b"}, MaxPerHost: &capOne},
			preferred: "worker-b",
			loads:     []HostLoad{{Host: "worker-a", Running: 1}},
			wantHost:  "worker-b",
			wantOK:    true,
		},
		{
			name:      "unknown preferred falls back to least loaded",
			policy:    HostSelection{Hosts: []string{"worker-a", "worker-b"}},
			preferred: "worker-x",
			loads:     []HostLoad{{Host: "worker-a", Running: 2}, {Host: "worker-b", Running: 1}},
			wantHost:  "worker-b",
			wantOK:    true,
		},
		{
			name:     "ties use configured host order",
			policy:   HostSelection{Hosts: []string{"worker-a", "worker-b"}},
			loads:    []HostLoad{{Host: "worker-b", Running: 1}, {Host: "worker-a", Running: 1}},
			wantHost: "worker-a",
			wantOK:   true,
		},
		{
			name:   "all hosts full rejects",
			policy: HostSelection{Hosts: []string{"worker-a", "worker-b"}, MaxPerHost: &capOne},
			loads:  []HostLoad{{Host: "worker-a", Running: 1}, {Host: "worker-b", Running: 1}},
			wantOK: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotHost, gotOK := tt.policy.Select(tt.preferred, tt.loads)
			if gotHost != tt.wantHost || gotOK != tt.wantOK {
				t.Fatalf("Select() = (%q, %v), want (%q, %v)", gotHost, gotOK, tt.wantHost, tt.wantOK)
			}
		})
	}
}

type processCall struct {
	name string
	args []string
	dir  string
}
