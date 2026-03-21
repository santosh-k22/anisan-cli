package mal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/anisan-cli/anisan/internal/sync"
	"github.com/anisan-cli/anisan/mal"
)

const (
	malAPIStatusURL = "https://api.myanimelist.net/v2/anime/%d/my_list_status"
	serviceName     = "anisan"
	accountName     = "mal-token"
)

// Client implements the MediaTracker interface utilizing the MyAnimeList v2 API.
// It relies on the globally tuned HTTP transport to prevent resource exhaustion.
type Client struct{}

// NewClient initializes a MAL client.
func NewClient() *Client {
	return &Client{}
}

// UpdateEpisodeProgress executes an authenticated PATCH request to synchronize
// the media progress state for the specified entry.
func (c *Client) UpdateEpisodeProgress(ctx context.Context, id int, episode int, totalEpisodes int) error {
	// Credentials are retrieved utilizing the structural loader which unmarshals the JSON representation correctly.
	_, err := mal.LoadToken()
	if err != nil {
		return fmt.Errorf("failed to retrieve MAL token: %w", err)
	}

	targetURL := fmt.Sprintf(malAPIStatusURL, id)

	formData := url.Values{}
	formData.Set("num_watched_episodes", strconv.Itoa(episode))

	// Dynamically handle completion state if metadata provides a total episode count.
	// As per forensic audit: explicitly transition to 'completed' when current episode matches total.
	if totalEpisodes > 0 && episode >= totalEpisodes {
		formData.Set("status", "completed")
	} else {
		formData.Set("status", "watching")
	}

	encodedData := formData.Encode()

	// Execute the HTTP transaction via the structural automatic-refresh wrapper.
	resp, err := mal.AuthenticatedRequest(http.MethodPatch, targetURL, encodedData)
	if err != nil {
		// Network transport failure (e.g., DNS resolution, connection refused, dial timeout).
		// Intercept the state mutation and commit it to the offline persistence queue for
		// staggered background reconciliation once network availability is restored.
		_ = sync.QueueFailure("mal", id, "UpdateEpisodeProgress", encodedData)
		return fmt.Errorf("sync_queued")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// The API actively rejected the payload (e.g., 502 Bad Gateway, 429 Too Many Requests).
		// Drain the body memory to safely release the TCP socket, and append to the offline queue.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = sync.QueueFailure("mal", id, "UpdateEpisodeProgress", encodedData)
		return fmt.Errorf("sync_queued")
	}

	// Drain the response body on success to ensure Keep-Alive functions correctly
	_, _ = io.Copy(io.Discard, resp.Body)

	return nil
}

// CheckAuth preemptively checks for the MAL token in the secure keyring.
func (c *Client) CheckAuth(ctx context.Context) error {
	_, err := mal.LoadToken()
	if err != nil {
		return fmt.Errorf("MAL authentication missing. Please run 'anisan mal auth'")
	}
	return nil
}
