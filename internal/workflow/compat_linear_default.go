package workflow

import (
	"github.com/Miss-you/go-symphony/internal/config"
	linearbridge "github.com/Miss-you/go-symphony/internal/toolbridge/linear"
)

func CompatLinearDefaultBundle(raw config.Workflow, settings config.Settings) (Bundle, error) {
	bridge, err := linearbridge.New(settings.Provider, nil)
	if err != nil {
		return Bundle{}, err
	}

	return Bundle{
		ID:             CompatLinearDefault,
		PromptTemplate: config.EffectivePromptTemplate(raw),
		DynamicTools:   bridge.ToolSpecs(),
		ToolHandler:    bridge,
	}, nil
}
