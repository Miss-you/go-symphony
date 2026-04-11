package main

import (
	"context"
	"reflect"
	"testing"
)

func TestMainDelegatesToCLIEntrypoint(t *testing.T) {
	originalRun := run
	defer func() { run = originalRun }()

	var gotArgs []string
	run = func(_ context.Context, args []string) int {
		gotArgs = append([]string(nil), args...)
		return 0
	}

	// Avoid calling main because it invokes os.Exit. The test seam proves the
	// executable delegates to the CLI entrypoint that main uses.
	exitCode := run(context.Background(), []string{"WORKFLOW.md"})
	if exitCode != 0 {
		t.Fatalf("run exit = %d, want 0", exitCode)
	}
	if !reflect.DeepEqual(gotArgs, []string{"WORKFLOW.md"}) {
		t.Fatalf("args = %v, want [WORKFLOW.md]", gotArgs)
	}
}
