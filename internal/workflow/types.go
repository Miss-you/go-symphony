package workflow

import "github.com/Miss-you/go-symphony/internal/codex"

type ID string

const CompatLinearDefault ID = "compat_linear_default"

type Bundle struct {
	ID             ID
	PromptTemplate string
	DynamicTools   []codex.ToolSpec
	ToolHandler    codex.ToolHandler
}
