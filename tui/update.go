// Package tui provides the primary terminal user interface implementation.
package tui

import (
	"context"
	"fmt"
	"io"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/anisan-cli/anisan/anilist"
	"github.com/anisan-cli/anisan/color"
	"github.com/anisan-cli/anisan/history"
	"github.com/anisan-cli/anisan/internal/tracker"
	"github.com/anisan-cli/anisan/internal/ui/render"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/mal"
	"github.com/anisan-cli/anisan/open"
	"github.com/anisan-cli/anisan/player"
	"github.com/anisan-cli/anisan/provider"
	"github.com/anisan-cli/anisan/query"
	"github.com/anisan-cli/anisan/source"
	"github.com/anisan-cli/anisan/style"
	bubblesKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/samber/lo"
	"github.com/spf13/viper"
)

type searchDebounceMsg struct {
	id    int
	query string
}

type playSyncMsg struct {
	url     string
	title   string
	headers map[string]string
}

// playbackCmd implements tea.ExecCommand for seamless terminal handoff.
type playbackCmd struct {
	player  player.Player
	url     string
	title   string
	headers map[string]string
}

func (p playbackCmd) Run() error {
	return p.player.PlaySync(p.url, p.title, p.headers)
}

func (p playbackCmd) SetStdin(r io.Reader)  {}
func (p playbackCmd) SetStdout(w io.Writer) {}
func (p playbackCmd) SetStderr(w io.Writer) {}

func (b *statefulBubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case provider.ScraperUpdatedMsg:
		return b, b.loadProviders()
	case coverArtMsg:
		b.coverArtString = string(msg)
		return b, nil
	case metadataPopulatedMsg:
		// Refresh list and cover art after metadata fetch.
		if b.state == animesState {
			cmd = b.animesC.SetItems(b.animesC.Items())
			if item := b.animesC.SelectedItem(); item != nil {
				if selected, ok := item.(*listItem).internal.(*source.Anime); ok && selected == msg.anime {
					cmd = tea.Batch(cmd, b.fetchCoverArt(msg.anime))
				}
			}
		}
		return b, cmd
	case spinner.TickMsg:
		var cmd tea.Cmd
		b.spinnerC, cmd = b.spinnerC.Update(msg)
		return b, cmd
	case playSyncMsg:
		b.progressStatus = "" // clear "Launching..." before TTY handoff
		return b, tea.Exec(playbackCmd{
			player:  b.mpvPlayer,
			url:     msg.url,
			title:   msg.title,
			headers: msg.headers,
		}, func(err error) tea.Msg {
			if err != nil {
				return err
			}
			return nil
		})
	case searchDebounceMsg:
		if msg.id == b.lastSearchID && msg.query != "" {
			go query.Remember(msg.query, 1)
			b.progressStatus = fmt.Sprintf("Searching for %s...", msg.query)
			b.newState(loadingState)
			return b, tea.Batch(b.startLoading(), b.searchAnime(msg.query), b.waitForAnimes(), b.spinnerC.Tick)
		}
		return b, nil
	case *mal.Anime:
		return b, b.applyManualTrackerUpdate(msg)
	case *anilist.Anime:
		return b, b.applyManualTrackerUpdate(msg)
	case error:
		if msg.Error() == "sync_queued" {
			return b, b.showNotification("Tracking offline. Queued for background sync.", 3*time.Second)
		}
		b.raiseError(msg)
	case timer.TickMsg:
		var cmd tea.Cmd
		b.timerC, cmd = b.timerC.Update(msg)
		return b, cmd
	case timer.TimeoutMsg:
		b.hideNotification()
		return b, nil
	case tea.WindowSizeMsg:
		return b, b.resize(msg.Width, msg.Height)
	case tea.MouseMsg:
		if msg.Type == tea.MouseWheelUp || msg.Type == tea.MouseWheelDown {
			var l *list.Model
			switch b.state {
			case historyState:
				l = &b.historyC
			case sourcesState:
				l = &b.sourcesC
			case animesState:
				l = &b.animesC
			case episodesState:
				l = &b.episodesC
			case trackerSelectState:
				l = &b.trackerC
			case postWatchState:
				l = &b.postWatchC
			}
			if l != nil {
				if msg.Type == tea.MouseWheelUp {
					l.CursorUp()
				} else {
					l.CursorDown()
				}
				// Update side pane after scroll.
				return b, nil
			}
		}
	case tea.KeyMsg:
		switch {
		case bubblesKey.Matches(msg, b.keymap.forceQuit):
			return b, tea.Quit
		case bubblesKey.Matches(msg, b.keymap.showHelp):
			if b.state != searchState && b.state != manualIDState {
				b.helpC.ShowAll = !b.helpC.ShowAll
				return b, nil
			}
		}

		if b.busy && b.state != readState && b.state != errorState {
			return b, nil
		}

		switch {
		case bubblesKey.Matches(msg, b.keymap.back):
			onListBack := func(l *list.Model) tea.Cmd {
				l.ResetSelected()
				l.ResetFilter()
				return tea.Batch(cmd, l.NewStatusMessage(""))
			}

			switch b.state {
			case searchState:
				b.inputC.SetValue("")
			case episodesState:
				if b.episodesC.FilterState() != list.Unfiltered {
					b.episodesC, cmd = b.episodesC.Update(msg)
					return b, cmd
				}
				b.selectedEpisodes = make(map[*source.Episode]struct{})
				cmd = onListBack(&b.episodesC)
			case trackerSelectState:
				if b.trackerC.FilterState() != list.Unfiltered {
					b.trackerC, cmd = b.trackerC.Update(msg)
					return b, cmd
				}
				cmd = onListBack(&b.trackerC)
			case animesState:
				if b.animesC.FilterState() != list.Unfiltered {
					b.animesC, cmd = b.animesC.Update(msg)
					return b, cmd
				}
				cmd = onListBack(&b.animesC)
			case historyState:
				if b.historyC.FilterState() != list.Unfiltered {
					b.historyC, cmd = b.historyC.Update(msg)
					return b, cmd
				}
				cmd = onListBack(&b.historyC)
			case sourcesState:
				if b.sourcesC.FilterState() != list.Unfiltered {
					b.sourcesC, cmd = b.sourcesC.Update(msg)
					return b, cmd
				}
				cmd = onListBack(&b.sourcesC)
			}

			b.previousState()
			b.stopLoading()
			return b, cmd
		}
	}

	switch b.state {
	case loadingState:
		return b.updateLoading(msg)
	case historyState:
		return b.updateHistory(msg)
	case sourcesState:
		return b.updateSources(msg)
	case searchState:
		return b.updateSearch(msg)
	case animesState:
		return b.updateAnimes(msg)
	case episodesState:
		return b.updateEpisodes(msg)
	case trackerSelectState:
		return b.updateTrackerSelect(msg)
	case readState:
		return b.updateRead(msg)
	case postWatchState:
		return b.updatePostWatch(msg)
	case manualIDState:
		return b.updateManualID(msg)
	case errorState:
		return b.updateError(msg)
	}

	return b, nil
}

func (b *statefulBubble) updateLoading(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds = make([]tea.Cmd, 0)
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case bubblesKey.Matches(msg, b.keymap.back):
			if b.statesHistory.Len() > 0 {
				b.previousState()
			} else {
				return b, tea.Quit
			}
		}
	case anilistTrackerFetchMsg:
		items := make([]list.Item, len(msg.animes))
		var marked int
		for i, al := range msg.animes {
			if al.ID == msg.closestID {
				marked = i
			}
			items[i] = &listItem{
				internal: al,
				marked:   al.ID == msg.closestID,
			}
		}

		cmd = b.trackerC.SetItems(items)
		b.newState(trackerSelectState)
		b.trackerC.Select(marked)
		return b, tea.Batch(cmd, b.stopLoading())
	case malTrackerFetchMsg:
		items := make([]list.Item, len(msg.animes))
		var marked int
		for i := range msg.animes {
			if msg.animes[i].ID == msg.closestID {
				marked = i
			}
			items[i] = &listItem{
				internal: &msg.animes[i],
				marked:   msg.animes[i].ID == msg.closestID,
			}
		}

		cmd = b.trackerC.SetItems(items)
		b.newState(trackerSelectState)
		b.trackerC.Select(marked)
		return b, tea.Batch(cmd, b.stopLoading())
	case []*source.Anime:
		items := make([]list.Item, len(msg))
		for i, m := range msg {
			items[i] = &listItem{internal: m}
		}

		cmds = append(cmds, b.animesC.SetItems(items))
		b.newState(animesState)
		b.stopLoading()

		cmds = append(cmds, b.batchPopulateMetadata(msg))

		if len(msg) > 0 {
			cmds = append(cmds, b.fetchCoverArt(msg[0]))
		}
	case []*source.Episode:
		if b.statesHistory.Peek() == historyState {
			b.newState(historyState)
			b.stopLoading()
			cmds = append(cmds, func() tea.Msg {
				return msg
			})
		}
	case []source.Source:
		b.selectedSources = msg

		if b.statesHistory.Peek() == historyState {
			b.newState(historyState)
			b.stopLoading()
			cmds = append(cmds, func() tea.Msg {
				return msg
			})
		} else {
			b.stopLoading()
			b.newState(searchState)
		}
	}

	b.spinnerC, cmd = b.spinnerC.Update(msg)
	return b, tea.Batch(append(cmds, cmd)...)
}

func (b *statefulBubble) updateHistory(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case []source.Source: // Sources loaded for history entry
		b.selectedSources = msg
		selected := b.historyC.SelectedItem().(*listItem).internal.(*history.SavedEpisode)

		anime := &source.Anime{
			Name:   selected.AnimeName,
			URL:    selected.AnimeURL,
			Index:  0,
			ID:     selected.AnimeID,
			Source: b.selectedSources[0],
		}

		b.selectedAnime = anime

		b.progressStatus = fmt.Sprintf("Loading episodes for %s...", anime.Name)
		b.newState(loadingState)
		return b, tea.Batch(b.getEpisodes(anime), b.waitForEpisodes(), b.startLoading())

	case []*source.Episode:
		sort.Slice(msg, func(i, j int) bool {
			if msg[i].Index != msg[j].Index {
				return msg[i].Index < msg[j].Index
			}
			return msg[i].Name < msg[j].Name
		})

		items := make([]list.Item, len(msg))
		for i, c := range msg {
			items[i] = &listItem{internal: c}
		}

		cmd = b.episodesC.SetItems(items)

		// Find the episode the user was watching from history
		selected := b.historyC.SelectedItem().(*listItem).internal.(*history.SavedEpisode)
		var epToPlay *source.Episode
		var epIdx int
		for i, ep := range msg {
			if ep.URL == selected.URL {
				epToPlay = msg[i]
				epIdx = i
				break
			}
		}
		// Fallback: play the first episode if URL changed
		if epToPlay == nil && len(msg) > 0 {
			epToPlay = msg[0]
			epIdx = 0
		}

		// Auto-advance to the next episode if the current one is completed
		if epToPlay != nil && selected.WatchedPercentage >= float64(viper.GetInt(key.PlayerCompletionPercentage)) {
			if epIdx+1 < len(msg) {
				epToPlay = msg[epIdx+1]
				epIdx++
			}
		}

		if epToPlay != nil {
			b.episodesC.Select(epIdx)

			// Auto-play from history and push state for navigation.
			b.currentPlayingEpisode = epToPlay
			b.newState(episodesState)
			b.newState(readState)
			b.stopLoading()

			var trackerCmd tea.Cmd
			if viper.GetBool("tracker.auto_link") {
				trackerCmd = tea.Batch(b.fetchAndSetTracker(b.selectedAnime), b.waitForTrackerFetchAndSet())
			} else {
				// Load tracker cache based on backend.
				if viper.GetString("tracker.backend") == "mal" {
					trackerCmd = b.tryLoadMALCache(b.selectedAnime)
				} else {
					trackerCmd = b.tryLoadAnilistCache(b.selectedAnime)
				}
			}
			return b, tea.Batch(cmd, b.readEpisode(epToPlay), b.startLoading(), trackerCmd)
		} else {
			// If no episodes were found at all, just fall back to the empty episodes list view
			b.newState(episodesState)
			b.stopLoading()
			return b, cmd
		}
	case tea.KeyMsg:
		// In Filtering state, all keypresses go to the text input box.
		// Let Bubbletea's list handle them exclusively — typing refines the filter,
		// Enter commits it (transitions to FilterApplied), Esc cancels.
		if b.historyC.FilterState() == list.Filtering {
			break
		}
		switch {

		case bubblesKey.Matches(msg, b.keymap.up):
			visible := b.historyC.VisibleItems()
			if len(visible) > 0 && b.historyC.Index() == 0 {
				b.historyC.Select(len(visible) - 1)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.down):
			visible := b.historyC.VisibleItems()
			if len(visible) > 0 && b.historyC.Index() == len(visible)-1 {
				b.historyC.Select(0)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.openURL):
			if b.historyC.SelectedItem() != nil {
				entry := b.historyC.SelectedItem().(*listItem).internal.(*history.SavedEpisode)
				err := open.Start(entry.URL)
				if err != nil {
					b.raiseError(err)
				}
			}
		case bubblesKey.Matches(msg, b.keymap.remove):
			if b.historyC.SelectedItem() != nil {
				entry := b.historyC.SelectedItem().(*listItem).internal.(*history.SavedEpisode)
				_ = history.Remove(entry)
				cmd, err := b.loadHistory()
				if err != nil {
					return nil, nil // Error during load
				}
				return b, cmd
			}
		case bubblesKey.Matches(msg, b.keymap.selectOne, b.keymap.confirm):
			if b.historyC.SelectedItem() != nil {
				selected := b.historyC.SelectedItem().(*listItem).internal.(*history.SavedEpisode)
				providers := lo.Map(b.sourcesC.Items(), func(i list.Item, _ int) *provider.Provider {
					return i.(*listItem).internal.(*provider.Provider)
				})

				p, ok := lo.Find(providers, func(p *provider.Provider) bool {
					return p.ID == selected.SourceID
				})

				if !ok {
					err := fmt.Errorf("provider %s not found (was used for %s)", selected.SourceID, selected.AnimeName)
					b.raiseError(err)
					return b, nil
				}

				b.newState(loadingState)
				b.progressStatus = "Initializing source..."
				return b, tea.Batch(b.startLoading(), b.loadSources([]*provider.Provider{p}), b.waitForSourcesLoaded())
			}
		case bubblesKey.Matches(msg, b.keymap.back):
			if b.historyC.FilterState() != list.Unfiltered {
				break
			}
			b.previousState()
			return b, nil
		}
	}

	prevFilterState := b.historyC.FilterState()
	oldIdx := b.historyC.Index()
	b.historyC, cmd = b.historyC.Update(msg)

	// Single-Enter selection for filtered list.
	if prevFilterState == list.Filtering && b.historyC.FilterState() == list.FilterApplied {
		if b.historyC.SelectedItem() != nil {
			selected := b.historyC.SelectedItem().(*listItem).internal.(*history.SavedEpisode)
			providers := lo.Map(b.sourcesC.Items(), func(i list.Item, _ int) *provider.Provider {
				return i.(*listItem).internal.(*provider.Provider)
			})
			p, ok := lo.Find(providers, func(p *provider.Provider) bool {
				return p.ID == selected.SourceID
			})
			if ok {
				b.newState(loadingState)
				b.progressStatus = "Initializing source..."
				return b, tea.Batch(b.startLoading(), b.loadSources([]*provider.Provider{p}), b.waitForSourcesLoaded())
			}
		}
	}

	if b.historyC.Index() != oldIdx {
		if b.historyC.SelectedItem() != nil {
			m, _ := b.historyC.SelectedItem().(*listItem).internal.(*history.SavedEpisode)
			cmd = tea.Batch(cmd, b.fetchCoverArtFromURL(m.CoverURL, ""))
		}
	}

	return b, cmd
}

func (b *statefulBubble) updateSources(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Prevent navigation conflicts during list filtering.
		if b.sourcesC.FilterState() == list.Filtering {
			break
		}
		switch {

		case bubblesKey.Matches(msg, b.keymap.up):
			if n := len(b.sourcesC.VisibleItems()); n > 0 && b.sourcesC.Index() == 0 {
				b.sourcesC.Select(n - 1)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.down):
			p := b.sourcesC.VisibleItems()
			if n := len(p); n > 0 && b.sourcesC.Index() == n-1 {
				b.sourcesC.Select(0)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.back):
			if b.sourcesC.FilterState() != list.Unfiltered {
				break
			}
			b.previousState()
			return b, nil
		case bubblesKey.Matches(msg, b.keymap.selectAll):
			for _, item := range b.sourcesC.Items() {
				item := item.(*listItem)
				item.marked = true
				b.selectedProviders[item.internal.(*provider.Provider)] = struct{}{}
			}
		case bubblesKey.Matches(msg, b.keymap.clearSelection):
			for _, item := range b.sourcesC.Items() {
				item := item.(*listItem)
				item.marked = false
				delete(b.selectedProviders, item.internal.(*provider.Provider))
			}
		case bubblesKey.Matches(msg, b.keymap.selectOne):
			if b.sourcesC.SelectedItem() == nil {
				break
			}
			item := b.sourcesC.SelectedItem().(*listItem)
			p := item.internal.(*provider.Provider)

			if item.marked {
				delete(b.selectedProviders, p)
			} else {
				b.selectedProviders[p] = struct{}{}
			}
			item.toggleMark()
		case bubblesKey.Matches(msg, b.keymap.saveAsDefault):
			if b.sourcesC.SelectedItem() == nil {
				break
			}
			item := b.sourcesC.SelectedItem().(*listItem)
			p := item.internal.(*provider.Provider)

			viper.Set(key.DefaultSources, []string{p.Name})
			if err := viper.WriteConfig(); err != nil {
				b.raiseError(err)
				break
			}

			// Update the results header to indicate the currently active provider.
			b.animesC.Title = fmt.Sprintf("Anime - %s", p.Name)
			b.sourcesC.NewStatusMessage(fmt.Sprintf("Saved %s as default source", p.Name))

			b.newState(loadingState)
			return b, tea.Batch(b.startLoading(), b.loadSources([]*provider.Provider{p}), b.waitForSourcesLoaded())

		case bubblesKey.Matches(msg, b.keymap.confirm):
			if b.sourcesC.SelectedItem() == nil {
				break
			}
			item := b.sourcesC.SelectedItem().(*listItem)

			if len(b.selectedProviders) == 0 {
				p := item.internal.(*provider.Provider)
				b.animesC.Title = fmt.Sprintf("Anime - %s", p.Name)
				b.progressStatus = fmt.Sprintf("Loading anime from %s...", p.Name)
				b.newState(loadingState)
				return b, tea.Batch(b.startLoading(), b.loadSources([]*provider.Provider{p}), b.waitForSourcesLoaded())
			}

			b.progressStatus = "Loading selected providers..."
			b.newState(loadingState)
			return b, tea.Batch(b.startLoading(), b.loadSources(lo.Keys(b.selectedProviders)), b.waitForSourcesLoaded())
		}
	}

	b.sourcesC, cmd = b.sourcesC.Update(msg)
	return b, cmd
}

func (b *statefulBubble) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case bubblesKey.Matches(msg, b.keymap.changeSource):
			b.newState(sourcesState)
			return b, b.loadProviders()
		case bubblesKey.Matches(msg, b.keymap.confirm) && b.inputC.Value() != "":
			// Instant search override.
			b.lastSearchID++
			b.progressStatus = fmt.Sprintf("Searching for %s...", b.inputC.Value())
			b.startLoading()
			b.newState(loadingState)
			go query.Remember(b.inputC.Value(), 1)
			return b, tea.Batch(b.searchAnime(b.inputC.Value()), b.waitForAnimes(), b.spinnerC.Tick)
		case bubblesKey.Matches(msg, b.keymap.acceptSearchSuggestion):
			if s := b.inputC.AvailableSuggestions(); len(s) > 0 {
				b.inputC.SetValue(s[0])
				b.inputC.SetCursor(len(b.inputC.Value()))
				// Trigger periodic debounce for accepted suggestions too
				b.lastSearchID++
				id := b.lastSearchID
				val := b.inputC.Value()
				return b, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
					return searchDebounceMsg{id: id, query: val}
				})
			}
			return b, nil
		case bubblesKey.Matches(msg, b.keymap.back):
			b.previousState()
			return b, nil
		}
	}

	oldVal := b.inputC.Value()
	b.inputC, cmd = b.inputC.Update(msg)
	newVal := b.inputC.Value()

	if _, ok := msg.(tea.KeyMsg); ok {
		if newVal != oldVal && newVal != "" {
			b.lastSearchID++
			id := b.lastSearchID
			return b, tea.Batch(cmd, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
				return searchDebounceMsg{id: id, query: newVal}
			}))
		}
	}

	return b, cmd
}

func (b *statefulBubble) updateAnimes(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Prevent navigation conflicts during list filtering.
		if b.animesC.FilterState() == list.Filtering {
			break
		}
		switch {
		case bubblesKey.Matches(msg, b.keymap.changeSource):
			b.newState(sourcesState)
			return b, b.loadProviders()

		case bubblesKey.Matches(msg, b.keymap.up):
			if n := len(b.animesC.VisibleItems()); n > 0 && b.animesC.Index() == 0 {
				b.animesC.Select(n - 1)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.down):
			p := b.animesC.VisibleItems()
			if n := len(p); n > 0 && b.animesC.Index() == n-1 {
				b.animesC.Select(0)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.back):
			if b.animesC.FilterState() != list.Unfiltered {
				break
			}
			b.previousState()
			return b, nil
		case bubblesKey.Matches(msg, b.keymap.confirm, b.keymap.selectOne):
			if b.animesC.SelectedItem() == nil {
				break
			}
			m, _ := b.animesC.SelectedItem().(*listItem).internal.(*source.Anime)
			b.selectedAnime = m
			b.coverArtString = "" // clear stale image
			b.progressStatus = fmt.Sprintf("Loading episodes for %s...", m.Name)
			go query.Remember(m.Name, 2)
			return b, tea.Batch(b.getEpisodes(m), b.waitForEpisodes(), b.startLoading(), b.fetchCoverArt(m))

		}
	case []*source.Episode:
		sort.Slice(msg, func(i, j int) bool {
			if msg[i].Index != msg[j].Index {
				return msg[i].Index < msg[j].Index
			}
			return msg[i].Name < msg[j].Name
		})

		items := make([]list.Item, len(msg))
		for i, c := range msg {
			items[i] = &listItem{internal: c}
		}

		cmd = b.episodesC.SetItems(items)
		b.coverArtString = "" // clear previous image
		b.newState(episodesState)
		b.stopLoading()

		var finalCmd tea.Cmd

		// If --continue is specified and we just loaded the episodes list
		if b.options != nil && b.options.Continue {
			b.options.Continue = false // consume the flag so it only auto-plays once

			var epToPlay *source.Episode
			// Try to find the last watched episode in history
			historyData, err := history.Get()
			if err == nil {
				// We need the Anime name to look up its history
				var animeName string
				if len(msg) > 0 {
					animeName = msg[0].Anime.Name
				}

				// Find this anime in history
				var lastWatched *history.SavedEpisode
				for _, entry := range historyData {
					if entry.AnimeName == animeName {
						lastWatched = entry
						break
					}
				}

				if lastWatched != nil {
					// Find the NEXT episode in our ascending list
					for i, e := range msg {
						if e.URL == lastWatched.URL {
							if i+1 < len(msg) {
								epToPlay = msg[i+1]
							} else {
								// They finished the last episode, just play the last one again or do nothing
								epToPlay = msg[i]
							}
							break
						}
					}
				}
			}

			// If no history found, play the first episode
			if epToPlay == nil && len(msg) > 0 {
				epToPlay = msg[0]
			}

			if epToPlay != nil {
				b.progressStatus = fmt.Sprintf("Launching %s - %s", b.selectedAnime.Name, epToPlay.Name)
				finalCmd = tea.Batch(cmd, b.readEpisode(epToPlay), b.fetchCoverArt(b.selectedAnime))
			} else {
				finalCmd = tea.Batch(cmd, b.fetchCoverArt(b.selectedAnime))
			}
		} else {
			finalCmd = tea.Batch(cmd, b.fetchCoverArt(b.selectedAnime))
		}

		if viper.GetBool("tracker.auto_link") {
			return b, tea.Batch(finalCmd, b.fetchAndSetTracker(b.selectedAnime), b.waitForTrackerFetchAndSet())
		}

		return b, finalCmd
	}

	oldIdx := b.animesC.Index()
	b.animesC, cmd = b.animesC.Update(msg)

	if b.animesC.Index() != oldIdx {
		if b.animesC.SelectedItem() != nil {
			m, _ := b.animesC.SelectedItem().(*listItem).internal.(*source.Anime)
			cmd = tea.Batch(cmd, b.fetchCoverArt(m))
		}
	}

	return b, cmd
}

func (b *statefulBubble) updateEpisodes(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case *anilist.Anime:
		b.trackerName = msg.Name()
		b.trackerURL = msg.SiteURL
		cmd = b.episodesC.NewStatusMessage(fmt.Sprintf(`Linked to %s %s`, style.Fg(color.Orange)(b.trackerName), style.Faint(b.trackerURL)))
		return b, cmd
	case *mal.Anime:
		b.trackerName = msg.Title
		b.trackerURL = fmt.Sprintf("https://myanimelist.net/anime/%d", msg.ID)
		cmd = b.episodesC.NewStatusMessage(fmt.Sprintf(`Linked to %s %s`, style.Fg(color.Orange)(b.trackerName), style.Faint(b.trackerURL)))
		return b, cmd
	case tea.KeyMsg:
		// Prevent navigation conflicts during list filtering.
		if b.episodesC.FilterState() == list.Filtering {
			break
		}
		switch {

		case bubblesKey.Matches(msg, b.keymap.up):
			if n := len(b.episodesC.VisibleItems()); n > 0 && b.episodesC.Index() == 0 {
				b.episodesC.Select(n - 1)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.down):
			p := b.episodesC.VisibleItems()
			if n := len(p); n > 0 && b.episodesC.Index() == n-1 {
				b.episodesC.Select(0)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.back):
			if b.episodesC.FilterState() != list.Unfiltered {
				break
			}
			b.previousState()
			return b, nil
		case bubblesKey.Matches(msg, b.keymap.manualID):
			if b.episodesC.SelectedItem() == nil {
				break
			}
			b.newState(manualIDState)
			b.idInputC.Focus()
			b.idInputC.SetValue("")
			return b, textinput.Blink
		case bubblesKey.Matches(msg, b.keymap.openURL):
			if b.episodesC.SelectedItem() == nil {
				break
			}
			episode := b.episodesC.SelectedItem().(*listItem).internal.(*source.Episode)
			err := open.Start(episode.URL)
			if err != nil {
				b.raiseError(err)
			}
		case bubblesKey.Matches(msg, b.keymap.trackerSelect):
			// If already linked, provide the "Open in Browser" QoL feature
			if b.trackerURL != "" {
				if err := open.Start(b.trackerURL); err != nil {
					b.raiseError(err)
				}
				return b, nil
			}

			// Start manual linking.
			backend := viper.GetString("tracker.backend")
			if backend == "mal" {
				b.progressStatus = fmt.Sprintf("Searching MAL for %s", b.selectedAnime.Name)
				b.newState(loadingState)
				return b, tea.Batch(b.startLoading(), b.fetchMALAnime(b.selectedAnime.Name), b.waitForMALAnime())
			}

			// Default to AniList
			// Transition to loadingState to ensure the asynchronous []anilist.Anime payload is caught.
			b.progressStatus = fmt.Sprintf("Fetching AniList for %s", b.selectedAnime.Name)
			b.newState(loadingState)
			return b, tea.Batch(b.startLoading(), b.fetchAnilist(b.selectedAnime), b.waitForAnilist())
		case bubblesKey.Matches(msg, b.keymap.selectVolume):
			if b.episodesC.SelectedItem() == nil {
				break
			}
			episode := b.episodesC.SelectedItem().(*listItem).internal.(*source.Episode)
			if episode.Volume == "" {
				break
			}
			for _, item := range b.episodesC.Items() {
				item := item.(*listItem)
				if item.internal.(*source.Episode).Volume == episode.Volume {
					if !item.marked {
						b.selectedEpisodes[item.internal.(*source.Episode)] = struct{}{}
					}
					item.marked = true
				}
			}
		case bubblesKey.Matches(msg, b.keymap.selectOne):
			if b.episodesC.SelectedItem() == nil {
				break
			}
			item := b.episodesC.SelectedItem().(*listItem)
			episode := item.internal.(*source.Episode)

			item.toggleMark()
			if item.marked {
				b.selectedEpisodes[episode] = struct{}{}
			} else {
				delete(b.selectedEpisodes, episode)
			}
		case bubblesKey.Matches(msg, b.keymap.selectAll):
			items := b.episodesC.Items()
			if len(items) == 0 {
				break
			}
			for _, item := range items {
				item := item.(*listItem)
				item.marked = true
				episode := item.internal.(*source.Episode)
				b.selectedEpisodes[episode] = struct{}{}
			}
		case bubblesKey.Matches(msg, b.keymap.clearSelection):
			items := b.episodesC.Items()
			if len(items) == 0 {
				break
			}
			for _, item := range items {
				item := item.(*listItem)
				item.marked = false
				episode := item.internal.(*source.Episode)
				delete(b.selectedEpisodes, episode)
			}
		case bubblesKey.Matches(msg, b.keymap.read):
			if b.episodesC.SelectedItem() == nil {
				break
			}
			episode := b.episodesC.SelectedItem().(*listItem).internal.(*source.Episode)
			b.progressStatus = fmt.Sprintf("Launching %s - %s", b.selectedAnime.Name, episode.Name)
			b.currentPlayingEpisode = episode
			b.newState(readState)
			return b, tea.Batch(b.readEpisode(episode), b.waitForEpisodeRead(), b.startLoading())
		case bubblesKey.Matches(msg, b.keymap.confirm):
			if len(b.selectedEpisodes) != 0 {
				b.newState(confirmState)
			} else if viper.GetBool(key.TUIReadOnEnter) {
				if b.episodesC.SelectedItem() == nil {
					break
				}
				episode := b.episodesC.SelectedItem().(*listItem).internal.(*source.Episode)
				b.progressStatus = fmt.Sprintf("Launching %s - %s", b.selectedAnime.Name, episode.Name)
				b.currentPlayingEpisode = episode
				b.newState(readState)
				return b, tea.Batch(b.readEpisode(episode), b.waitForEpisodeRead(), b.startLoading())
			}
		}
	}

	b.episodesC, cmd = b.episodesC.Update(msg)
	return b, cmd
}

func (b *statefulBubble) updateTrackerSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Prevent navigation conflicts during list filtering.
		if b.trackerC.FilterState() == list.Filtering {
			break
		}
		switch {
		case bubblesKey.Matches(msg, b.keymap.up):
			if n := len(b.trackerC.VisibleItems()); n > 0 && b.trackerC.Index() == 0 {
				b.trackerC.Select(n - 1)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.down):
			p := b.trackerC.VisibleItems()
			if n := len(p); n > 0 && b.trackerC.Index() == n-1 {
				b.trackerC.Select(0)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.openURL):
			if b.trackerC.SelectedItem() == nil {
				break
			}
			switch m := b.trackerC.SelectedItem().(*listItem).internal.(type) {
			case *anilist.Anime:
				err := open.Start(m.SiteURL)
				if err != nil {
					b.raiseError(err)
				}
			case *mal.Anime:
				url := fmt.Sprintf("https://myanimelist.net/anime/%d", m.ID)
				err := open.Start(url)
				if err != nil {
					b.raiseError(err)
				}
			}
		case bubblesKey.Matches(msg, b.keymap.confirm, b.keymap.selectOne):
			if b.trackerC.SelectedItem() == nil {
				break
			}
			switch m := b.trackerC.SelectedItem().(*listItem).internal.(type) {
			case *anilist.Anime:
				err := anilist.SetRelation(b.selectedAnime.Name, m)
				if err != nil {
					b.raiseError(err)
					break
				}
				b.stopLoading() // Ensure status is cleared
				b.previousState()
				cmd = b.episodesC.NewStatusMessage(fmt.Sprintf(`Linked to %s %s`, style.Fg(color.Orange)(m.Name()), style.Faint(m.SiteURL)))
				return b, cmd
			case *mal.Anime:
				err := mal.SetRelation(b.selectedAnime.Name, m)
				if err != nil {
					b.raiseError(err)
					break
				}
				b.stopLoading() // Ensure status is cleared
				b.previousState()

				url := fmt.Sprintf("https://myanimelist.net/anime/%d", m.ID)
				cmd = b.episodesC.NewStatusMessage(fmt.Sprintf(`Linked to %s %s`, style.Fg(color.Orange)(m.Title), style.Faint(url)))

				return b, cmd
			}
		case bubblesKey.Matches(msg, b.keymap.back):
			if b.trackerC.FilterState() != list.Unfiltered {
				break
			}
			b.previousState()
			return b, nil
		}
	}

	b.trackerC, cmd = b.trackerC.Update(msg)
	return b, cmd
}

func (b *statefulBubble) updatePostWatch(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case bubblesKey.Matches(msg, b.keymap.up):
			if n := len(b.postWatchC.Items()); n > 0 && b.postWatchC.Index() == 0 {
				b.postWatchC.Select(n - 1)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.down):
			if n := len(b.postWatchC.Items()); n > 0 && b.postWatchC.Index() == n-1 {
				b.postWatchC.Select(0)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.confirm, b.keymap.selectOne):
			if b.postWatchC.SelectedItem() == nil {
				break
			}
			selection := b.postWatchC.SelectedItem().(*listItem).internal.(string)
			switch selection {
			case "Next":
				current := b.currentPlayingEpisode
				if current == nil {
					b.previousState()
					return b, nil
				}
				// Play next chronological episode.
				items := b.episodesC.Items()
				idx := -1
				for i, item := range items {
					ep := item.(*listItem).internal.(*source.Episode)
					if ep.URL == current.URL {
						idx = i
						break
					}
				}

				if idx != -1 && idx+1 < len(items) {
					nextEp := items[idx+1].(*listItem).internal.(*source.Episode)
					b.progressStatus = fmt.Sprintf("Launching %s - %s", b.selectedAnime.Name, nextEp.Name)
					b.currentPlayingEpisode = nextEp
					b.newState(readState)
					return b, tea.Batch(b.readEpisode(nextEp), b.startLoading())
				}
				// If no next episode, notify the user instead of silently returning
				return b, b.postWatchC.NewStatusMessage("Anime completed! No further episodes.")

			case "Replay":
				if b.currentPlayingEpisode != nil {
					b.progressStatus = fmt.Sprintf("Launching %s - %s", b.selectedAnime.Name, b.currentPlayingEpisode.Name)
					b.newState(readState)
					return b, tea.Batch(b.readEpisode(b.currentPlayingEpisode), b.startLoading())
				}

			case "Previous":
				current := b.currentPlayingEpisode
				if current == nil {
					b.previousState()
					return b, nil
				}
				items := b.episodesC.Items()
				idx := -1
				for i, item := range items {
					ep := item.(*listItem).internal.(*source.Episode)
					if ep.URL == current.URL {
						idx = i
						break
					}
				}

				// List is Ascending (Episode 1 at top).
				// Prev Episode (Chronological) -> Go UP the list (idx-1).
				if idx > 0 {
					prevEp := items[idx-1].(*listItem).internal.(*source.Episode)
					b.progressStatus = fmt.Sprintf("Launching %s - %s", b.selectedAnime.Name, prevEp.Name)
					b.currentPlayingEpisode = prevEp
					b.newState(readState)
					return b, tea.Batch(b.readEpisode(prevEp), b.startLoading())
				}
				b.previousState()

			case "Back to Episodes":
				// Return to menu (back)
				b.previousState()
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.back):
			b.previousState()
		}
	}

	b.postWatchC, cmd = b.postWatchC.Update(msg)
	return b, cmd
}

func (b *statefulBubble) updateRead(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case mpvExitMsg:
		b.stopLoading()
		b.mpvPlayer = nil

		if b.currentPlayingEpisode != nil {
			_ = history.Save(b.currentPlayingEpisode, msg.Percentage)

			if activeTracker, err := b.getActiveTracker(); err == nil && activeTracker != nil {
				// Prevent double-sync if MPVWatcher already handled it (e.g., at 100% EOF).
				if b.syncGuard != nil && !b.syncGuard.CompareAndSwap(false, true) {
					return b, cmd
				}

				go func(ep *source.Episode) {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()

					trackerID, totalEpisodes := b.getTrackerMetadata(ep.Anime)
					if trackerID != 0 {
						_ = activeTracker.UpdateEpisodeProgress(ctx, trackerID, int(ep.Index), totalEpisodes)
					}
				}(b.currentPlayingEpisode)
			}
		}

		if b.nextEpisodeToPlay != nil {
			ep := b.nextEpisodeToPlay
			b.nextEpisodeToPlay = nil
			return b, tea.Batch(b.readEpisode(ep), b.startLoading())
		}

		b.newState(postWatchState)
		b.postWatchC.Select(0)
		return b, nil
	case tea.KeyMsg:
		switch {
		case bubblesKey.Matches(msg, b.keymap.back):
			if b.mpvPlayer != nil {
				_ = b.mpvPlayer.Close()
			}
			b.previousState()
			return b, b.stopLoading()
		case bubblesKey.Matches(msg, b.keymap.forceQuit):
			if b.mpvPlayer != nil {
				_ = b.mpvPlayer.Close()
			}
			return b, tea.Quit
		}
	}

	b.spinnerC, cmd = b.spinnerC.Update(msg)
	return b, cmd
}
func (b *statefulBubble) updateError(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case bubblesKey.Matches(msg, b.keymap.quit):
			return b, tea.Quit
		case bubblesKey.Matches(msg, b.keymap.back):
			b.previousState()
			return b, nil
		}
	}
	return b, cmd
}

func (b *statefulBubble) updateManualID(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyEnter:
			idStr := b.idInputC.Value()
			b.previousState()
			b.idInputC.Blur()

			if idStr == "" {
				return b, nil
			}

			id, err := strconv.Atoi(idStr)
			if err != nil {
				b.raiseError(fmt.Errorf("invalid ID: %s must be a number", idStr))
				return b, nil
			}

			if b.selectedAnime == nil {
				return b, nil
			}

			cleanName := b.selectedAnime.Name
			if idx := strings.LastIndex(cleanName, "("); idx != -1 {
				cleanName = strings.TrimSpace(cleanName[:idx])
			}

			b.progressStatus = "Fetching metadata..."
			b.newState(loadingState)
			return b, tea.Batch(b.startLoading(), func() tea.Msg {
				if viper.GetString("tracker.backend") == "mal" {
					m, err := mal.GetByID(id)
					if err != nil {
						return fmt.Errorf("failed to fetch MAL metadata for ID %d: %w", id, err)
					}
					if err := mal.SetRelation(cleanName, m); err != nil {
						return err
					}
					return m
				}

				al, err := anilist.GetByID(id)
				if err != nil {
					return fmt.Errorf("failed to fetch Anilist metadata for ID %d: %w", id, err)
				}
				if err := anilist.SetRelation(cleanName, al); err != nil {
					return err
				}
				return al
			})

		case msg.Type == tea.KeyEsc:
			b.previousState()
			b.idInputC.Blur()
		}
	}

	b.idInputC, cmd = b.idInputC.Update(msg)
	return b, cmd
}

type coverArtMsg string

type cachedArt struct {
	img     image.Image
	summary string
}

func (b *statefulBubble) getDynamicImageSize() (int, int) {
	// Compute layout dimensions.
	targetWidth := b.width / 4
	if targetWidth > 60 {
		targetWidth = 60
	}
	if targetWidth < 30 {
		targetWidth = 30
	}

	targetHeight := b.height / 2
	if targetHeight > 36 {
		targetHeight = 36
	}
	if targetHeight < 16 {
		targetHeight = 16
	}
	return targetWidth, targetHeight
}

func (b *statefulBubble) fetchCoverArtFromURL(url string, summary string) tea.Cmd {
	return func() tea.Msg {
		var (
			img   image.Image
			ansii string
		)

		if url != "" {
			// Check RAM cache.
			if val, ok := b.coverArtCache.Load(url); ok {
				if cached, ok := val.(*cachedArt); ok {
					img = cached.img
					// If the summary is provided but the cache doesn't have it, update it
					if summary != "" && cached.summary == "" {
						cached.summary = summary
					}
					// If the summary is not provided, use the cached one
					if summary == "" {
						summary = cached.summary
					}
				}
			}

			// If cache miss, fetch and decode asynchronously
			if img == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
				if err == nil {
					resp, err := b.httpClient.Do(req)
					if err == nil {
						defer resp.Body.Close()
						if resp.StatusCode == http.StatusOK {
							if decoded, _, err := image.Decode(resp.Body); err == nil {
								img = decoded
								// Persist to RAM cache for subsequent O(1) retrieval
								b.coverArtCache.Store(url, &cachedArt{
									img:     img,
									summary: summary,
								})
							}
						}
					}
				}
			}
		}

		// Perform ANSI rendering based on the LATEST terminal dimensions
		if img != nil {
			w, h := b.getDynamicImageSize()
			ansii = render.RenderCoverArt(img, uint(w), uint(h), b.imageMode)
		}

		// Render soft-wrapped synopsis.
		if summary != "" {
			w, _ := b.getDynamicImageSize()
			r, err := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(w+2),
			)
			if err == nil {
				if out, err := r.Render(summary); err == nil {
					if ansii == "" {
						ansii = out
					} else {
						ansii += "\n" + out
					}
				}
			}
		}

		return coverArtMsg(ansii)
	}
}

func (b *statefulBubble) fetchCoverArt(anime *source.Anime) tea.Cmd {
	if anime == nil {
		return func() tea.Msg { return coverArtMsg("") }
	}

	url := anime.Metadata.Cover.ExtraLarge
	if url == "" {
		url = anime.Metadata.Cover.Large
	}
	if url == "" {
		url = anime.Metadata.Cover.Medium
	}

	return b.fetchCoverArtFromURL(url, anime.Metadata.Summary)
}

// resolveMalID resolves MAL ID for AniSkip.
func (b *statefulBubble) resolveMalID(animeName string) int {
	cleanName := animeName
	if idx := strings.LastIndex(cleanName, "("); idx != -1 {
		cleanName = strings.TrimSpace(cleanName[:idx])
	}

	backend := viper.GetString("tracker.backend")

	if backend == "mal" {
		// Use MAL native ID.
		if cached := mal.GetCachedRelation(cleanName); cached != nil {
			return cached.ID
		}
		if res, err := mal.FindClosest(cleanName); err == nil {
			return res.ID
		}
	} else {
		// Use AniList cross-reference ID (IDMal).
		if cached := anilist.GetCachedRelation(cleanName); cached != nil {
			return cached.IDMal
		}
		if al, err := anilist.FindClosest(cleanName); err == nil {
			return al.IDMal
		}
	}

	return 0 // Yields 0 if unresolved, causing aniskip to gracefully degrade
}

func (b *statefulBubble) applyManualTrackerUpdate(trackerData any) tea.Cmd {
	if b.selectedAnime == nil {
		return nil
	}

	meta := &source.Metadata{}

	switch m := trackerData.(type) {
	case *mal.Anime:
		meta.Title = m.Title
		meta.Status = m.Status
		meta.Episodes = m.NumEpisodes
		if m.Mean > 0 {
			meta.Score = int(m.Mean * 10)
		}
		if m.MainPicture.Large != "" {
			meta.Cover.ExtraLarge = m.MainPicture.Large
		} else if m.MainPicture.Medium != "" {
			meta.Cover.ExtraLarge = m.MainPicture.Medium
		}
	case *anilist.Anime:
		meta.Title = m.Name()
		meta.Status = m.Status
		meta.Score = m.AverageScore
		meta.Episodes = m.Episodes
		if m.CoverImage.ExtraLarge != "" {
			meta.Cover.ExtraLarge = m.CoverImage.ExtraLarge
		} else if m.CoverImage.Large != "" {
			meta.Cover.ExtraLarge = m.CoverImage.Large
		}
		if m.StartDate.Year != 0 {
			meta.StartDate = source.Date{Year: m.StartDate.Year, Month: m.StartDate.Month, Day: m.StartDate.Day}
		}
		meta.Genres = m.Genres
	}

	b.selectedAnime.Metadata = *meta

	var cmd tea.Cmd
	// Force redraw of the active list
	switch b.state {
	case animesState:
		cmd = b.animesC.SetItems(b.animesC.Items())
	case episodesState:
		cmd = b.episodesC.SetItems(b.episodesC.Items())
	case trackerSelectState:
		cmd = b.trackerC.SetItems(b.trackerC.Items())
	}

	// Trigger cover art fetch for the new metadata
	return tea.Batch(cmd, b.fetchCoverArt(b.selectedAnime))
}

// getActiveTracker returns the initialized tracker backend based on configuration.
func (b *statefulBubble) getActiveTracker() (tracker.MediaTracker, error) {
	if !viper.GetBool("tracker.enable") {
		return nil, nil
	}
	activeTracker := tracker.InitializeTracker()
	if activeTracker == nil {
		return nil, fmt.Errorf("tracker initialization failed")
	}
	authCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := activeTracker.CheckAuth(authCtx); err != nil {
		return nil, err
	}
	return activeTracker, nil
}

// getTrackerMetadata retrieves service-specific ID and total episodes for the current anime.
func (b *statefulBubble) getTrackerMetadata(anime *source.Anime) (trackerID int, totalEpisodes int) {
	if anime == nil {
		return 0, 0
	}

	backend := viper.GetString("tracker.backend")
	if backend == "mal" {
		if cached := mal.GetCachedRelation(anime.Name); cached != nil {
			return cached.ID, cached.NumEpisodes
		}
		if res, err := mal.FindClosest(anime.Name); err == nil {
			return res.ID, res.NumEpisodes
		}
	} else {
		if cached := anilist.GetCachedRelation(anime.Name); cached != nil {
			return cached.ID, cached.Episodes
		}
		if al, err := anilist.FindClosest(anime.Name); err == nil {
			return al.ID, al.Episodes
		}
	}
	return 0, 0
}
