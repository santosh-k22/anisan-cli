package tracker

import "context"

// MediaStatus represents the unified synchronization state.
type MediaStatus string

const (
	StatusWatching  MediaStatus = "watching"
	StatusCompleted MediaStatus = "completed"
)

// MediaTracker defines the unified interface for media synchronization.
// By accepting a context, the interface ensures robust lifecycle management,
// allowing for explicit timeouts and cancellation during CLI termination.
type MediaTracker interface {
	// CheckAuth verifies if the necessary credentials exist before initiating sync.
	CheckAuth(ctx context.Context) error

	// UpdateEpisodeProgress synchronizes state. totalEpisodes allows for dynamic StatusCompleted triggers.
	UpdateEpisodeProgress(ctx context.Context, id int, episode int, totalEpisodes int) error
}
