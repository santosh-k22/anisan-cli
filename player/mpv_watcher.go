package player

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/anisan-cli/anisan/internal/tracker"
)

// ErrSyncQueued is returned by the tracker when the mutation is deferred to a persistence queue.
// It is not a failure — the update will be applied asynchronously.
var ErrSyncQueued = errors.New("sync_queued")

// mpvEvent is the minimal typed structure for MPV IPC events.
// Using a typed struct avoids per-event heap allocations from map[string]interface{}.
type mpvEvent struct {
	Event  string  `json:"event"`
	Name   string  `json:"name"`
	Data   float64 `json:"data"`
	Reason string  `json:"reason"`
}

// MPVWatcher monitors the MPV player's state via IPC and triggers tracker sync on playback events.
type MPVWatcher struct {
	socketPath    string
	mediaTracker  tracker.MediaTracker
	updateTrigger float64
	mediaID       int
	episodeNum    int
	totalEps      int
	syncGuard     *atomic.Bool
}

// NewMPVWatcher initializes a tracker-aware watcher for a specific media entry.
func NewMPVWatcher(socket string, t tracker.MediaTracker, mediaID, ep, totalEps int, guard *atomic.Bool) *MPVWatcher {
	return &MPVWatcher{
		socketPath:    socket,
		mediaTracker:  t,
		updateTrigger: 80.0,
		mediaID:       mediaID,
		episodeNum:    ep,
		totalEps:      totalEps,
		syncGuard:     guard,
	}
}

func (w *MPVWatcher) Poll(ctx context.Context) error {
	conn, err := net.DialTimeout("unix", w.socketPath, 2*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to mpv socket: %w", err)
	}
	defer conn.Close()

	// Register property observation for 'percent-pos'. Standard events like 'end-file'
	// are delivered automatically to all connected IPC clients without explicit observation.
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

		var event mpvEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			continue
		}

		// Case 1: Standard property observation for 'percent-pos' (80% threshold).
		if event.Event == "property-change" && event.Name == "percent-pos" {
			if !updateFired && event.Data >= w.updateTrigger {
				if err := w.triggerUpdate(ctx); err == nil {
					updateFired = true
				}
			}
		}

		// Case 2: Native 'end-file' event for deterministic completion (EOF heist).
		if event.Event == "end-file" {
			if !updateFired && event.Reason == "eof" {
				_ = w.triggerUpdate(ctx)
			}
			return nil
		}
	}
}

// triggerUpdate executes the tracker synchronization if the sync guard allows.
func (w *MPVWatcher) triggerUpdate(ctx context.Context) error {
	if w.syncGuard != nil {
		if !w.syncGuard.CompareAndSwap(false, true) {
			return nil
		}
	}

	if err := w.mediaTracker.UpdateEpisodeProgress(ctx, w.mediaID, w.episodeNum, w.totalEps); err != nil {
		if !errors.Is(err, ErrSyncQueued) {
			if w.syncGuard != nil {
				w.syncGuard.Store(false)
			}
			return err
		}
	}
	return nil
}
