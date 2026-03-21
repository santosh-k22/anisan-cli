package anilist

import (
	"context"
	"encoding/json"
	"fmt"

	al "github.com/anisan-cli/anisan/anilist"
	"github.com/anisan-cli/anisan/internal/sync"
)

// Client implements the MediaTracker interface by wrapping the legacy Anilist integration logic.
type Client struct{}

// NewClient returns an initialized instance of the Anilist tracker client.
func NewClient() *Client {
	return &Client{}
}

var saveMediaListEntryMutation = `
mutation ($mediaId: Int, $progress: Int, $status: MediaListStatus) {
    SaveMediaListEntry (mediaId: $mediaId, progress: $progress, status: $status) {
        id
        progress
        status
    }
}
`

// UpdateEpisodeProgress synchronizes the media progress state with the Anilist API.
// It maps the canonical interface call to the internal mutation engine, and implements offline queue resilience.
func (c *Client) UpdateEpisodeProgress(ctx context.Context, id int, episode int, totalEpisodes int) error {
	status := al.MediaListStatusCurrent
	if totalEpisodes > 0 && episode >= totalEpisodes {
		status = al.MediaListStatusCompleted
	}
	err := al.UpdateMediaListEntry(ctx, id, episode, status)
	if err != nil {
		// Reconstruct the GraphQL payload for offline queuing.
		variables := map[string]interface{}{
			"mediaId":  id,
			"progress": episode,
			"status":   status,
		}

		body := map[string]interface{}{
			"query":     saveMediaListEntryMutation,
			"variables": variables,
		}

		jsonBody, _ := json.Marshal(body)

		// Intercept the state mutation and commit it to the offline persistence queue for
		// staggered background reconciliation once network availability is restored.
		_ = sync.QueueFailure("anilist", id, "UpdateEpisodeProgress", string(jsonBody))
		return fmt.Errorf("sync_queued")
	}

	return nil
}

// CheckAuth preemptively checks for the AniList token.
func (c *Client) CheckAuth(ctx context.Context) error {
	_, err := al.GetToken()
	if err != nil {
		return fmt.Errorf("AniList authentication missing. Please run 'anisan anilist auth'")
	}
	return nil
}
