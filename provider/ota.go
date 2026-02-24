// Package provider manages built-in and custom scraping providers.
package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/anisan-cli/anisan/log"
	"github.com/anisan-cli/anisan/where"
	tea "github.com/charmbracelet/bubbletea"
)

const RepoRawURL = "https://raw.githubusercontent.com/santosh-k22/anisan-cli/main/config/sources/"

// ScraperUpdatedMsg is dispatched to the Bubbletea event loop when OTA updates complete successfully.
type ScraperUpdatedMsg struct{}

// UpdateScrapers spawns a non-blocking background goroutine to fetch OTA script logic updates.
// It uses SHA-256 hash checks to avoid redundant disk writes.
func UpdateScrapers() tea.Cmd {
	return func() tea.Msg {
		filesToUpdate := []string{"common.lua", "allanime.lua"}
		updated := false

		// Timeout to prevent the goroutine from leaking during DNS failures
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client := &http.Client{}

		for _, file := range filesToUpdate {
			if updateSingleFile(ctx, client, file) {
				updated = true
			}
		}

		if updated {
			log.Info("OTA Scraper Updates completed successfully. Emitting reload event.")
			return ScraperUpdatedMsg{}
		}

		log.Info("OTA Scraper Check completed. No updates available.")
		return nil
	}
}

func updateSingleFile(ctx context.Context, client *http.Client, filename string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, RepoRawURL+filename, nil)
	if err != nil {
		log.Warnf("Failed to create OTA request for %s: %v", filename, err)
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Warnf("OTA network failure for %s: %v", filename, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warnf("OTA returned non-200 for %s: %d", filename, resp.StatusCode)
		return false
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	remoteHashRaw := sha256.Sum256(bodyBytes)
	remoteHash := hex.EncodeToString(remoteHashRaw[:])

	localPath := filepath.Join(where.Sources(), filename)
	localBytes, err := os.ReadFile(localPath)

	if err == nil {
		localHashRaw := sha256.Sum256(localBytes)
		localHash := hex.EncodeToString(localHashRaw[:])
		if localHash == remoteHash {
			// Hashes match, exit immediately.
			return false
		}
	}

	// Hashes differ or local file missing, perform update.
	tmpPath := localPath + ".tmp"
	err = os.WriteFile(tmpPath, bodyBytes, 0644)
	if err != nil {
		log.Warnf("OTA failed to write tmp file for %s: %v", filename, err)
		return false
	}

	// Atomic swap prevents corrupt state
	err = os.Rename(tmpPath, localPath)
	if err != nil {
		_ = os.Remove(tmpPath) // Cleanup on failure
		log.Warnf("OTA failed atomic swap for %s: %v", filename, err)
		return false
	}

	log.Infof("OTA updated scraper script: %s", filename)
	return true
}
