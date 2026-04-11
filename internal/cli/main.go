package cli

import (
	"context"
	"fmt"
	"os"
)

func Main(ctx context.Context, args []string) int {
	if ctx == nil {
		ctx = context.Background()
	}
	workflowPath := ""
	if len(args) > 0 {
		workflowPath = args[0]
	}
	runtime, err := StartRuntime(ctx, RuntimeOptions{WorkflowPath: workflowPath})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = runtime.Close() }()
	<-ctx.Done()
	return 0
}
