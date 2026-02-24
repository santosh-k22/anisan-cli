// Package tui provides the primary terminal user interface implementation.
package tui

import (
	"github.com/anisan-cli/anisan/color"
	"github.com/anisan-cli/anisan/style"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
)

// statefulKeymap defines the keyboard interactions available within various application states.
type statefulKeymap struct {
	state state

	quit, forceQuit,
	selectOne, selectAll, selectVolume, clearSelection,
	acceptSearchSuggestion,
	anilistSelect,
	malSelect,
	remove,
	confirm,
	openURL,
	read,
	back,
	filter,
	up, down, left, right,
	top, bottom,
	nextEp, prevEp, playPause, replay,
	manualID, saveAsDefault, changeSource,
	showHelp key.Binding
}

// setState updates the active keymap configuration to match the specified application state.
func (k *statefulKeymap) setState(newState state) {
	k.state = newState
}

func newStatefulKeymap() *statefulKeymap {
	return &statefulKeymap{
		quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		forceQuit: key.NewBinding(
			key.WithKeys("ctrl+c", "ctrl+d"),
			key.WithHelp("ctrl+c", "quit"),
		),
		remove: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "remove"),
		),
		selectOne: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "select one"),
		),
		selectAll: key.NewBinding(
			key.WithKeys("ctrl+a", "tab", "*"),
			key.WithHelp("tab", "select all"),
		),
		selectVolume: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "select volume"),
		),
		clearSelection: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("backspace", "clear selection"),
		),
		confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		openURL: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open url"),
		),
		read: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp(style.Fg(color.Orange)("enter"), style.Fg(color.Orange)("play")),
		),
		acceptSearchSuggestion: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "accept search suggestion"),
		),
		anilistSelect: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "anilist"),
		),
		malSelect: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "select mal anime"),
		),
		back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑", "up"),
		),
		down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓", "down"),
		),
		left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←", "left"),
		),
		right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→", "right"),
		),
		top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		nextEp: key.NewBinding(
			key.WithKeys("n", "right"),
			key.WithHelp("n", "next episode"),
		),
		prevEp: key.NewBinding(
			key.WithKeys("p", "left"),
			key.WithHelp("p", "prev episode"),
		),
		playPause: key.NewBinding(
			key.WithKeys("space"),
			key.WithHelp("space", "pause/resume"),
		),
		replay: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "replay"),
		),
		showHelp: key.NewBinding(
			key.WithKeys("?", "h"),
			key.WithHelp("?", "help"),
		),
		manualID: key.NewBinding(
			key.WithKeys("I"),
			key.WithHelp("I", "manual id"),
		),
		saveAsDefault: key.NewBinding(
			key.WithKeys("S", "ctrl+s"),
			key.WithHelp("S", "save as default"),
		),
		changeSource: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "change source"),
		),
	}
}

func (k *statefulKeymap) help() ([]key.Binding, []key.Binding) {
	h := func(bindings ...key.Binding) []key.Binding {
		return bindings
	}

	to2 := func(a []key.Binding) ([]key.Binding, []key.Binding) {
		return a, a
	}

	switch k.state {
	case loadingState:
		return to2(h(k.forceQuit, k.back))
	case historyState:
		return to2(h(k.confirm, k.remove, k.back, k.openURL))
	case sourcesState:
		search := withDescription(k.confirm, "search with selected")
		return h(k.selectOne, k.selectAll, search, k.saveAsDefault), h(k.selectOne, k.selectAll, k.clearSelection, search, k.saveAsDefault)
	case searchState:
		return to2(h(k.confirm, k.acceptSearchSuggestion, k.changeSource, k.forceQuit))
	case animesState:
		return to2(h(k.confirm, k.changeSource, k.back))
	case episodesState:
		return h(k.confirm, k.malSelect, k.anilistSelect, k.manualID, k.back), h(k.confirm, k.selectOne, k.selectAll, k.clearSelection, k.openURL, k.selectVolume, k.anilistSelect, k.malSelect, k.back)
	case anilistSelectState:
		return to2(h(k.confirm, k.openURL, k.back))
	case readState:
		return to2(h(k.back, k.forceQuit))
	case errorState:
		return to2(h(k.back, k.quit))
	default:
		return to2(h())
	}
}

func (k *statefulKeymap) ShortHelp() []key.Binding {
	short, _ := k.help()
	return short
}

func (k *statefulKeymap) FullHelp() [][]key.Binding {
	_, full := k.help()
	return [][]key.Binding{full}
}

func (k *statefulKeymap) forList() list.KeyMap {
	return list.KeyMap{
		CursorUp:   k.up,
		CursorDown: k.down,
		NextPage:   k.right,
		PrevPage:   k.left,
		GoToStart:  k.top,
		GoToEnd:    k.bottom,
		// Filter:               k.filter, // DISABLED
		ClearFilter:          k.back,
		CancelWhileFiltering: k.back,
		AcceptWhileFiltering: k.confirm,
		ShowFullHelp:         k.showHelp,
		CloseFullHelp:        k.showHelp,
		Quit:                 k.quit,
		ForceQuit:            k.forceQuit,
	}
}

func withDescription(k key.Binding, description string) key.Binding {
	return key.NewBinding(
		key.WithKeys(k.Keys()...),
		key.WithHelp(k.Help().Key, description),
	)
}
