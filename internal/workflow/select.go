package workflow

import (
	"errors"
	"fmt"

	"github.com/Miss-you/go-symphony/internal/config"
)

var ErrUnsupportedProvider = errors.New("unsupported workflow provider")

type UnsupportedProviderError struct {
	Provider config.ProviderKind
}

func (e *UnsupportedProviderError) Error() string {
	return fmt.Sprintf("%v: %s", ErrUnsupportedProvider, e.Provider)
}

func (e *UnsupportedProviderError) Unwrap() error {
	return ErrUnsupportedProvider
}

func Select(raw config.Workflow, settings config.Settings) (Bundle, error) {
	switch settings.Provider.Kind {
	case config.ProviderLinear:
		return CompatLinearDefaultBundle(raw, settings)
	default:
		return Bundle{}, &UnsupportedProviderError{Provider: settings.Provider.Kind}
	}
}
