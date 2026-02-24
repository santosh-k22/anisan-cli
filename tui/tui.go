// Package tui provides the primary terminal user interface implementation.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Options encapsulates the runtime configuration for the terminal user interface.
type Options struct {
	Continue bool
}

// Run initializes and executes the primary Bubble Tea application loop.
func Run(options *Options) error {
	bubble := newBubble(options)

	if options.Continue {
		_, err := bubble.loadHistory()
		if err != nil {
			return err
		}
		bubble.newState(historyState)
	} else {
		bubble.newState(sourcesState)
	}

	_, err := tea.NewProgram(bubble, tea.WithAltScreen()).Run()
	return err
}
