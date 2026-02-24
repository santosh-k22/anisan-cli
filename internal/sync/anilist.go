// Package sync implements asynchronous background synchronization and offline queuing for external service tracking.
package sync

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/anisan-cli/anisan/anilist"
)

// SyncMutation encapsulates a single tracking operation for deferred synchronization.
type SyncMutation struct {
	Timestamp int64  `json:"timestamp"`
	MediaID   int    `json:"media_id"`
	Action    string `json:"action"`
	Payload   string `json:"payload"`
}

func getSyncFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	dir := filepath.Join(home, ".config", "anisan")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "failed_syncs.json")
}

// QueueFailure persists a failed tracking operation to a local JSON-log for deferred reconciliation.
func QueueFailure(mediaID int, action, payload string) error {
	f, err := os.OpenFile(getSyncFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	mutation := SyncMutation{
		Timestamp: time.Now().Unix(),
		MediaID:   mediaID,
		Action:    action,
		Payload:   payload,
	}

	// Stream JSON directly to disk buffer
	encoder := json.NewEncoder(f)
	return encoder.Encode(mutation)
}

// ReconcileFailures initializes an asynchronous background process to synchronize previously failed tracking attempts.
func ReconcileFailures() {
	go func() {
		path := getSyncFile()
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			return
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return
		}

		var mutations []SyncMutation
		decoder := json.NewDecoder(bytes.NewReader(content))
		for decoder.More() {
			var m SyncMutation
			if err := decoder.Decode(&m); err == nil {
				mutations = append(mutations, m)
			}
		}

		if len(mutations) == 0 {
			return
		}

		client := &http.Client{Timeout: 10 * time.Second}
		successCount := 0

		for i, m := range mutations {
			// Apply incremental delay with randomized jitter to manage request throttling.
			backoff := time.Duration((1<<i)*100)*time.Millisecond + time.Duration(rand.Intn(100))*time.Millisecond
			time.Sleep(backoff)

			req, err := http.NewRequest(http.MethodPost, "https://graphql.anilist.co", bytes.NewBufferString(m.Payload))
			if err != nil {
				continue
			}

			// Use the stored authentication token if available.
			token, err := anilist.GetToken()
			if err == nil && token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")

			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					successCount++
				}
			}
		}

		// Atomic state update: Truncate the failure log if all operations successfully synchronized.
		if successCount == len(mutations) {
			_ = os.Truncate(path, 0)
		}
	}()
}
