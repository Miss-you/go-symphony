package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolvedWorkflowPathDefaultsToCurrentWorkingDirectory(t *testing.T) {
	tempDir := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	got, err := resolvedWorkflowPath("")
	if err != nil {
		t.Fatalf("resolvedWorkflowPath: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd after chdir: %v", err)
	}

	want := filepath.Join(cwd, workflowFileName)
	if got != want {
		t.Fatalf("path mismatch: got %q want %q", got, want)
	}
}

func TestResolvedWorkflowPathKeepsExplicitOverride(t *testing.T) {
	t.Parallel()

	explicit := "tmp/custom/WORKFLOW.md"

	got, err := resolvedWorkflowPath(explicit)
	if err != nil {
		t.Fatalf("resolvedWorkflowPath: %v", err)
	}

	if got != explicit {
		t.Fatalf("path mismatch: got %q want %q", got, explicit)
	}
}

func TestLoadPromptOnlyWorkflow(t *testing.T) {
	t.Parallel()

	path := writeWorkflowFile(t, "PROMPT_ONLY_WORKFLOW.md", "Prompt only\n")

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(got.Config) != 0 {
		t.Fatalf("expected empty config, got %#v", got.Config)
	}
	if got.Prompt != "Prompt only" || got.PromptTemplate != "Prompt only" {
		t.Fatalf("unexpected prompt payload: %#v", got)
	}
}

func TestLoadUnterminatedFrontMatter(t *testing.T) {
	t.Parallel()

	path := writeWorkflowFile(t, "UNTERMINATED_WORKFLOW.md", "---\ntracker:\n  kind: linear\n")

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tracker, ok := got.Config["tracker"].(map[string]any)
	if !ok {
		t.Fatalf("expected tracker map, got %#v", got.Config["tracker"])
	}
	if tracker["kind"] != "linear" {
		t.Fatalf("unexpected tracker kind: %#v", tracker["kind"])
	}
	if got.PromptTemplate != "" {
		t.Fatalf("expected empty prompt, got %q", got.PromptTemplate)
	}
}

func TestLoadRejectsNonMapFrontMatter(t *testing.T) {
	t.Parallel()

	path := writeWorkflowFile(t, "INVALID_FRONT_MATTER_WORKFLOW.md", "---\n- not-a-map\n---\nPrompt body\n")

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}

	var loadErr *LoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("expected LoadError, got %T", err)
	}
	if loadErr.Code != ErrWorkflowFrontMatterNotMap {
		t.Fatalf("unexpected code: got %q want %q", loadErr.Code, ErrWorkflowFrontMatterNotMap)
	}
}

func TestLoadMissingWorkflowFileReturnsTypedError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "MISSING_WORKFLOW.md")

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}

	var loadErr *LoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("expected LoadError, got %T", err)
	}
	if loadErr.Code != ErrMissingWorkflowFile {
		t.Fatalf("unexpected code: got %q want %q", loadErr.Code, ErrMissingWorkflowFile)
	}
	if loadErr.Path != path {
		t.Fatalf("unexpected path: got %q want %q", loadErr.Path, path)
	}
}

func TestLoadInvalidYAMLFrontMatterReturnsTypedError(t *testing.T) {
	t.Parallel()

	path := writeWorkflowFile(t, "INVALID_YAML_WORKFLOW.md", "---\ntracker: [\n---\nPrompt body\n")

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}

	var loadErr *LoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("expected LoadError, got %T", err)
	}
	if loadErr.Code != ErrWorkflowParse {
		t.Fatalf("unexpected code: got %q want %q", loadErr.Code, ErrWorkflowParse)
	}
}

func TestLoadTrimsPromptBody(t *testing.T) {
	t.Parallel()

	path := writeWorkflowFile(t, "TRIM_WORKFLOW.md", "---\ntracker:\n  kind: linear\n---\n\n  Prompt body  \n")

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.PromptTemplate != "Prompt body" {
		t.Fatalf("unexpected prompt: got %q want %q", got.PromptTemplate, "Prompt body")
	}
}

func TestEffectivePromptTemplateReturnsDefaultForBlankPrompt(t *testing.T) {
	t.Parallel()

	workflow := Workflow{PromptTemplate: "  \n"}

	got := EffectivePromptTemplate(workflow)

	if got == "" {
		t.Fatal("expected non-empty default prompt")
	}
	if got == workflow.PromptTemplate {
		t.Fatalf("expected fallback prompt, got original %q", got)
	}
}

func writeWorkflowFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}
	return path
}
