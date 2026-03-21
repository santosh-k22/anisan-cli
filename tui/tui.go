package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Options struct {
	Continue bool
}

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
