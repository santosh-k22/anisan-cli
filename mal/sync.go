// Package mal provides a client for the MyAnimeList REST API.
package mal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
)

// UpdateMyListStatus updates the user's progress and status for a specific anime entry on MyAnimeList.
// Valid status values include: watching, completed, on_hold, dropped, and plan_to_watch.
func UpdateMyListStatus(animeID int, episode int, status string) (*UpdateStatus, error) {
	endpoint := fmt.Sprintf("%s/anime/%d/my_list_status", apiEndpoint, animeID)

	data := url.Values{}
	data.Set("num_watched_episodes", strconv.Itoa(episode))
	if status != "" {
		data.Set("status", status)
	}

	resp, err := authenticatedRequest("PATCH", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("mal update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mal update error %d: %s", resp.StatusCode, string(body))
	}

	var result UpdateStatus
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("mal update decode: %w", err)
	}

	return &result, nil
}
