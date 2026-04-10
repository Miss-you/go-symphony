package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const workflowFileName = "WORKFLOW.md"

const defaultPromptTemplate = `You are working on a Linear issue.

Identifier: {{ issue.identifier }}
Title: {{ issue.title }}

Body:
{% if issue.description %}
{{ issue.description }}
{% else %}
No description provided.
{% endif %}`

type ErrorCode string

const (
	ErrMissingWorkflowFile       ErrorCode = "missing_workflow_file"
	ErrWorkflowParse            ErrorCode = "workflow_parse_error"
	ErrWorkflowFrontMatterNotMap ErrorCode = "workflow_front_matter_not_a_map"
)

type LoadError struct {
	Code ErrorCode
	Path string
	Err  error
}

func (e *LoadError) Error() string {
	switch e.Code {
	case ErrMissingWorkflowFile:
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Path, e.Err)
	case ErrWorkflowFrontMatterNotMap:
		return string(e.Code)
	default:
		if e.Err == nil {
			return string(e.Code)
		}
		return fmt.Sprintf("%s: %v", e.Code, e.Err)
	}
}

func (e *LoadError) Unwrap() error {
	return e.Err
}

type Workflow struct {
	Path           string
	Config         map[string]any
	Prompt         string
	PromptTemplate string
}

func Load(path string) (Workflow, error) {
	resolved, err := resolvedWorkflowPath(path)
	if err != nil {
		return Workflow{}, err
	}

	content, err := os.ReadFile(resolved)
	if err != nil {
		return Workflow{}, &LoadError{Code: ErrMissingWorkflowFile, Path: resolved, Err: err}
	}

	return Parse(content, resolved)
}

func Parse(content []byte, path string) (Workflow, error) {
	frontMatterLines, promptLines := splitFrontMatter(string(content))

	config, err := decodeFrontMatter(frontMatterLines)
	if err != nil {
		return Workflow{}, err
	}

	prompt := strings.TrimSpace(strings.Join(promptLines, "\n"))

	return Workflow{
		Path:           path,
		Config:         config,
		Prompt:         prompt,
		PromptTemplate: prompt,
	}, nil
}

func EffectivePromptTemplate(workflow Workflow) string {
	if strings.TrimSpace(workflow.PromptTemplate) == "" {
		return defaultPromptTemplate
	}
	return workflow.PromptTemplate
}

func resolvedWorkflowPath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Join(cwd, workflowFileName), nil
}

func splitFrontMatter(content string) ([]string, []string) {
	lines := splitLines(content)
	if len(lines) == 0 || lines[0] != "---" {
		return nil, lines
	}

	frontMatter := make([]string, 0, len(lines))
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			return frontMatter, lines[i+1:]
		}
		frontMatter = append(frontMatter, lines[i])
	}

	return frontMatter, nil
}

func splitLines(content string) []string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.Split(normalized, "\n")
}

func decodeFrontMatter(lines []string) (map[string]any, error) {
	yamlContent := strings.TrimSpace(strings.Join(lines, "\n"))
	if yamlContent == "" {
		return map[string]any{}, nil
	}

	var raw any
	if err := yaml.Unmarshal([]byte(yamlContent), &raw); err != nil {
		return nil, &LoadError{Code: ErrWorkflowParse, Err: err}
	}

	config, ok := normalizeYAMLValue(raw).(map[string]any)
	if !ok {
		return nil, &LoadError{Code: ErrWorkflowFrontMatterNotMap}
	}

	return config, nil
}

func normalizeYAMLValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, child := range typed {
			normalized[key] = normalizeYAMLValue(child)
		}
		return normalized
	case map[any]any:
		normalized := make(map[string]any, len(typed))
		for key, child := range typed {
			normalized[fmt.Sprint(key)] = normalizeYAMLValue(child)
		}
		return normalized
	case []any:
		normalized := make([]any, len(typed))
		for i, child := range typed {
			normalized[i] = normalizeYAMLValue(child)
		}
		return normalized
	default:
		return typed
	}
}

func writeFile(path string, reader io.Reader) error {
	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(reader); err != nil {
		return err
	}
	return os.WriteFile(path, buffer.Bytes(), 0o644)
}
