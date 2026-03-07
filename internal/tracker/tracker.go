package tracker

import "context"

// MediaTracker defines the unified interface for media synchronization.
// By accepting a context, the interface ensures robust lifecycle management,
// allowing for explicit timeouts and cancellation during CLI termination.
type MediaTracker interface {
	// UpdateEpisodeProgress synchronizes the media progress state with the remote tracking backend.
	UpdateEpisodeProgress(ctx context.Context, id int, episode int) error
}
