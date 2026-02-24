package history

import (
	"fmt"

	"github.com/anisan-cli/anisan/source"
)

// SavedEpisode represents a single playback entry preserved in the user's history.
type SavedEpisode struct {
	SourceID           string  `json:"source_id"`
	AnimeName          string  `json:"anime_name"`
	AnimeURL           string  `json:"anime_url"`
	AnimeEpisodesTotal int     `json:"anime_episodes_total"`
	Name               string  `json:"name"`
	URL                string  `json:"url"`
	ID                 string  `json:"id"`
	Index              int     `json:"index"`
	AnimeID            string  `json:"anime_id"`
	WatchedPercentage  float64 `json:"watched_percentage"`

	// Metadata contains technical details populated at runtime; not persisted to disk.
	Metadata *source.Metadata `json:"-"`
}

func (s *SavedEpisode) encode() string {
	return fmt.Sprintf("%s (%s)", s.AnimeName, s.SourceID)
}

func (s *SavedEpisode) String() string {
	return fmt.Sprintf("%s : %d / %d", s.AnimeName, s.Index, s.AnimeEpisodesTotal)
}

func newSavedEpisode(episode *source.Episode) *SavedEpisode {
	return &SavedEpisode{
		SourceID:           episode.Anime.Source.ID(),
		AnimeName:          episode.Anime.Name,
		AnimeURL:           episode.Anime.URL,
		Name:               episode.Name,
		URL:                episode.URL,
		ID:                 episode.ID,
		AnimeID:            episode.Anime.ID,
		AnimeEpisodesTotal: len(episode.Anime.Episodes),
		Index:              int(episode.Index),
	}
}
