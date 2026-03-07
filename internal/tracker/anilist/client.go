package anilist

import (
	"context"

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
func (c *Client) UpdateEpisodeProgress(ctx context.Context, id int, episode int) error {
	// The internal Anilist implementation currently operates synchronously;
	// this wrapper ensures interface compliance for the dual-tracker registry.
	return al.UpdateMediaListEntry(id, episode, al.MediaListStatusCurrent)
}
