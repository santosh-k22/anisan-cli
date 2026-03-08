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
	Synopsis      string
	Genres        []JikanGenre
	Images        JikanImages
}

type JikanGenre struct {
	Name string `json:"name"`
}

type JikanImages struct {
	Jpg struct {
		ImageURL      string `json:"image_url"`
		LargeImageURL string `json:"large_image_url"`
	} `json:"jpg"`
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
			TitleEnglish string       `json:"title_english"`
			Year         int          `json:"year"`
			Score        float64      `json:"score"`
			Status       string       `json:"status"`
			Episodes     int          `json:"episodes"`
			Synopsis     string       `json:"synopsis"`
			Genres       []JikanGenre `json:"genres"`
			Images       JikanImages  `json:"images"`
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
		Synopsis:      stencil.Data.Synopsis,
		Genres:        stencil.Data.Genres,
		Images:        stencil.Data.Images,
	}, nil
}

// SearchByName queries the Jikan v4 search endpoint by anime title.
// This is a completely unauthenticated call — no MAL token is required.
// Returns the best-matching AnimeMetadata or an error if nothing was found.
func SearchByName(ctx context.Context, name string) (*AnimeMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.jikan.moe/v4/anime", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jikan search request: %w", err)
	}

	q := req.URL.Query()
	q.Set("q", name)
	q.Set("limit", "5")
	q.Set("sfw", "false")
	req.URL.RawQuery = q.Encode()

	resp, err := jikanClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Jikan search network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Jikan search returned HTTP %d", resp.StatusCode)
	}

	var stencil struct {
		Data []struct {
			MalID        int          `json:"mal_id"`
			TitleEnglish string       `json:"title_english"`
			Title        string       `json:"title"`
			Year         int          `json:"year"`
			Score        float64      `json:"score"`
			Status       string       `json:"status"`
			Episodes     int          `json:"episodes"`
			Synopsis     string       `json:"synopsis"`
			Genres       []JikanGenre `json:"genres"`
			Images       JikanImages  `json:"images"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&stencil); err != nil {
		return nil, fmt.Errorf("failed to decode Jikan search payload: %w", err)
	}

	if len(stencil.Data) == 0 {
		return nil, fmt.Errorf("no Jikan results for %q", name)
	}

	d := stencil.Data[0]
	title := d.TitleEnglish
	if title == "" {
		title = d.Title
	}

	return &AnimeMetadata{
		EnglishTitle:  title,
		Year:          d.Year,
		Score:         d.Score,
		Status:        d.Status,
		TotalEpisodes: d.Episodes,
		Synopsis:      d.Synopsis,
		Genres:        d.Genres,
		Images:        d.Images,
	}, nil
}
