// Package tui provides the primary terminal user interface implementation.
package tui

import (
	"fmt"

	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/provider"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
)

// Init initializes the terminal user interface, triggering initial data loads and hardware checks.
func (b *statefulBubble) Init() tea.Cmd {
	// Auto-load sources if DefaultSources config is set
	if names := viper.GetStringSlice(key.DefaultSources); b.state != historyState && len(names) != 0 {
		var providers []*provider.Provider

		for _, name := range names {
			p, ok := provider.Get(name)
			if !ok {
				b.raiseError(fmt.Errorf("provider %s not found", name))
				return nil
			}

			providers = append(providers, p)
		}

		// If exactly one source is loaded, inject it into the Anime list title
		if len(providers) == 1 {
			b.animesC.Title = fmt.Sprintf("Anime Results - %s", providers[0].Name)
		}

		b.setState(loadingState)
		return tea.Batch(b.startLoading(), b.loadSources(providers), b.waitForSourcesLoaded(), provider.UpdateScrapers())
	}

	return tea.Batch(textinput.Blink, b.loadProviders(), provider.UpdateScrapers())
}
