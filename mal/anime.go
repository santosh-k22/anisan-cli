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
}

// SearchResult encapsulates a paginated response from the MyAnimeList search endpoint.
type SearchResult struct {
	Data []struct {
		Node Anime `json:"node"`
	} `json:"data"`
}

// UpdateStatus summarizes the current state of an anime entry after a successful list update operation.
type UpdateStatus struct {
	Status             string `json:"status"`
	Score              int    `json:"score"`
	NumWatchedEpisodes int    `json:"num_watched_episodes"`
	IsRewatching       bool   `json:"is_rewatching"`
	UpdatedAt          string `json:"updated_at"`
}

// UserListEntry represents an anime in the user's list.
type UserListEntry struct {
	Node       Anime `json:"node"`
	ListStatus struct {
		Status             string `json:"status"`
		Score              int    `json:"score"`
		NumWatchedEpisodes int    `json:"num_watched_episodes"`
		IsRewatching       bool   `json:"is_rewatching"`
		UpdatedAt          string `json:"updated_at"`
	} `json:"list_status"`
}

// UserList defines a collection of anime entries retrieved from a user's MyAnimeList profile.
type UserList struct {
	Data []UserListEntry `json:"data"`
}
