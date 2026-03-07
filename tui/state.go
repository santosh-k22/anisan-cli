// Package tui provides the primary terminal user interface implementation.
package tui

type state int

const (
	loadingState state = iota
	errorState
	historyState
	sourcesState
	searchState
	animesState
	episodesState
	trackerSelectState
	confirmState
	readState
	postWatchState
	manualIDState
)
