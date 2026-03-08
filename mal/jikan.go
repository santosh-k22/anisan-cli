package mal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/anisan-cli/anisan/network"
)

var (
	jikanMu       sync.Mutex
	jikanLastCall time.Time
)

// enforceRateLimit synchronizes calls to ensure compliance with Jikan v4's 3 requests/second throughput constraint.
func enforceRateLimit() {
	jikanMu.Lock()
	defer jikanMu.Unlock()

	elapsed := time.Since(jikanLastCall)
	if elapsed < 334*time.Millisecond {
		time.Sleep((334 * time.Millisecond) - elapsed)
	}
	jikanLastCall = time.Now()
}

// GetByID fetches bare-minimum MAL metadata directly via Jikan for manual sideloading.
func GetByID(id int) (*Anime, error) {
	enforceRateLimit()

	endpoint := fmt.Sprintf("https://api.jikan.moe/v4/anime/%d", id)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := network.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jikan api returned status: %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			MalID    int     `json:"mal_id"`
			Title    string  `json:"title"`
			Episodes int     `json:"episodes"`
			Score    float64 `json:"score"`
			Status   string  `json:"status"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &Anime{
		ID:          result.Data.MalID,
		Title:       result.Data.Title,
		NumEpisodes: result.Data.Episodes,
		Mean:        result.Data.Score,
		Status:      result.Data.Status,
	}, nil
}
