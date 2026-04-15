package linearprobe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Miss-you/go-symphony/internal/config"
	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/tracker"
)

type Result struct {
	Settings   config.Settings
	Candidates []domain.WorkItem
	Terminal   []domain.WorkItem
	Refresh    []domain.WorkItem
}

func Run(ctx context.Context, settings config.Settings, reader tracker.TrackerReader, refreshIDs []string) (Result, error) {
	if reader == nil {
		return Result{}, errors.New("nil tracker reader")
	}
	candidates, err := reader.ListCandidates(ctx)
	if err != nil {
		return Result{}, err
	}
	terminal, err := reader.ListByStates(ctx, settings.Provider.TerminalStates)
	if err != nil {
		return Result{}, err
	}
	if len(refreshIDs) == 0 && len(candidates) > 0 {
		refreshIDs = []string{candidates[0].ID}
	}
	refresh, err := reader.RefreshByIDs(ctx, refreshIDs)
	if err != nil {
		return Result{}, err
	}
	return Result{Settings: settings, Candidates: candidates, Terminal: terminal, Refresh: refresh}, nil
}

func Render(w io.Writer, result Result, limit int) {
	_, _ = fmt.Fprintln(w, "Linear probe")
	_, _ = fmt.Fprintf(w, "project: %s\n", result.Settings.Provider.Project)
	_, _ = fmt.Fprintf(w, "active_states: %s\n", strings.Join(result.Settings.Provider.ActiveStates, ", "))
	_, _ = fmt.Fprintf(w, "terminal_states: %s\n", strings.Join(result.Settings.Provider.TerminalStates, ", "))
	renderItems(w, "candidates", result.Candidates, limit)
	renderItems(w, "terminal", result.Terminal, limit)
	renderItems(w, "refresh", result.Refresh, limit)
}

func renderItems(w io.Writer, label string, items []domain.WorkItem, limit int) {
	_, _ = fmt.Fprintf(w, "%s: %d\n", label, len(items))
	shown := len(items)
	if limit > 0 && shown > limit {
		shown = limit
	}
	for i := 0; i < shown; i++ {
		item := items[i]
		_, _ = fmt.Fprintf(w, "- %s | %s | %s\n", item.Identifier, item.State, item.Title)
	}
	if shown < len(items) {
		_, _ = fmt.Fprintf(w, "... %d more\n", len(items)-shown)
	}
}
