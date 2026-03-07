package jikan

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// AnimeMetadata encapsulates the high-level media details retrieved via Jikan.
// It is optimized for flat consumption within the TUI model projection.
type AnimeMetadata struct {
	EnglishTitle  string
	Year          int
	Score         float64
	Status        string
	TotalEpisodes int
}

var jikanClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        5,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     10 * time.Second,
	},
}

// FetchMetadata retrieves extended media details via the Jikan v4 REST service.
// It employs a type-safe anonymous stencil to perform subtractive JSON decoding,
// strictly filtering the upstream payload to minimize memory instantiation.
func FetchMetadata(ctx context.Context, id int) (*AnimeMetadata, error) {
	url := fmt.Sprintf("https://api.jikan.moe/v4/anime/%d", id)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to formulate Jikan metadata request: %w", err)
	}

	resp, err := jikanClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Jikan API network connectivity error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Jikan API responded with HTTP %d error status", resp.StatusCode)
	}

	// Subtractive Decoding: The internal struct acts as a type-safe filter.
	// The standard library's json.Decoder will ignore all fields not explicitly
	// defined in the stencil, minimizing memory churn for large payloads.
	var stencil struct {
		Data struct {
			TitleEnglish string  `json:"title_english"`
			Year         int     `json:"year"`
			Score        float64 `json:"score"`
			Status       string  `json:"status"`
			Episodes     int     `json:"episodes"`
		} `json:"data"`
	}

	// Decode directly from the network stream buffer
	if err := json.NewDecoder(resp.Body).Decode(&stencil); err != nil {
		return nil, fmt.Errorf("failed to decode Jikan JSON payload: %w", err)
	}

	return &AnimeMetadata{
		EnglishTitle:  stencil.Data.TitleEnglish,
		Year:          stencil.Data.Year,
		Score:         stencil.Data.Score,
		Status:        stencil.Data.Status,
		TotalEpisodes: stencil.Data.Episodes,
	}, nil
}
