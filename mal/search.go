// Package mal provides a client for the MyAnimeList REST API.
package mal

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// SearchAnime executes a search for anime titles on the MyAnimeList service.
func SearchAnime(query string) ([]Anime, error) {
	u, _ := url.Parse(apiEndpoint + "/anime")
	q := u.Query()
	q.Set("q", query)
	q.Set("limit", "5")
	q.Set("fields", "status,num_episodes,mean")
	u.RawQuery = q.Encode()

	resp, err := authenticatedRequest("GET", u.String(), "")
	if err != nil {
		return nil, fmt.Errorf("mal search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("mal search error: status %d", resp.StatusCode)
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("mal search decode: %w", err)
	}

	animes := make([]Anime, 0, len(result.Data))
	for _, node := range result.Data {
		animes = append(animes, node.Node)
	}
	return animes, nil
}
