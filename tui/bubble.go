// Package tui provides the primary terminal user interface implementation.
package tui

import (
	"fmt"
	"time"

	"github.com/anisan-cli/anisan/anilist"
	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/internal/ui"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/mal"
	"github.com/anisan-cli/anisan/player"
	"github.com/anisan-cli/anisan/provider"
	"github.com/anisan-cli/anisan/source"
	"github.com/anisan-cli/anisan/style"
	"github.com/anisan-cli/anisan/util"
	"github.com/charmbracelet/bubbles/help"
	bubblesKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/spf13/viper"
)

// statefulBubble encapsulates the comprehensive application state, including component models and workflow tracking.
type statefulBubble struct {
	state         state
	statesHistory util.Stack[state]
	loading       bool
	busy          bool // Protects against rapid input during async ops

	keymap *statefulKeymap

	// components
	spinnerC   spinner.Model
	inputC     textinput.Model
	historyC   list.Model
	sourcesC   list.Model
	animesC    list.Model
	episodesC  list.Model
	anilistC   list.Model
	malListC   list.Model
	postWatchC list.Model
	progressC  progress.Model
	helpC      help.Model
	idInputC   textinput.Model // For manual MAL/Anilist ID override

	selectedProviders map[*provider.Provider]struct{}
	selectedSources   []source.Source
	selectedAnime     *source.Anime
	selectedEpisodes  map[*source.Episode]struct{} // set

	sourcesLoadedChannel        chan []source.Source
	foundAnimesChannel          chan []*source.Anime
	foundEpisodesChannel        chan []*source.Episode
	fetchedAnilistAnimesChannel chan []*anilist.Anime
	fetchedMALAnimesChannel     chan []mal.Anime
	fetchedMALUserListChannel   chan []mal.UserListEntry
	closestAnilistAnimeChannel  chan *anilist.Anime
	episodeReadChannel          chan struct{}
	errorChannel                chan error

	progressStatus string

	currentPlayingEpisode *source.Episode
	nextEpisodeToPlay     *source.Episode // Queued episode for seamless transitions
	mpvPlayer             player.Player
	anilistAnime          *anilist.Anime // Store linked anime
	lastError             error

	width, height    int
	searchSuggestion mo.Option[string]
	notifier         *ui.Model

	options *Options
}

// raiseError dispatches a terminal error and transitions the application to the failure view.
func (b *statefulBubble) raiseError(err error) {
	b.lastError = err
	b.newState(errorState)
}

// setState performs a synchronous transition of both the application workflow and its associated keymap.
func (b *statefulBubble) setState(s state) {
	b.state = s
	b.keymap.setState(s)
}

// newState facilitates an idempotent transition to a target state, recording the previous state in the navigation history when appropriate.
func (b *statefulBubble) newState(s state) {
	if b.state == s {
		return
	}

	// Do not push these states to history
	if !lo.Contains([]state{
		loadingState,
		readState,
		anilistSelectState,
	}, b.state) {
		b.statesHistory.Push(b.state)
	}

	b.setState(s)
}

// previousState restores the application to its immediate predecessor in the navigation stack.
func (b *statefulBubble) previousState() {
	if b.statesHistory.Len() > 0 {
		s := b.statesHistory.Pop()
		b.setState(s)
	}
}

// resize propagates terminal dimension changes to all child component models.
func (b *statefulBubble) resize(width, height int) {
	x, y := paddingStyle.GetFrameSize()
	xx, yy := listExtraPaddingStyle.GetFrameSize()

	styledWidth := width - x
	styledHeight := height - y

	listWidth := width - xx
	listHeight := height - yy

	b.historyC.SetSize(listWidth, listHeight)
	b.historyC.Help.Width = listWidth

	b.sourcesC.SetSize(listWidth, listHeight)
	b.sourcesC.Help.Width = listWidth

	b.animesC.SetSize(listWidth, listHeight)
	b.animesC.Help.Width = listWidth

	b.episodesC.SetSize(listWidth, listHeight)
	b.episodesC.Help.Width = listWidth

	b.anilistC.SetSize(listWidth, listHeight)
	b.anilistC.Help.Width = listWidth

	b.malListC.SetSize(listWidth, listHeight)
	b.malListC.Help.Width = listWidth

	b.postWatchC.SetSize(listWidth, listHeight)
	b.postWatchC.Help.Width = listWidth

	b.progressC.Width = listWidth
	b.idInputC.Width = listWidth

	b.width = styledWidth
	b.height = styledHeight
	b.helpC.Width = listWidth
}

// startLoading enters a concurrent loading state, initializing visual indicators across child components.
func (b *statefulBubble) startLoading() tea.Cmd {
	b.loading = true
	b.busy = true
	return tea.Batch(b.animesC.StartSpinner(), b.episodesC.StartSpinner(), b.malListC.StartSpinner())
}

// stopLoading exits the loading state and synchronizes child component visual indicators.
func (b *statefulBubble) stopLoading() tea.Cmd {
	b.loading = false
	b.busy = false
	b.animesC.StopSpinner()
	b.episodesC.StopSpinner()
	b.malListC.StopSpinner()
	return nil
}

// newBubble performs a complete initialization of the application's primary UI model.
func newBubble(options *Options) *statefulBubble {
	keymap := newStatefulKeymap()
	bubble := statefulBubble{
		statesHistory: util.Stack[state]{},
		keymap:        keymap,

		sourcesLoadedChannel:        make(chan []source.Source),
		foundAnimesChannel:          make(chan []*source.Anime),
		foundEpisodesChannel:        make(chan []*source.Episode),
		fetchedAnilistAnimesChannel: make(chan []*anilist.Anime),
		fetchedMALAnimesChannel:     make(chan []mal.Anime),
		fetchedMALUserListChannel:   make(chan []mal.UserListEntry),
		closestAnilistAnimeChannel:  make(chan *anilist.Anime),
		episodeReadChannel:          make(chan struct{}),
		errorChannel:                make(chan error),

		selectedProviders: make(map[*provider.Provider]struct{}),
		selectedEpisodes:  make(map[*source.Episode]struct{}),

		notifier: &ui.Model{},
	}

	// Options encapsulates the runtime configuration for the terminal user interface.
	type listOptions struct {
		TitleStyle mo.Option[lipgloss.Style]
	}

	makeList := func(title string, description bool, options *listOptions) list.Model {
		delegate := list.NewDefaultDelegate()
		delegate.SetSpacing(viper.GetInt(key.TUIItemSpacing))
		delegate.ShowDescription = description
		delegate.Styles.SelectedTitle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder(), false, false, false, true).
			BorderForeground(style.AccentColor).
			Foreground(style.AccentColor).
			Padding(0, 0, 0, 1)
		delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(lipgloss.Color("7"))
		delegate.Styles.SelectedDesc = delegate.Styles.SelectedTitle

		listC := list.New([]list.Item{}, delegate, 0, 0)
		listC.KeyMap = bubble.keymap.forList()
		listC.AdditionalShortHelpKeys = bubble.keymap.ShortHelp
		listC.AdditionalFullHelpKeys = func() []bubblesKey.Binding {
			return bubble.keymap.FullHelp()[0]
		}
		listC.Title = title
		listC.Styles.NoItems = paddingStyle
		if titleStyle, ok := options.TitleStyle.Get(); ok {
			listC.Styles.Title = titleStyle
		}
		listC.StatusMessageLifetime = time.Hour * 999
		// Enable wrap-around infinite scrolling for all lists
		listC.SetShowPagination(false)
		listC.SetShowStatusBar(false)

		return listC
	}

	bubble.helpC = help.New()

	bubble.spinnerC = spinner.New()
	bubble.spinnerC.Spinner = spinner.Dot
	bubble.spinnerC.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	bubble.inputC = textinput.New()
	bubble.inputC.Placeholder = fmt.Sprintf("Search Anime (v%s)", constant.Version)
	bubble.inputC.CharLimit = 60
	bubble.inputC.Prompt = viper.GetString(key.TUISearchPromptString)

	bubble.progressC = progress.New(progress.WithDefaultGradient())

	bubble.idInputC = textinput.New()
	bubble.idInputC.Placeholder = "Enter Manual ID"
	bubble.idInputC.CharLimit = 20
	bubble.idInputC.Prompt = "MAL/Anilist ID: "

	bubble.historyC = makeList("History", true, &listOptions{})

	bubble.sourcesC = makeList("Anime Sources", false, &listOptions{
		TitleStyle: mo.Some(
			lipgloss.NewStyle().Foreground(style.Base).Background(style.AccentColor).Padding(0, 1),
		),
	})
	bubble.sourcesC.SetStatusBarItemName("source", "sources")

	bubble.animesC = makeList("Anime Results", true, &listOptions{
		TitleStyle: mo.Some(
			lipgloss.NewStyle().Foreground(style.Base).Background(style.Lavender).Padding(0, 1),
		),
	})
	bubble.animesC.SetStatusBarItemName("anime", "anime")

	bubble.options = options

	bubble.historyC = makeList("History", false, &listOptions{
		TitleStyle: mo.Some(
			lipgloss.NewStyle().Foreground(style.Base).Background(style.Yellow).Padding(0, 1),
		),
	})
	bubble.historyC.SetStatusBarItemName("entry", "entries")

	bubble.episodesC = makeList("Episodes", true, &listOptions{
		TitleStyle: mo.Some(
			lipgloss.NewStyle().Foreground(style.Base).Background(style.Peach).Padding(0, 1),
		),
	})
	bubble.episodesC.SetStatusBarItemName("episode", "episodes")

	bubble.anilistC = makeList("Anime on Anilist", true, &listOptions{
		TitleStyle: mo.Some(
			lipgloss.NewStyle().Foreground(style.Base).Background(style.Blue).Padding(0, 1),
		),
	})
	bubble.anilistC.SetStatusBarItemName("anime", "animes")

	bubble.malListC = makeList("MyAnimeList", true, &listOptions{
		TitleStyle: mo.Some(
			lipgloss.NewStyle().Foreground(style.Base).Background(style.Blue).Padding(0, 1),
		),
	})
	bubble.malListC.SetStatusBarItemName("anime", "animes")

	bubble.postWatchC = makeList("Post-Watch Menu", false, &listOptions{
		TitleStyle: mo.Some(
			lipgloss.NewStyle().Foreground(style.Base).Background(style.Mauve).Padding(0, 1),
		),
	})
	bubble.postWatchC.SetItems([]list.Item{
		&listItem{internal: "Next"},
		&listItem{internal: "Replay"},
		&listItem{internal: "Previous"},
		&listItem{internal: "Back to Episodes"},
	})
	bubble.postWatchC.SetStatusBarItemName("option", "options")

	if w, h, err := util.TerminalSize(); err == nil {
		bubble.resize(w, h)
	}

	bubble.inputC.Focus()

	return &bubble
}
