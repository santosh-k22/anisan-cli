package player

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/anisan-cli/anisan/internal/tracker"
)

// MPVWatcher manages an IPC bridge to an external mpv instance to monitor
// and synchronize playback state with the active media trackers.
type MPVWatcher struct {
	socketPath    string
	mediaTracker  tracker.MediaTracker
	updateTrigger float64
	mediaID       int
	episodeNum    int
	totalEps      int
}

// NewMPVWatcher initializes a tracker-aware watcher for a specific media entry.
func NewMPVWatcher(socket string, t tracker.MediaTracker, mediaID, ep, totalEps int) *MPVWatcher {
	return &MPVWatcher{
		socketPath:    socket,
		mediaTracker:  t,
		updateTrigger: 80.0,
		mediaID:       mediaID,
		episodeNum:    ep,
		totalEps:      totalEps,
	}
}

// Poll establishes the IPC link and initiates the event processing loop.
// It implements a zero-allocation read cycle directly from the socket to
// minimize intermediary buffer instantiation.
func (w *MPVWatcher) Poll(ctx context.Context) error {
	conn, err := net.DialTimeout("unix", w.socketPath, 2*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to mpv socket: %w", err)
	}
	defer conn.Close()

	// Register property observation for 'percent-pos'. This enables real-time
	// state updates via the IPC event stream for asynchronous progress detection.
	observeCmd := `{"command": ["observe_property", 1, "percent-pos"]}` + "\n"
	if _, err := conn.Write([]byte(observeCmd)); err != nil {
		return fmt.Errorf("failed to write observe command: %w", err)
	}

	decoder := json.NewDecoder(bufio.NewReader(conn))
	updateFired := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Stencil Struct: Optimized for minimal heap escape during unmarshaling.
		var event struct {
			Event string  `json:"event"`
			Name  string  `json:"name"`
			Data  float64 `json:"data"`
		}

		if err := decoder.Decode(&event); err != nil {
			// io.EOF or io.ErrUnexpectedEOF signifies that the remote mpv instance
			// has terminated the IPC socket. This is handled as a graceful exit.
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			// For any other transient JSON parse error, skip this line and continue.
			continue
		}

		if event.Event == "property-change" && event.Name == "percent-pos" {
			if !updateFired && event.Data >= w.updateTrigger {
				if err := w.mediaTracker.UpdateEpisodeProgress(ctx, w.mediaID, w.episodeNum, w.totalEps); err != nil {
					// "sync_queued" is a non-fatal sentinel indicating that the mutation was
					// offloaded to the persistence queue; it does not signify a failure of the read loop.
					if err.Error() != "sync_queued" {
						return fmt.Errorf("tracker update failed: %w", err)
					}
				}
				updateFired = true
			}
		}
	}
}
