// Package tui provides the primary terminal user interface implementation.
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"runtime"

	"github.com/anisan-cli/anisan/anilist"
	"github.com/anisan-cli/anisan/aniskip"
	"github.com/anisan-cli/anisan/color"
	"github.com/anisan-cli/anisan/history"
	"github.com/anisan-cli/anisan/internal/tracker"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/log"
	"github.com/anisan-cli/anisan/mal"
	"github.com/anisan-cli/anisan/player"
	"github.com/anisan-cli/anisan/provider"
	"github.com/anisan-cli/anisan/source"
	"github.com/anisan-cli/anisan/style"
	"github.com/anisan-cli/anisan/util"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/samber/lo"
	"github.com/spf13/viper"
)

func (b *statefulBubble) loadProviders() tea.Cmd {
	providers := provider.Builtins()
	customProviders := provider.Customs()

	var items []list.Item
	for _, p := range providers {
		items = append(items, &listItem{
			internal: p,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.Compare(items[i].FilterValue(), items[j].FilterValue()) < 0
	})

	var customItems []list.Item
	for _, p := range customProviders {
		customItems = append(customItems, &listItem{
			internal: p,
		})
	}
	sort.Slice(customItems, func(i, j int) bool {
		return strings.Compare(customItems[i].FilterValue(), customItems[j].FilterValue()) < 0
	})

	return b.sourcesC.SetItems(append(items, customItems...))
}

// metadataPopulatedMsg is sent by batchPopulateMetadata after each anime's metadata has been fetched.
// Delivering this through the Bubbletea event loop triggers a proper UI re-render.
type metadataPopulatedMsg struct {
	anime *source.Anime
}

// batchPopulateMetadata concurrently fetches metadata for up to the first N animes in the list.
// Each result is delivered as a metadataPopulatedMsg tea.Cmd so the description row re-renders.
func (b *statefulBubble) batchPopulateMetadata(animes []*source.Anime) tea.Cmd {
	limit := len(animes)
	if limit > 10 {
		limit = 10 // Eagerly enrich the first page of visible results
	}
	cmds := make([]tea.Cmd, limit)
	for i, anime := range animes[:limit] {
		anime := anime // capture for closure
		cmds[i] = func() tea.Msg {
			_ = anime.PopulateMetadata(func(string) {})
			return metadataPopulatedMsg{anime: anime}
		}
	}
	return tea.Batch(cmds...)
}

func (b *statefulBubble) loadHistory() (tea.Cmd, error) {
	// Retrieve local history records and sort chronologically for display.
	saved, err := history.Get()
	if err != nil {
		return nil, err
	}

	entries := lo.Values(saved)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].AnimeName == entries[j].AnimeName {
			return strings.Compare(entries[i].Name, entries[j].Name) < 0
		}
		return strings.Compare(entries[i].AnimeName, entries[j].AnimeName) < 0
	})

	var items []list.Item
	var suggestions []string
	seenSuggestions := make(map[string]bool)

	for _, e := range entries {
		items = append(items, &listItem{
			internal: e,
		})
		if !seenSuggestions[e.AnimeName] {
			seenSuggestions[e.AnimeName] = true
			suggestions = append(suggestions, e.AnimeName)
		}
	}
	b.inputC.SetSuggestions(suggestions)

	// Asynchronously hydrate history entries with remote metadata via unified tracker.
	go func(historyEntries []*history.SavedEpisode) {
		// Group by name to avoid duplicate fetches
		nameMap := make(map[string][]*history.SavedEpisode)
		for _, e := range historyEntries {
			nameMap[e.AnimeName] = append(nameMap[e.AnimeName], e)
		}
		backend := viper.GetString("tracker.backend")
		for name, eps := range nameMap {
			var meta *source.Metadata
			if backend == "mal" {
				// Retrieve the closest MyAnimeList result and map it to the generalized Metadata schema.
				if res, err := mal.SearchAnime(name); err == nil && len(res) > 0 {
					m := res[0]
					meta = &source.Metadata{
						Title:    m.Title,
						Status:   m.Status,
						Episodes: m.NumEpisodes,
					}
					if m.Mean > 0 {
						meta.Score = int(m.Mean * 10)
					}
				}
			} else {
				// Retrieve the closest AniList result and map it to the generalized Metadata schema.
				al, err := anilist.FindClosest(name)
				if err == nil && al != nil {
					meta = &source.Metadata{
						Title:    al.Name(),
						Status:   al.Status,
						Score:    al.AverageScore,
						Episodes: al.Episodes,
					}
					if al.StartDate.Year != 0 {
						meta.StartDate = source.Date{
							Year:  al.StartDate.Year,
							Month: al.StartDate.Month,
							Day:   al.StartDate.Day,
						}
					}
				}
			}

			if meta != nil {
				for _, ep := range eps {
					ep.Metadata = meta
				}
			}
		}
	}(entries)

	return tea.Batch(b.historyC.SetItems(items), b.loadProviders()), nil
}

func (b *statefulBubble) loadSources(ps []*provider.Provider) tea.Cmd {
	// Initialize scraper backends concurrently using a WaitGroup to minimize interface blocking.
	return func() tea.Msg {
		var (
			sources = make([]source.Source, len(ps))
			wg      = sync.WaitGroup{}
			mutex   = sync.Mutex{}
			err     error
		)

		wg.Add(len(ps))
		for i, p := range ps {
			go func(i int, p *provider.Provider) {
				defer wg.Done()

				if err != nil {
					return
				}

				log.Info("loading source " + p.ID)
				b.progressStatus = "Initializing source"
				var s source.Source
				s, err = p.CreateSource()

				if err != nil {
					log.Error(err)
					b.errorChannel <- err
					return
				}

				if s == nil {
					log.Errorf("source %s creation returned nil", p.ID)
					return
				}

				log.Info("source " + p.ID + " loaded")

				mutex.Lock()
				sources[i] = s
				mutex.Unlock()
			}(i, p)
		}

		wg.Wait()

		validSources := lo.Filter(sources, func(s source.Source, _ int) bool {
			return s != nil
		})

		if len(validSources) == 0 && len(ps) > 0 {
			if err != nil {
				return err
			}
			return fmt.Errorf("failed to load any sources")
		}

		b.sourcesLoadedChannel <- validSources
		return nil
	}
}

func (b *statefulBubble) waitForSourcesLoaded() tea.Cmd {
	// Block until the sourcesLoadedChannel yields the initialized scraper backends.
	return func() tea.Msg {
		select {
		case res := <-b.sourcesLoadedChannel:
			return res
		case err := <-b.errorChannel:
			b.lastError = err
			return err
		}
	}
}

func (b *statefulBubble) searchAnime(query string) tea.Cmd {
	// Execute a fan-out search across all active providers and aggregate results into a unified set.
	return func() tea.Msg {
		log.Info("searching for " + query)
		b.progressStatus = fmt.Sprintf("Searching among %s", util.Quantify(len(b.selectedSources), "source", "sources"))

		var animes = make([]*source.Anime, 0)
		var mutex sync.Mutex

		wg := sync.WaitGroup{}
		wg.Add(len(b.selectedSources))
		for _, s := range b.selectedSources {
			go func(s source.Source) {
				defer wg.Done()
				sourceAnimes, err := s.Search(query)

				if err != nil {
					log.Error(err)
					b.errorChannel <- err
					return
				}

				log.Infof("found %s from source %s", util.Quantify(len(sourceAnimes), "anime", "animes"), s.Name())
				mutex.Lock()
				animes = append(animes, sourceAnimes...)
				mutex.Unlock()
			}(s)
		}

		wg.Wait()

		log.Infof("found %d animes from %d sources", len(animes), len(b.selectedSources))
		b.foundAnimesChannel <- animes
		return nil
	}
}

func (b *statefulBubble) waitForAnimes() tea.Cmd {
	return func() tea.Msg {
		select {
		case found := <-b.foundAnimesChannel:
			return found
		case err := <-b.errorChannel:
			b.lastError = err
			return err
		}
	}
}

func (b *statefulBubble) getEpisodes(anime *source.Anime) tea.Cmd {
	// Fetch the canonical episode list from the anime's origin source.
	return func() tea.Msg {
		log.Info("getting episodes of " + anime.Name)
		episodes, err := anime.Source.EpisodesOf(anime)
		if err != nil {
			log.Error(err)
			b.errorChannel <- err
		} else {
			log.Infof("found %s", util.Quantify(len(episodes), "episode", "episodes"))
			b.foundEpisodesChannel <- episodes
		}

		return nil
	}
}

func (b *statefulBubble) waitForEpisodes() tea.Cmd {
	return func() tea.Msg {
		select {
		case found := <-b.foundEpisodesChannel:
			return found
		case err := <-b.errorChannel:
			b.lastError = err
			return err
		}
	}
}

func (b *statefulBubble) readEpisode(episode *source.Episode) tea.Cmd {
	return func() tea.Msg {
		b.currentPlayingEpisode = episode

		// Persist playback initiation to the user's history record.
		_ = history.Save(episode, 0.0)

		title := fmt.Sprintf("%s - %s", episode.Anime.Name, episode.Name)

		log.Infof("Playing %s via mpv IPC", title)
		b.progressStatus = fmt.Sprintf("Launching %s", style.Fg(color.Purple)(title))

		var (
			skipTimes *aniskip.SkipTimes
			anilistID int
			malID     int
			totalEps  int
		)
		backend := viper.GetString("tracker.backend")
		// Rapid Cache Resolution: Prioritize local relation mappings to minimize blocking on network I/O.
		if backend == "mal" {
			if m := mal.GetCachedRelation(episode.Anime.Name); m != nil {
				malID = m.ID
				totalEps = m.NumEpisodes
			} else if res, err := mal.FindClosest(episode.Anime.Name); err == nil {
				malID = res.ID
				totalEps = res.NumEpisodes
			}
		} else {
			if al, err := anilist.FindClosest(episode.Anime.Name); err == nil {
				anilistID = al.ID
				malID = al.IDMal
				totalEps = al.Episodes
			}
		}
		// Aniskip execution (strictly requires MAL ID)
		if viper.GetBool(key.Aniskip) {
			resolvedMalID := b.resolveMalID(episode.Anime.Name)
			if resolvedMalID != 0 {
				log.Infof("Fetching skip times for MAL ID %d Episode %d", resolvedMalID, episode.Index)
				skipTimes, _ = aniskip.GetSkipTimes(resolvedMalID, int(episode.Index))
				if skipTimes != nil {
					log.Infof("Skip times found: Intro %v-%v, Outro %v-%v", skipTimes.Opening.Start, skipTimes.Opening.End, skipTimes.Ending.Start, skipTimes.Ending.End)
				}
			} else {
				log.Warn("MAL ID not found, skipping intro skip fetch.")
			}
		}

		if b.mpvPlayer == nil {
			if viper.GetString(key.Player) == "iina" && runtime.GOOS == "darwin" {
				b.mpvPlayer = player.NewIINA()
			} else {
				b.mpvPlayer = player.NewMPV()
			}
		}

		videoURL := episode.URL
		var headers map[string]string

		log.Infof("Fetching videos for episode %s", episode.Name)
		videos, err := episode.Source().VideosOf(episode)
		if err == nil && len(videos) > 0 {
			videoURL = videos[0].URL
			headers = videos[0].Headers
			if videos[0].Quality != "" {
				log.Infof("Selected video: %s (%s)", videoURL, videos[0].Quality)
			} else {
				log.Infof("Selected video: %s", videoURL)
			}
		} else {
			if err != nil {
				log.Warnf("VideosOf failed: %v, falling back to episode URL", err)
			} else {
				log.Warnf("VideosOf returned no videos, falling back to episode URL")
			}
		}

		err = b.mpvPlayer.Play(videoURL, title, headers)
		if err != nil {
			log.Errorf("failed to play episode: %v", err)
			b.errorChannel <- fmt.Errorf("mpv playback failed: %w", err)
			return nil
		}

		var maxPercentage float64

		// Technical Note: AniSkip and IPC monitoring are exclusive to the MPV player implementation.
		if mpvPlayer, isMPV := b.mpvPlayer.(*player.MPV); isMPV {
			// Initialize the skipper with fetched times (if any)
			skipper := player.NewSkipper(mpvPlayer, skipTimes)
			if err := skipper.ApplyChapters(); err != nil {
				log.Warnf("Failed to apply chapters: %v", err)
			}

			b.mpvPlayer.StopIPCTicker()

			// Instantiate the resolved tracking backend.
			activeTracker := tracker.InitializeTracker()
			// Resolve the canonical media identifier for the active tracking backend.
			var trackerMediaID = anilistID
			if viper.GetString(key.TrackerBackend) == "mal" {
				trackerMediaID = malID
			}

			if trackerMediaID != 0 {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Launch the decoupled IPC watcher to monitor playback progress asynchronously.
				watcher := player.NewMPVWatcher(mpvPlayer.Socket(), activeTracker, trackerMediaID, int(episode.Index), totalEps)
				go func() {
					if err := watcher.Poll(ctx); err != nil && err != context.Canceled {
						log.Warnf("IPC watcher terminated: %v", err)
					}
				}()
			}

			b.mpvPlayer.StartIPCTicker(func(pos, dur int) {
				if dur > 0 {
					p := (float64(pos) / float64(dur)) * 100.0
					if p > maxPercentage {
						maxPercentage = p
					}
				}

				// Monitor playback position for automated intro/outro skipping.

				if _, err := skipper.Check(float64(pos)); err != nil {
					log.Warnf("Skipper check failed: %v", err)
				}
			})
		} else {
			// For non-IPC players like IINA, we just assume it was fully watched
			// if the application was cleanly launched and executed to completion.
			maxPercentage = 100.0
		}

		log.Infof("mpv launched on socket %s", b.mpvPlayer.Socket())
		return b.waitForMpvExit(&maxPercentage)()
	}
}

func (b *statefulBubble) waitForEpisodeRead() tea.Cmd {
	return func() tea.Msg {
		select {
		case res := <-b.episodeReadChannel:
			return res
		case err := <-b.errorChannel:
			b.lastError = err
			return err
		}
	}
}

type mpvExitMsg struct {
	Percentage float64
}

func (b *statefulBubble) waitForMpvExit(maxPercentage *float64) tea.Cmd {
	return func() tea.Msg {
		<-b.mpvPlayer.Wait()
		return mpvExitMsg{Percentage: *maxPercentage}
	}
}

// fetchAndSetTracker dynamically links either MAL or AniList in the background based on viper config.
func (b *statefulBubble) fetchAndSetTracker(anime *source.Anime) tea.Cmd {
	return func() tea.Msg {
		cleanName := anime.Name
		if idx := strings.LastIndex(cleanName, "("); idx != -1 {
			cleanName = strings.TrimSpace(cleanName[:idx])
		}

		backend := viper.GetString("tracker.backend")

		if backend == "mal" {
			res, err := mal.SearchAnime(cleanName)
			if err == nil && len(res) > 0 {
				_ = mal.SetRelation(anime.Name, &res[0])
				b.closestTrackerAnimeChannel <- &res[0]
			}
			return nil
		}

		alAnime, err := anilist.FindClosest(cleanName)
		if err == nil {
			b.closestTrackerAnimeChannel <- alAnime
		}
		return nil
	}
}

// waitForTrackerFetchAndSet unifies the channel listener.
func (b *statefulBubble) waitForTrackerFetchAndSet() tea.Cmd {
	return func() tea.Msg {
		return <-b.closestTrackerAnimeChannel
	}
}

func (b *statefulBubble) tryLoadAnilistCache(anime *source.Anime) tea.Cmd {
	return func() tea.Msg {
		// Sanitize series name by removing the episode count suffix.
		cleanName := anime.Name
		if idx := strings.LastIndex(cleanName, "("); idx != -1 {
			cleanName = strings.TrimSpace(cleanName[:idx])
		}

		if alAnime := anilist.GetCachedRelation(cleanName); alAnime != nil {
			return alAnime
		}
		return nil
	}
}

// tryLoadMALCache retrieves locally persisted relation mappings for MyAnimeList.
func (b *statefulBubble) tryLoadMALCache(anime *source.Anime) tea.Cmd {
	return func() tea.Msg {
		// Enforce the same string normalization utilized in manual linking
		cleanName := anime.Name
		if idx := strings.LastIndex(cleanName, "("); idx != -1 {
			cleanName = strings.TrimSpace(cleanName[:idx])
		}

		cached := mal.GetCachedRelation(cleanName)
		if cached != nil {
			return cached // Yields *mal.Anime to the update loop
		}

		return nil
	}
}

type anilistTrackerFetchMsg struct {
	animes    []*anilist.Anime
	closestID int
}

func (b *statefulBubble) fetchAnilist(anime *source.Anime) tea.Cmd {
	return func() tea.Msg {
		// Strip "(XX eps)" suffix from name for cleaner search
		cleanName := anime.Name
		if idx := strings.LastIndex(cleanName, "("); idx != -1 {
			cleanName = strings.TrimSpace(cleanName[:idx])
		}

		log.Info("fetching anilist for " + cleanName)
		b.progressStatus = fmt.Sprintf("Fetching anilist for %s", style.Fg(color.Purple)(cleanName))
		animes, err := anilist.SearchByName(cleanName)
		if err != nil {
			log.Error(err)
			b.errorChannel <- err
		} else {
			log.Infof("found %s", util.Quantify(len(animes), "anime", "animes"))
			closest, err := anilist.FindClosest(cleanName)
			id := -1
			if err == nil {
				id = closest.ID
			}
			b.fetchedTrackerAnimesChannel <- anilistTrackerFetchMsg{
				animes:    animes,
				closestID: id,
			}
		}
		return nil
	}
}

func (b *statefulBubble) waitForAnilist() tea.Cmd {
	return func() tea.Msg {
		select {
		case found := <-b.fetchedTrackerAnimesChannel:
			return found
		case err := <-b.errorChannel:
			b.lastError = err
			return err
		}
	}
}

type malTrackerFetchMsg struct {
	animes    []mal.Anime
	closestID int
}

func (b *statefulBubble) fetchMALAnime(query string) tea.Cmd {
	return func() tea.Msg {
		// Sanitize series name by removing the episode count suffix.
		cleanName := query
		if idx := strings.LastIndex(cleanName, "("); idx != -1 {
			cleanName = strings.TrimSpace(cleanName[:idx])
		}

		log.Info("searching MAL for " + cleanName)
		b.progressStatus = fmt.Sprintf("Searching MAL for %s", style.Fg(color.Purple)(cleanName))
		animes, err := mal.SearchAnime(cleanName)
		if err != nil {
			log.Error(err)
			b.errorChannel <- err
		} else {
			log.Infof("found %d MAL entries", len(animes))
			closest, err := mal.FindClosest(cleanName)
			id := -1
			if err == nil {
				id = closest.ID
			}
			b.fetchedTrackerAnimesChannel <- malTrackerFetchMsg{
				animes:    animes,
				closestID: id,
			}
		}
		return nil
	}
}

func (b *statefulBubble) waitForMALAnime() tea.Cmd {
	return func() tea.Msg {
		select {
		case found := <-b.fetchedTrackerAnimesChannel:
			return found
		case err := <-b.errorChannel:
			b.lastError = err
			return err
		}
	}
}
