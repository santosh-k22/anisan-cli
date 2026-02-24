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
	u.RawQuery = q.Encode()

	resp, err := authenticatedRequest("GET", u.String(), nil)
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

// GetUserList retrieves the authenticated user's anime collection from MyAnimeList.
// The status parameter filters the results: watching, completed, on_hold, dropped, plan_to_watch, or all.
func GetUserList(status string) ([]UserListEntry, error) {
	u, _ := url.Parse(apiEndpoint + "/users/@me/animelist")
	q := u.Query()
	if status != "" && status != "all" {
		q.Set("status", status)
	}
	q.Set("fields", "list_status,num_episodes,start_date,end_date")
	q.Set("limit", "1000") // Fetch up to 1000 items
	u.RawQuery = q.Encode()

	resp, err := authenticatedRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("mal user list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("mal user list error: status %d", resp.StatusCode)
	}

	var result UserList
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("mal user list decode: %w", err)
	}

	return result.Data, nil
}
