// Package tui provides the primary terminal user interface implementation.
package tui

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/history"
	"github.com/anisan-cli/anisan/internal/ui/render"
	"github.com/anisan-cli/anisan/key"
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
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/spf13/viper"
)

// statefulBubble represents the main TUI model, coordinating components and state transitions.
type statefulBubble struct {
	state         state
	statesHistory util.Stack[state]
	loading       bool
	busy          bool // busy prevents rapid input processing during asynchronous operations

	keymap *statefulKeymap

	// components
	spinnerC   spinner.Model
	inputC     textinput.Model
	historyC   list.Model
	sourcesC   list.Model
	animesC    list.Model
	episodesC  list.Model
	trackerC   list.Model
	postWatchC list.Model
	progressC  progress.Model
	helpC      help.Model
	idInputC   textinput.Model // idInputC handles manual overrides for MyAnimeList or AniList IDs
	timerC     timer.Model
	httpClient *http.Client

	// cache
	coverArtCache sync.Map

	// delegates
	historyD   list.DefaultDelegate
	sourcesD   list.DefaultDelegate
	animesD    list.DefaultDelegate
	episodesD  list.DefaultDelegate
	trackerD   list.DefaultDelegate
	postWatchD list.DefaultDelegate
	lastSearchID int

	selectedProviders map[*provider.Provider]struct{}
	selectedSources   []source.Source
	selectedAnime     *source.Anime
	selectedEpisodes  map[*source.Episode]struct{} // Set of episodes selected for batch operations

	sourcesLoadedChannel        chan []source.Source
	foundAnimesChannel          chan []*source.Anime
	foundEpisodesChannel        chan []*source.Episode
	fetchedTrackerAnimesChannel chan any // Transports []mal.Anime or []*anilist.Anime
	closestTrackerAnimeChannel  chan any // Transports *mal.Anime or *anilist.Anime
	episodeReadChannel          chan struct{}
	errorChannel                chan error

	progressStatus string

	currentPlayingEpisode *source.Episode
	nextEpisodeToPlay     *source.Episode // Queued episode for seamless transitions
	mpvPlayer             player.Player
	trackerName           string // Canonical title on the tracking service (AniList/MAL)
	trackerURL            string // Direct URL to the anime record on the tracking service
	lastError             error

	width, height   int
	coverArtString  string            // ANSI-rendered raster data for the currently highlighted item
	imageMode       render.RenderMode // Evaluated terminal capability for image rendering
	imageColWidth   int               // Fixed horizontal constraint for the side-pane image container
	narrow          bool              // narrow indicates terminal width < 80, triggering graceful UI degradation

	options *Options
}

// raiseError dispatches a terminal error and transitions the application to the failure view.
func (b *statefulBubble) raiseError(err error) {
	b.lastError = err
	b.newState(errorState)
}

// showNotification initializes a deterministic timer that will clear the status message after the given duration.
func (b *statefulBubble) showNotification(msg string, duration time.Duration) tea.Cmd {
	b.progressStatus = msg
	b.timerC = timer.NewWithInterval(duration, time.Second)
	return b.timerC.Init()
}

// hideNotification instantly clears any active ephemeral notification.
func (b *statefulBubble) hideNotification() {
	b.progressStatus = ""
	b.timerC = timer.Model{} // Reset the timer safely
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
		trackerSelectState,
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

func (b *statefulBubble) resize(width, height int) tea.Cmd {
	if width == b.width && height == b.height {
		return nil
	}
	x, y := paddingStyle.GetFrameSize()
	xx, yy := listExtraPaddingStyle.GetFrameSize()

	styledWidth := width - x
	styledHeight := height - y

	b.width = styledWidth
	b.height = styledHeight
	b.narrow = width < 80

	// Compute image constraints dynamically.
	imgW, _ := b.getDynamicImageSize()
	b.imageColWidth = imgW + 4 // Account for lipgloss horizontal padding (2 left, 2 right)

	listWidth := width - xx - b.imageColWidth // Constrain lists so they don't overlap the fixed right pane
	listHeight := height - yy

	// Apply strict truncation constraints and styles to all list delegates dynamically.
	updateDelegate := func(l *list.Model, d *list.DefaultDelegate) {
		// Hide metadata descriptions on small windows to prevent layout claustrophobia
		d.ShowDescription = !b.narrow

		// Subtract horizontal padding and border space to prevent overflow
		maxWidth := listWidth - 4
		d.Styles.NormalTitle = d.Styles.NormalTitle.MaxWidth(maxWidth)
		d.Styles.SelectedTitle = d.Styles.SelectedTitle.MaxWidth(maxWidth)
		d.Styles.NormalDesc = d.Styles.NormalDesc.MaxWidth(maxWidth)
		d.Styles.SelectedDesc = d.Styles.SelectedDesc.MaxWidth(maxWidth)
		l.SetDelegate(*d)
	}

	updateDelegate(&b.historyC, &b.historyD)
	b.historyC.SetSize(listWidth, listHeight)
	b.historyC.Help.Width = listWidth

	updateDelegate(&b.sourcesC, &b.sourcesD)
	b.sourcesC.SetSize(listWidth, listHeight)
	b.sourcesC.Help.Width = listWidth

	updateDelegate(&b.animesC, &b.animesD)
	b.animesC.SetSize(listWidth, listHeight)
	b.animesC.Help.Width = listWidth

	updateDelegate(&b.episodesC, &b.episodesD)
	b.episodesC.SetSize(listWidth, listHeight)
	b.episodesC.Help.Width = listWidth

	updateDelegate(&b.trackerC, &b.trackerD)
	b.trackerC.SetSize(listWidth, listHeight)
	b.trackerC.Help.Width = listWidth

	updateDelegate(&b.postWatchC, &b.postWatchD)
	b.postWatchC.SetSize(listWidth, listHeight)
	b.postWatchC.Help.Width = listWidth

	b.progressC.Width = listWidth
	b.idInputC.Width = listWidth

	b.width = styledWidth
	b.height = styledHeight
	b.helpC.Width = listWidth

	// Trigger cover art reflow for the currently selected item if it exists.
	if b.state == animesState {
		if item := b.animesC.SelectedItem(); item != nil {
			if a, ok := item.(*listItem).internal.(*source.Anime); ok {
				return b.fetchCoverArt(a)
			}
		}
	} else if b.state == historyState {
		if item := b.historyC.SelectedItem(); item != nil {
			if h, ok := item.(*listItem).internal.(*history.SavedEpisode); ok {
				// We need a source.Anime for fetchCoverArt. 
				// The history might not have it immediately, so we just use the selectedAnime if it matches name.
				if b.selectedAnime != nil && b.selectedAnime.Name == h.AnimeName {
					return b.fetchCoverArt(b.selectedAnime)
				}
			}
		}
	}
	return nil
}

// startLoading enters a concurrent loading state, initializing visual indicators across child components.
func (b *statefulBubble) startLoading() tea.Cmd {
	b.loading = true
	b.busy = true
	return tea.Batch(b.animesC.StartSpinner(), b.episodesC.StartSpinner(), b.trackerC.StartSpinner())
}

// stopLoading exits the loading state and synchronizes child component visual indicators.
func (b *statefulBubble) stopLoading() tea.Cmd {
	b.loading = false
	b.busy = false
	b.animesC.StopSpinner()
	b.episodesC.StopSpinner()
	b.trackerC.StopSpinner()
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
		fetchedTrackerAnimesChannel: make(chan any),
		closestTrackerAnimeChannel:  make(chan any),
		episodeReadChannel:          make(chan struct{}),
		errorChannel:                make(chan error),

		selectedProviders: make(map[*provider.Provider]struct{}),
		selectedEpisodes:  make(map[*source.Episode]struct{}),
		imageMode:         render.DetectProtocol(),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Options encapsulates the runtime configuration for the terminal user interface.
	type listOptions struct {
		TitleStyle mo.Option[lipgloss.Style]
		PinkStyle  bool
	}

	makeList := func(title string, description bool, options *listOptions) (list.Model, list.DefaultDelegate) {
		delegate := list.NewDefaultDelegate()
		delegate.SetSpacing(viper.GetInt(key.TUIItemSpacing))
		delegate.ShowDescription = description

		delegate.Styles.SelectedTitle = style.SelectedTitleStyle
		delegate.Styles.SelectedDesc = style.SelectedDescStyle

		delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(style.Text)

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

		return listC, delegate
	}

	bubble.helpC = help.New()

	bubble.spinnerC = spinner.New()
	bubble.spinnerC.Spinner = spinner.Dot
	bubble.spinnerC.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	bubble.inputC = textinput.New()
	bubble.inputC.Placeholder = fmt.Sprintf("Search Anime (v%s)", constant.Version)
	bubble.inputC.CharLimit = 60
	bubble.inputC.Prompt = viper.GetString(key.TUISearchPromptString)
	bubble.inputC.ShowSuggestions = true

	bubble.progressC = progress.New(
		progress.WithoutPercentage(),
		progress.WithGradient(string(style.AccentColor.Dark), string(style.SecondaryColor.Dark)),
	)

	bubble.idInputC = textinput.New()
	bubble.idInputC.Placeholder = "Enter Manual ID"
	bubble.idInputC.CharLimit = 20
	bubble.idInputC.Prompt = "MAL/Anilist ID: "

	bubble.sourcesC, bubble.sourcesD = makeList("Anime Sources", false, &listOptions{
		TitleStyle: mo.Some(
			lipgloss.NewStyle().Foreground(style.Base).Background(style.AccentColor).Padding(0, 1),
		),
	})
	bubble.sourcesC.SetStatusBarItemName("source", "sources")

	bubble.animesC, bubble.animesD = makeList("Anime Results", true, &listOptions{
		TitleStyle: mo.Some(
			lipgloss.NewStyle().Foreground(style.Base).Background(style.Lavender).Padding(0, 1),
		),
	})
	bubble.animesC.SetStatusBarItemName("anime", "anime")

	bubble.options = options

	bubble.historyC, bubble.historyD = makeList("History", true, &listOptions{
		TitleStyle: mo.Some(
			lipgloss.NewStyle().Foreground(style.Base).Background(style.Yellow).Padding(0, 1),
		),
	})
	bubble.historyC.SetStatusBarItemName("entry", "entries")
	bubble.historyC.SetFilteringEnabled(true)

	bubble.episodesC, bubble.episodesD = makeList("Episodes", true, &listOptions{
		TitleStyle: mo.Some(
			lipgloss.NewStyle().Foreground(style.Base).Background(style.Peach).Padding(0, 1),
		),
	})
	bubble.episodesC.SetStatusBarItemName("episode", "episodes")

	bubble.trackerC, bubble.trackerD = makeList("Tracker Results", true, &listOptions{
		TitleStyle: mo.Some(
			lipgloss.NewStyle().Foreground(style.Base).Background(style.Blue).Padding(0, 1),
		),
	})
	bubble.trackerC.SetStatusBarItemName("anime", "animes")

	bubble.postWatchC, bubble.postWatchD = makeList("Post-Watch Menu", false, &listOptions{
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
