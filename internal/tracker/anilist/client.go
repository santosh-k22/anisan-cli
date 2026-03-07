package anilist

import (
	"context"
	"fmt"

	al "github.com/anisan-cli/anisan/anilist"
)

// Client implements the MediaTracker interface by wrapping the legacy Anilist integration logic.
type Client struct{}

// NewClient returns an initialized instance of the Anilist tracker client.
func NewClient() *Client {
	return &Client{}
}

// UpdateEpisodeProgress synchronizes the media progress state with the Anilist API.
// It maps the canonical interface call to the internal mutation engine.
func (c *Client) UpdateEpisodeProgress(ctx context.Context, id int, episode int, totalEpisodes int) error {
	// The internal Anilist implementation currently operates synchronously;
	// this wrapper ensures interface compliance for the dual-tracker registry.
	status := al.MediaListStatusCurrent
	if totalEpisodes > 0 && episode >= totalEpisodes {
		status = al.MediaListStatusCompleted
	}
	return al.UpdateMediaListEntry(id, episode, status)
}

// CheckAuth preemptively checks for the AniList token.
func (c *Client) CheckAuth(ctx context.Context) error {
	_, err := al.GetToken()
	if err != nil {
		return fmt.Errorf("AniList authentication missing. Please run 'anisan anilist auth'")
	}
	return nil
}
