// Package source defines the domain models and interfaces for media discovery and retrieval.
package source

// Episode represents a discrete media segment within an anime series.
type Episode struct {
	// Source ID (e.g. "1").
	ID string `json:"id"`
	// Display name (e.g. "Episode 1").
	Name string `json:"name"`
	// Direct URL to the episode page.
	URL string `json:"url"`
	// Episode number/index.
	Index uint16 `json:"index"`
	// Volume number (mostly for consistency, often empty for anime).
	Volume string `json:"volume"`

	Anime *Anime `json:"-"`

	// Videos associated with this episode.
	// Populated only when necessary.
	Videos []*Video `json:"videos,omitempty"`
}

// String returns the canonical string representation of the episode identifier.
func (e *Episode) String() string {
	return e.Name
}

// Source returns the parent source.
func (e *Episode) Source() Source {
	if e.Anime == nil {
		return nil
	}
	return e.Anime.Source
}
