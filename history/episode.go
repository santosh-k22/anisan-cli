package history

import (
	"fmt"

	"github.com/anisan-cli/anisan/source"
)

// SavedEpisode represents a single playback entry preserved in the user's history.
type SavedEpisode struct {
	SourceID           string   `json:"source_id"`
	AnimeName          string   `json:"anime_name"`
	AnimeURL           string   `json:"anime_url"`
	AnimeEpisodesTotal int      `json:"anime_episodes_total"`
	Name               string   `json:"name"`
	URL                string   `json:"url"`
	ID                 string   `json:"id"`
	Index              int      `json:"index"`
	AnimeID            string   `json:"anime_id"`
	WatchedPercentage  float64  `json:"watched_percentage"`
	Score              int      `json:"score"`
	Status             string   `json:"status"`
	Genres             []string `json:"genres"`
	CoverURL           string   `json:"cover_url"` // Persistent high-fidelity cover image URL for offline viewing.

	// Metadata contains technical details populated at runtime; not persisted to disk.
	Metadata *source.Metadata `json:"-"`
}

func (s *SavedEpisode) encode() string {
	// encode generates a unique identifier for deduplicating history records.
	return fmt.Sprintf("%s (%s)", s.AnimeName, s.SourceID)
}

func (s *SavedEpisode) String() string {
	return fmt.Sprintf("%s : %d / %d", s.AnimeName, s.Index, s.AnimeEpisodesTotal)
}

// newSavedEpisode constructs a new persistent history entry from a live episode source,
// capturing essential metadata (Score, Status, Genres, Cover) for offline display.
func newSavedEpisode(episode *source.Episode) *SavedEpisode {
	saved := &SavedEpisode{
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

	saved.Score = episode.Anime.Metadata.Score
	saved.Status = episode.Anime.Metadata.Status
	saved.Genres = episode.Anime.Metadata.Genres

	if episode.Anime.Metadata.Cover.ExtraLarge != "" {
		saved.CoverURL = episode.Anime.Metadata.Cover.ExtraLarge
	} else if episode.Anime.Metadata.Cover.Large != "" {
		saved.CoverURL = episode.Anime.Metadata.Cover.Large
	} else if episode.Anime.Metadata.Cover.Medium != "" {
		saved.CoverURL = episode.Anime.Metadata.Cover.Medium
	}

	return saved
}
