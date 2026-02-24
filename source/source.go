// Package source defines the domain models and interfaces for media discovery and retrieval.
package source

// Source defines the required capabilities for a media provider scraping engine.
type Source interface {
	// Name returns the unique identifier for the scraping provider.
	Name() string

	// ID returns the unique identifier of the source.
	ID() string

	// Search executes a query against the provider to discover matching anime entities.
	Search(query string) ([]*Anime, error)

	// EpisodesOf retrieves the complete list of available episodes for a specific anime entity.
	EpisodesOf(anime *Anime) ([]*Episode, error)

	// VideosOf retrieves the available media streams or video fragments for a specific episode.
	VideosOf(episode *Episode) ([]*Video, error)
}
