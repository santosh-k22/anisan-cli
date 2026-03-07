package mal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/anisan-cli/anisan/internal/sync"
	"github.com/zalando/go-keyring"
)

const (
	malAPIStatusURL = "https://api.myanimelist.net/v2/anime/%d/my_list_status"
	serviceName     = "anisan"
	accountName     = "mal-token"
)

// Client implements the MediaTracker interface utilizing the MyAnimeList v2 API.
// It encapsulates a hardened HTTP transport to prevent resource exhaustion.
type Client struct {
	httpClient *http.Client
}

// NewClient initializes a MAL client with a strictly configured HTTP transport.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			// Explicit timeout to prevent hanging tracker routines during network degradation.
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     30 * time.Second,
				// Transparent compression is enabled; however, we rely on the
				// standard library's automatic decompression handling.
				DisableCompression: false,
			},
		},
	}
}

// UpdateEpisodeProgress executes an authenticated PATCH request to synchronize
// the media progress state for the specified entry.
func (c *Client) UpdateEpisodeProgress(ctx context.Context, id int, episode int, totalEpisodes int) error {
	// Credentials are retrieved securely from the operating system's keyring service,
	// eliminating the risks associated with plaintext token storage.
	token, err := keyring.Get(serviceName, accountName)
	if err != nil {
		return fmt.Errorf("failed to retrieve MAL token from secure keyring: %w", err)
	}

	targetURL := fmt.Sprintf(malAPIStatusURL, id)

	// Construct the x-www-form-urlencoded mutation payload as specified in
	// the MyAnimeList v2 API documentation.
	formData := url.Values{}
	formData.Set("num_watched_episodes", strconv.Itoa(episode))

	// Dynamically handle completion state if metadata provides a total episode count.
	if totalEpisodes > 0 && episode >= totalEpisodes {
		formData.Set("status", "completed")
	} else {
		formData.Set("status", "watching")
	}

	encodedData := formData.Encode()

	// Formulate the HTTP request using the provided context to respect timeouts
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, targetURL, strings.NewReader(encodedData))
	if err != nil {
		return fmt.Errorf("failed to construct MAL HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Execute the HTTP transaction via the hardened internal transport.
	resp, err := c.httpClient.Do(req)
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
		_, _ = io.ReadAll(resp.Body)
		_ = sync.QueueFailure("mal", id, "UpdateEpisodeProgress", encodedData)
		return fmt.Errorf("sync_queued")
	}

	// Drain the response body on success to ensure Keep-Alive functions correctly
	_, _ = io.Copy(io.Discard, resp.Body)

	return nil
}

// CheckAuth preemptively checks for the MAL token in the secure keyring.
func (c *Client) CheckAuth(ctx context.Context) error {
	_, err := keyring.Get(serviceName, accountName)
	if err != nil {
		return fmt.Errorf("MAL authentication missing. Please run 'anisan mal auth'")
	}
	return nil
}
