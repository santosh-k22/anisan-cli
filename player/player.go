// Package player defines a unified abstraction layer for media playback engines.
// The architecture supports multiple backends, with the primary implementation targeting 'mpv' via its JSON-IPC interface.
package player

// Player encapsulates the required capabilities for a media playback backend.
type Player interface {
	// Play starts playback of the given URL with the specified title.
	// If a player instance is already running, it loads the new file into it.
	Play(url string, title string, headers map[string]string) error

	// TogglePause inverts the current playback suspension state.
	TogglePause() error

	// GetTimePos retrieves the current absolute playback position in seconds.
	GetTimePos() (float64, error)

	// GetDuration retrieves the total temporal length of the active media file in seconds.
	GetDuration() (float64, error)

	// GetPercentWatched calculates the relative playback completion percentage (0-100).
	GetPercentWatched() (float64, error)

	// GetPausedStatus retrieves the current suspension state of the playback engine.
	GetPausedStatus() (bool, error)

	// HasActivePlayback verifies if the player has a media file currently initialized and active.
	HasActivePlayback() (bool, error)

	// Seek transitions the playback position to a specific absolute timestamp in seconds.
	Seek(seconds float64) error

	// IsRunning validates the liveness of the underlying playback process or handler.
	IsRunning() bool

	// Close terminates the playback engine and releases all associated system resources.
	Close() error

	// Socket retrieves the identifier for the Inter-Process Communication (IPC) channel.
	Socket() string

	// StartIPCTicker initializes a background synchronization task to poll playback metrics.
	// It executes the provided callback at regular intervals (typically 1Hz) with current state data.
	StartIPCTicker(callback func(timePos int, duration int))

	// StopIPCTicker terminates the background synchronization task.
	StopIPCTicker()
	// Wait returns a channel that is closed when the playback session terminates.
	Wait() <-chan struct{}
}
