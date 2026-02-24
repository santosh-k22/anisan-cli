// Package tui provides the primary terminal user interface implementation.
package tui

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"runtime"

	"github.com/anisan-cli/anisan/anilist"
	"github.com/anisan-cli/anisan/aniskip"
	"github.com/anisan-cli/anisan/color"
	"github.com/anisan-cli/anisan/history"
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

func (b *statefulBubble) loadHistory() (tea.Cmd, error) {
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
	for _, e := range entries {
		items = append(items, &listItem{
			internal: e,
		})
	}

	// Asynchronously hydrate history entries with remote metadata.
	go func(historyEntries []*history.SavedEpisode) {
		// Group by name to avoid duplicate fetches
		nameMap := make(map[string][]*history.SavedEpisode)
		for _, e := range historyEntries {
			nameMap[e.AnimeName] = append(nameMap[e.AnimeName], e)
		}

		for name, eps := range nameMap {
			// Find closest Anilist match
			al, err := anilist.FindClosest(name)
			if err == nil && al != nil {
				// Convert Anilist to Metadata
				meta := &source.Metadata{
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

				for _, ep := range eps {
					ep.Metadata = meta
				}
			}
		}
	}(entries)

	return tea.Batch(b.historyC.SetItems(items), b.loadProviders()), nil
}

func (b *statefulBubble) loadSources(ps []*provider.Provider) tea.Cmd {
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
		)

		if alAnime, err := anilist.FindClosest(episode.Anime.Name); err == nil {
			anilistID = alAnime.ID
			malID = alAnime.IDMal
		}

		if malID == 0 {
			if _, err := mal.LoadToken(); err == nil {
				log.Info("Searching MAL for " + episode.Anime.Name)
				if res, err := mal.SearchAnime(episode.Anime.Name); err == nil && len(res) > 0 {
					malID = res[0].ID
				}
			}
		}

		if viper.GetBool(key.Aniskip) {
			if malID != 0 {
				log.Infof("Fetching skip times for MAL ID %d Episode %d", malID, episode.Index)
				skipTimes, _ = aniskip.GetSkipTimes(malID, int(episode.Index))
				if skipTimes != nil {
					log.Infof("Skip times found: Intro %v-%v, Outro %v-%v", skipTimes.Opening.Start, skipTimes.Opening.End, skipTimes.Ending.Start, skipTimes.Ending.End)
				} else {
					log.Warn("No skip times found.")
				}
			} else {
				log.Warn("MAL ID not found, skipping intro skip fetch.")
			}
		} else {
			log.Info("Aniskip disabled in config.")
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

		anilistSynced := false
		malSynced := false

		var maxPercentage float64

		// Technical Note: AniSkip and IPC monitoring are exclusive to the MPV player implementation.
		if mpvPlayer, isMPV := b.mpvPlayer.(*player.MPV); isMPV {
			// Initialize the skipper with fetched times (if any)
			skipper := player.NewSkipper(mpvPlayer, skipTimes)
			if err := skipper.ApplyChapters(); err != nil {
				log.Warnf("Failed to apply chapters: %v", err)
			}

			b.mpvPlayer.StopIPCTicker()
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

				if dur > 0 && float64(pos) > float64(dur)*0.8 {
					if !anilistSynced && anilistID != 0 {
						anilistSynced = true
						go func() {
							log.Infof("Syncing progress to Anilist (ID: %d)", anilistID)
							_ = anilist.UpdateMediaListEntry(anilistID, int(episode.Index), anilist.MediaListStatusCurrent)
						}()
					}

					if !malSynced && malID != 0 {
						if _, err := mal.LoadToken(); err == nil {
							malSynced = true
							go func() {
								log.Infof("Syncing progress to MAL (ID: %d)", malID)
								_, err := mal.UpdateMyListStatus(malID, int(episode.Index), "watching")
								if err != nil {
									log.Errorf("MAL sync failed: %v", err)
								}
							}()
						}
					}
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

func (b *statefulBubble) fetchAndSetAnilist(anime *source.Anime) tea.Cmd {
	return func() tea.Msg {
		// Sanitize series name by removing the episode count suffix.
		cleanName := anime.Name
		if idx := strings.LastIndex(cleanName, "("); idx != -1 {
			cleanName = strings.TrimSpace(cleanName[:idx])
		}

		alAnime, err := anilist.FindClosest(cleanName)
		if err != nil {
			log.Warn(err)
		} else {
			b.closestAnilistAnimeChannel <- alAnime
		}
		return nil
	}
}

func (b *statefulBubble) waitForAnilistFetchAndSet() tea.Cmd {
	return func() tea.Msg {
		return <-b.closestAnilistAnimeChannel
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
			b.fetchedAnilistAnimesChannel <- animes
		}
		return nil
	}
}

func (b *statefulBubble) waitForAnilist() tea.Cmd {
	return func() tea.Msg {
		select {
		case found := <-b.fetchedAnilistAnimesChannel:
			return found
		case err := <-b.errorChannel:
			b.lastError = err
			return err
		}
	}
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
			b.fetchedMALAnimesChannel <- animes
		}
		return nil
	}
}

func (b *statefulBubble) waitForMALAnime() tea.Cmd {
	return func() tea.Msg {
		select {
		case found := <-b.fetchedMALAnimesChannel:
			return found
		case err := <-b.errorChannel:
			b.lastError = err
			return err
		}
	}
}
