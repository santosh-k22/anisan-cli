// Package aniskip provides a client for the AniSkip API, enabling automated retrieval of opening and ending skip timestamps.

package aniskip

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/anisan-cli/anisan/log"
)

const baseURL = "https://api.aniskip.com/v1/skip-times"

// SkipTimes encapsulates the temporal intervals for opening and ending sequences.
type SkipTimes struct {
	Opening  Interval `json:"opening"`
	Ending   Interval `json:"ending"`
	HasIntro bool     `json:"has_intro"`
	HasOutro bool     `json:"has_outro"`
}

// Interval represents a continuous temporal range defined in seconds.
type Interval struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// apiResponse defines the internal structural mapping for AniSkip API responses.
type apiResponse struct {
	Found   bool `json:"found"`
	Results []struct {
		Interval struct {
			StartTime float64 `json:"start_time"`
			EndTime   float64 `json:"end_time"`
		} `json:"interval"`
		SkipType string `json:"skip_type"`
	} `json:"results"`
}

// GetSkipTimes retrieves the skip intervals for a specific media entry and episode number from the AniSkip service.
// Returns nil (not an error) if no skip times are available.
func GetSkipTimes(malID int, episode int) (*SkipTimes, error) {
	url := fmt.Sprintf("%s/%d/%d?types=op&types=ed", baseURL, malID, episode)

	resp, err := http.Get(url)
	if err != nil {
		log.Warnf("aniskip API request failed: %v", err)
		return nil, nil // Graceful degradation
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warnf("aniskip API returned status %d", resp.StatusCode)
		// Recover gracefully: Maintain operation without skip interval data.
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read aniskip response: %w", err)
	}

	var data apiResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parse aniskip response: %w", err)
	}

	if !data.Found || len(data.Results) == 0 {
		// Null result: No skip segments registered for the specified episode.
		return nil, nil
	}

	times := &SkipTimes{}

	for _, result := range data.Results {
		switch result.SkipType {
		case "op":
			times.Opening = Interval{
				Start: result.Interval.StartTime,
				End:   result.Interval.EndTime,
			}
			times.HasIntro = true
		case "ed":
			times.Ending = Interval{
				Start: result.Interval.StartTime,
				End:   result.Interval.EndTime,
			}
			times.HasOutro = true
		}
	}

	return times, nil
}
