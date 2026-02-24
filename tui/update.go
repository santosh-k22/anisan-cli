// Package tui provides the primary terminal user interface implementation.
package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/anisan-cli/anisan/anilist"
	"github.com/anisan-cli/anisan/color"
	"github.com/anisan-cli/anisan/history"
	intAnilist "github.com/anisan-cli/anisan/integration/anilist"
	"github.com/anisan-cli/anisan/internal/ui"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/mal"
	"github.com/anisan-cli/anisan/open"
	"github.com/anisan-cli/anisan/provider"
	"github.com/anisan-cli/anisan/query"
	"github.com/anisan-cli/anisan/source"
	"github.com/anisan-cli/anisan/style"
	bubblesKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/spf13/viper"
)

func (b *statefulBubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Process Ephemeral UI Notifications (captures `string` and `ui.ClearNotificationMsg`)
	if uiCmd := b.notifier.Update(msg); uiCmd != nil {
		cmd = tea.Batch(cmd, uiCmd)
	}

	switch msg := msg.(type) {
	case provider.ScraperUpdatedMsg:
		// Provider updates are reloaded asynchronously.
		return b, b.loadProviders()
	case error:
		b.raiseError(msg)
	case tea.WindowSizeMsg:
		b.resize(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch {
		case bubblesKey.Matches(msg, b.keymap.forceQuit):
			return b, tea.Quit
		}

		// Input Guard: Ignore non-priority keys during asynchronous operations.
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
			case anilistSelectState:
				if b.anilistC.FilterState() != list.Unfiltered {
					b.anilistC, cmd = b.anilistC.Update(msg)
					return b, cmd
				}
				cmd = onListBack(&b.anilistC)
			case malSelectState:
				if b.malListC.FilterState() != list.Unfiltered {
					b.malListC, cmd = b.malListC.Update(msg)
					return b, cmd
				}
				cmd = onListBack(&b.malListC)
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
	case anilistSelectState:
		return b.updateAnilistSelect(msg)
	case malSelectState:
		return b.updateMALSelect(msg)
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
	case []*anilist.Anime:
		closest, err := anilist.FindClosest(b.selectedAnime.Name)
		id := -1
		if err == nil {
			id = closest.ID
		}

		items := make([]list.Item, len(msg))
		var marked int
		for i, al := range msg {
			if al.ID == id {
				marked = i
			}
			items[i] = &listItem{
				internal: al,
				marked:   al.ID == id,
			}
		}

		cmd = b.anilistC.SetItems(items)
		b.newState(anilistSelectState)
		b.anilistC.Select(marked)
		return b, tea.Batch(cmd, b.stopLoading())
	case []mal.Anime:
		items := make([]list.Item, len(msg))
		for i := range msg {
			items[i] = &listItem{internal: &msg[i]}
		}

		cmd = b.malListC.SetItems(items)
		b.newState(malSelectState)
		return b, tea.Batch(cmd, b.stopLoading())
	case []*source.Anime:
		items := make([]list.Item, len(msg))
		for i, m := range msg {
			items[i] = &listItem{internal: m}
		}

		cmds = append(cmds, b.animesC.SetItems(items))
		b.newState(animesState)
		b.stopLoading()

		// Asynchronously fetch metadata for each search result to ensure Description() renders accurately.
		go func(animes []*source.Anime) {
			for _, anime := range animes {
				_ = anime.PopulateMetadata(func(s string) {})
			}
		}(msg)
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

		// Directly fetch episodes for history entries, bypassing the search results view.
		// since the user already chose this anime from history.
		b.progressStatus = fmt.Sprintf("Loading episodes for %s...", anime.Name)
		b.newState(loadingState)
		return b, tea.Batch(b.getEpisodes(anime), b.waitForEpisodes(), b.startLoading())

	case []*source.Episode:
		// Episodes loaded from history — transition directly to episodesState.
		// This skips the redundant single-anime "Anime Results" screen.
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

		if epToPlay != nil {
			// Select the episode in the list so "Back to Episodes" returns to the correct cursor position.
			b.episodesC.Select(epIdx)

			// Initiate immediate playback for chronological history resumption.
			// Push episodesState so post-watch "Back to Episodes" lands correctly.
			b.currentPlayingEpisode = epToPlay
			b.newState(episodesState)
			b.newState(readState)
			b.stopLoading()

			var anilistCmd tea.Cmd
			if viper.GetBool(key.AnilistLinkOnAnimeSelect) {
				anilistCmd = tea.Batch(b.fetchAndSetAnilist(b.selectedAnime), b.waitForAnilistFetchAndSet())
			} else {
				anilistCmd = b.tryLoadAnilistCache(b.selectedAnime)
			}
			return b, tea.Batch(cmd, b.readEpisode(epToPlay), b.startLoading(), anilistCmd)
		} else {
			// If no episodes were found at all, just fall back to the empty episodes list view
			b.newState(episodesState)
			b.stopLoading()
			return b, cmd
		}
	// Episodes are now handled by updateAnimes (standard flow)
	case tea.KeyMsg:
		switch {

		case bubblesKey.Matches(msg, b.keymap.up):
			if n := len(b.historyC.Items()); n > 0 && b.historyC.Index() == 0 {
				b.historyC.Select(n - 1)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.down):
			p := b.historyC.Items()
			if len(p) > 0 && b.historyC.Index() == len(p)-1 {
				b.historyC.Select(0)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.openURL):
			if b.historyC.SelectedItem() != nil {
				entry := b.historyC.SelectedItem().(*listItem).internal.(*history.SavedEpisode)
				err := open.Run(entry.URL)
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
				return b, tea.Batch(b.startLoading(), b.loadSources([]*provider.Provider{p}), b.waitForSourcesLoaded())
			}
		case bubblesKey.Matches(msg, b.keymap.back):
			b.previousState()
			return b, nil
		}
	}

	b.historyC, cmd = b.historyC.Update(msg)
	return b, cmd
}

func (b *statefulBubble) updateSources(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {

		case bubblesKey.Matches(msg, b.keymap.up):
			if n := len(b.sourcesC.Items()); n > 0 && b.sourcesC.Index() == 0 {
				b.sourcesC.Select(n - 1)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.down):
			p := b.sourcesC.Items()
			if n := len(p); n > 0 && b.sourcesC.Index() == n-1 {
				b.sourcesC.Select(0)
				return b, nil
			}
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
			b.progressStatus = fmt.Sprintf("Searching for %s...", b.inputC.Value())
			b.startLoading()
			b.newState(loadingState)
			go query.Remember(b.inputC.Value(), 1)
			return b, tea.Batch(b.searchAnime(b.inputC.Value()), b.waitForAnimes(), b.spinnerC.Tick)
		case bubblesKey.Matches(msg, b.keymap.acceptSearchSuggestion) && b.searchSuggestion.IsPresent():
			b.inputC.SetValue(b.searchSuggestion.MustGet())
			b.searchSuggestion = mo.None[string]()
			b.inputC.SetCursor(len(b.inputC.Value()))
			return b, nil
		case bubblesKey.Matches(msg, b.keymap.back):
			b.previousState()
			return b, nil
		}
	}

	b.inputC, cmd = b.inputC.Update(msg)

	if b.inputC.Value() != "" {
		if suggestion, ok := query.Suggest(b.inputC.Value()).Get(); ok && suggestion != b.inputC.Value() {
			b.searchSuggestion = mo.Some(suggestion)
		} else {
			b.searchSuggestion = mo.None[string]()
		}
	} else if b.searchSuggestion.IsPresent() {
		b.searchSuggestion = mo.None[string]()
	}

	return b, cmd
}

func (b *statefulBubble) updateAnimes(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case bubblesKey.Matches(msg, b.keymap.changeSource):
			b.newState(sourcesState)
			return b, b.loadProviders()

		case bubblesKey.Matches(msg, b.keymap.up):
			if n := len(b.animesC.Items()); n > 0 && b.animesC.Index() == 0 {
				b.animesC.Select(n - 1)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.down):
			if n := len(b.animesC.Items()); n > 0 && b.animesC.Index() == n-1 {
				b.animesC.Select(0)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.confirm, b.keymap.selectOne):
			if b.animesC.SelectedItem() == nil {
				break
			}
			m, _ := b.animesC.SelectedItem().(*listItem).internal.(*source.Anime)
			b.selectedAnime = m
			b.progressStatus = fmt.Sprintf("Loading episodes for %s...", m.Name)
			go query.Remember(m.Name, 2)
			return b, tea.Batch(b.getEpisodes(m), b.waitForEpisodes(), b.startLoading())

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
				b.currentPlayingEpisode = epToPlay
				b.newState(readState)
				finalCmd = tea.Batch(cmd, b.readEpisode(epToPlay), b.startLoading())
			}
		} else {
			finalCmd = cmd
		}

		if viper.GetBool(key.AnilistLinkOnAnimeSelect) {
			// If auto-link is ON, fetchAndSetAnilist will search online if cache misses
			// It checks cache internally, so we don't need explicit tryLoadAnilistCache
			return b, tea.Batch(finalCmd, b.fetchAndSetAnilist(b.selectedAnime), b.waitForAnilistFetchAndSet())
		}

		// If auto-link is OFF, just check cache (no network)
		return b, tea.Batch(finalCmd, b.tryLoadAnilistCache(b.selectedAnime))
	}

	b.animesC, cmd = b.animesC.Update(msg)
	return b, cmd
}

func (b *statefulBubble) updateEpisodes(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case *anilist.Anime:
		b.anilistAnime = msg
		cmd = b.episodesC.NewStatusMessage(fmt.Sprintf(`Linked to %s %s`, style.Fg(color.Orange)(msg.Name()), style.Faint(msg.SiteURL)))
		return b, cmd
	case tea.KeyMsg:
		switch {

		case bubblesKey.Matches(msg, b.keymap.up):
			if n := len(b.episodesC.Items()); n > 0 && b.episodesC.Index() == 0 {
				b.episodesC.Select(n - 1)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.down):
			if n := len(b.episodesC.Items()); n > 0 && b.episodesC.Index() == n-1 {
				b.episodesC.Select(0)
				return b, nil
			}
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
		case bubblesKey.Matches(msg, b.keymap.anilistSelect):
			if b.anilistAnime != nil {
				if err := open.Start(b.anilistAnime.SiteURL); err != nil {
					b.raiseError(err)
				}
				return b, nil
			}
			b.newState(anilistSelectState)
			return b, tea.Batch(b.startLoading(), b.fetchAnilist(b.selectedAnime), b.waitForAnilist())
		case bubblesKey.Matches(msg, b.keymap.malSelect):
			b.newState(loadingState)
			return b, tea.Batch(b.startLoading(), b.fetchMALAnime(b.selectedAnime.Name), b.waitForMALAnime())
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
				b.currentPlayingEpisode = episode
				b.newState(readState)
				return b, tea.Batch(b.readEpisode(episode), b.waitForEpisodeRead(), b.startLoading())
			}
		}
	}

	b.episodesC, cmd = b.episodesC.Update(msg)
	return b, cmd
}

func (b *statefulBubble) updateAnilistSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {

		case bubblesKey.Matches(msg, b.keymap.up):
			if n := len(b.anilistC.Items()); n > 0 && b.anilistC.Index() == 0 {
				b.anilistC.Select(n - 1)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.down):
			if n := len(b.anilistC.Items()); n > 0 && b.anilistC.Index() == n-1 {
				b.anilistC.Select(0)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.openURL):
			if b.anilistC.SelectedItem() == nil {
				break
			}
			m, _ := b.anilistC.SelectedItem().(*listItem).internal.(*anilist.Anime)
			err := open.Start(m.SiteURL)
			if err != nil {
				b.raiseError(err)
			}
		case bubblesKey.Matches(msg, b.keymap.confirm, b.keymap.selectOne):
			if b.anilistC.SelectedItem() == nil {
				break
			}
			al := b.anilistC.SelectedItem().(*listItem).internal.(*anilist.Anime)
			err := anilist.SetRelation(b.selectedAnime.Name, al)
			if err != nil {
				b.raiseError(err)
				break
			}
			b.previousState()
			cmd = b.episodesC.NewStatusMessage(fmt.Sprintf(`Linked to %s %s`, style.Fg(color.Orange)(al.Name()), style.Faint(al.SiteURL)))
			return b, cmd
		case bubblesKey.Matches(msg, b.keymap.back):
			b.previousState()
			return b, nil
		}
	}

	b.anilistC, cmd = b.anilistC.Update(msg)
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
				// The episode collection is sorted in ascending chronological order.
				// Next Episode (Chronological) -> Select subsequent index (idx+1).
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
					b.currentPlayingEpisode = nextEp
					b.newState(readState)
					return b, tea.Batch(b.readEpisode(nextEp), b.startLoading())
				}
				// If no next episode, go back to episode list
				b.previousState()

			case "Replay":
				if b.currentPlayingEpisode != nil {
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

			if viper.GetBool(key.AnilistEnable) {
				integrator := intAnilist.New()
				err := integrator.MarkWatched(b.currentPlayingEpisode)
				if err != nil && err.Error() == "sync_queued" {
					return b, ui.NotifySyncFailure()
				}
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

func (b *statefulBubble) updateMALSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {

		case bubblesKey.Matches(msg, b.keymap.up):
			if n := len(b.malListC.Items()); n > 0 && b.malListC.Index() == 0 {
				b.malListC.Select(n - 1)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.down):
			if n := len(b.malListC.Items()); n > 0 && b.malListC.Index() == n-1 {
				b.malListC.Select(0)
				return b, nil
			}
		case bubblesKey.Matches(msg, b.keymap.openURL):
			if b.malListC.SelectedItem() == nil {
				break
			}
			m, _ := b.malListC.SelectedItem().(*listItem).internal.(*mal.Anime)
			url := fmt.Sprintf("https://myanimelist.net/anime/%d", m.ID)
			err := open.Start(url)
			if err != nil {
				b.raiseError(err)
			}
		case bubblesKey.Matches(msg, b.keymap.confirm, b.keymap.selectOne):
			if b.malListC.SelectedItem() == nil {
				break
			}
			al := b.malListC.SelectedItem().(*listItem).internal.(*mal.Anime)
			err := mal.SetRelation(b.selectedAnime.Name, al)
			if err != nil {
				b.raiseError(err)
				break
			}
			b.previousState()
			// Show status message?
			cmd = b.episodesC.NewStatusMessage(fmt.Sprintf("Linked to %s (MAL)", style.Fg(color.Orange)(al.Title)))
			return b, cmd
		}
	}

	b.malListC, cmd = b.malListC.Update(msg)
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

			// Fetch the full anime metadata from Anilist using the ID
			// A hollow struct will not persist properly in the cache.
			al, err := anilist.GetByID(id)
			if err != nil {
				b.raiseError(fmt.Errorf("failed to fetch Anilist metadata for ID %d: %w", id, err))
				return b, nil
			}

			// Clean anime name exactly like fetchAndSetAnilist
			cleanName := b.selectedAnime.Name
			if idx := strings.LastIndex(cleanName, "("); idx != -1 {
				cleanName = strings.TrimSpace(cleanName[:idx])
			}

			err = anilist.SetRelation(cleanName, al)
			if err != nil {
				b.raiseError(err)
				return b, nil
			}

			msgCmd := func() tea.Msg { return al }

			if b.state == episodesState {
				return b, tea.Batch(b.episodesC.NewStatusMessage(fmt.Sprintf("Manually linked to ID %d", id)), msgCmd)
			}
			return b, tea.Batch(b.animesC.NewStatusMessage(fmt.Sprintf("Manually linked to ID %d", id)), msgCmd)

		case msg.Type == tea.KeyEsc:
			b.previousState()
			b.idInputC.Blur()
		}
	}

	b.idInputC, cmd = b.idInputC.Update(msg)
	return b, cmd
}
