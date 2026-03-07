// Package mal provides a client for the MyAnimeList REST API.
package mal

// Anime represents an anime entry from the MyAnimeList REST API.
type Anime struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	MainPicture struct {
		Medium string `json:"medium"`
		Large  string `json:"large"`
	} `json:"main_picture"`
	Status      string  `json:"status,omitempty"`
	NumEpisodes int     `json:"num_episodes,omitempty"`
	Mean        float64 `json:"mean,omitempty"`
}

// SearchResult encapsulates a paginated response from the MyAnimeList search endpoint.
type SearchResult struct {
	Data []struct {
		Node Anime `json:"node"`
	} `json:"data"`
}
