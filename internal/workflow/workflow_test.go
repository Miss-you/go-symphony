package workflow

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/Miss-you/go-symphony/internal/codex"
	"github.com/Miss-you/go-symphony/internal/config"
)

func TestSelectReturnsCompatLinearDefaultForLinearProvider(t *testing.T) {
	raw := config.Workflow{PromptTemplate: "Work on {{ issue.identifier }}"}
	settings := config.Settings{
		Provider: config.ProviderSettings{
			Kind: config.ProviderLinear,
		},
	}

	bundle, err := Select(raw, settings)
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}

	if bundle.ID != CompatLinearDefault {
		t.Fatalf("bundle ID = %q, want %q", bundle.ID, CompatLinearDefault)
	}
	if bundle.PromptTemplate != raw.PromptTemplate {
		t.Fatalf("PromptTemplate = %q, want %q", bundle.PromptTemplate, raw.PromptTemplate)
	}
}

func TestSelectReturnsUnsupportedProviderError(t *testing.T) {
	settings := config.Settings{
		Provider: config.ProviderSettings{
			Kind: config.ProviderMemory,
		},
	}

	bundle, err := Select(config.Workflow{}, settings)
	if !errors.Is(err, ErrUnsupportedProvider) {
		t.Fatalf("Select error = %v, want ErrUnsupportedProvider", err)
	}
	if bundle.ID != "" {
		t.Fatalf("bundle ID = %q, want empty bundle on unsupported provider", bundle.ID)
	}
}

func TestCompatLinearDefaultUsesEffectivePromptTemplate(t *testing.T) {
	bundle, err := CompatLinearDefaultBundle(config.Workflow{}, linearSettings())
	if err != nil {
		t.Fatalf("CompatLinearDefaultBundle returned error: %v", err)
	}

	want := config.EffectivePromptTemplate(config.Workflow{})
	if bundle.PromptTemplate != want {
		t.Fatalf("PromptTemplate = %q, want default template %q", bundle.PromptTemplate, want)
	}

	raw := config.Workflow{PromptTemplate: "Custom workflow body"}
	bundle, err = CompatLinearDefaultBundle(raw, linearSettings())
	if err != nil {
		t.Fatalf("CompatLinearDefaultBundle returned error: %v", err)
	}
	if bundle.PromptTemplate != raw.PromptTemplate {
		t.Fatalf("PromptTemplate = %q, want %q", bundle.PromptTemplate, raw.PromptTemplate)
	}
}

func TestCompatLinearDefaultWiresLinearToolBridgeDirectly(t *testing.T) {
	bundle, err := CompatLinearDefaultBundle(config.Workflow{}, linearSettings())
	if err != nil {
		t.Fatalf("CompatLinearDefaultBundle returned error: %v", err)
	}

	if len(bundle.DynamicTools) != 1 {
		t.Fatalf("DynamicTools len = %d, want 1", len(bundle.DynamicTools))
	}
	if got := bundle.DynamicTools[0].Name; got != "linear_graphql" {
		t.Fatalf("dynamic tool name = %q, want linear_graphql", got)
	}
	if bundle.ToolHandler == nil {
		t.Fatal("ToolHandler is nil")
	}

	var sessionOptions codex.SessionOptions
	sessionOptions.Config.DynamicTools = bundle.DynamicTools
	sessionOptions.ToolHandler = bundle.ToolHandler

	result, err := bundle.ToolHandler.HandleTool(context.Background(), codex.ToolCall{
		Name:      "unknown_tool",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("HandleTool returned error: %v", err)
	}
	if result.Success {
		t.Fatal("unknown tool result succeeded, want Linear bridge failure response")
	}
	if len(result.ContentItems) != 1 || !strings.Contains(result.ContentItems[0].Text, "linear_graphql") {
		t.Fatalf("unknown tool response = %#v, want supported linear_graphql payload", result.ContentItems)
	}
}

func TestWorkflowPackageDependencyBoundary(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", ".")
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps . failed: %v", err)
	}

	forbidden := []string{
		"github.com/Miss-you/go-symphony/internal/orchestrator",
		"github.com/Miss-you/go-symphony/internal/tracker",
		"github.com/Miss-you/go-symphony/internal/workspace",
		"github.com/Miss-you/go-symphony/internal/runner",
		"github.com/Miss-you/go-symphony/internal/domain",
	}
	deps := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, dep := range deps {
		for _, forbiddenDep := range forbidden {
			if dep == forbiddenDep || strings.HasPrefix(dep, forbiddenDep+"/") {
				t.Fatalf("internal/workflow dependency graph includes forbidden dependency %q", dep)
			}
		}
	}
}

func linearSettings() config.Settings {
	return config.Settings{
		Provider: config.ProviderSettings{
			Kind: config.ProviderLinear,
		},
	}
}
