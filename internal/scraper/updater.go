// Package scraper provides high-level coordination and execution for virtualized Lua-based scraping modules.
package scraper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ScraperUpdatedMsg is sent when a script is successfully updated.
type ScraperUpdatedMsg struct{}

// UpdateScraper spawns an asynchronous Goroutine to fetch, hash, and atomically swap the Lua script
// without blocking the main Bubbletea execution loop.
func UpdateScraper(remoteURL, localPath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
		if err != nil {
			return nil
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil
		}

		remoteHashRaw := sha256.Sum256(bodyBytes)
		remoteHash := hex.EncodeToString(remoteHashRaw[:])

		localBytes, err := os.ReadFile(localPath)
		if err == nil {
			localHashRaw := sha256.Sum256(localBytes)
			localHash := hex.EncodeToString(localHashRaw[:])
			if localHash == remoteHash {
				return nil
			}
		}

		tmpPath := localPath + ".tmp"
		if err := os.WriteFile(tmpPath, bodyBytes, 0644); err != nil {
			return nil
		}

		// renameat2 system call equivalent: atomic swap prevents corrupt AST state
		if err := os.Rename(tmpPath, localPath); err != nil {
			_ = os.Remove(tmpPath)
			return nil
		}

		return ScraperUpdatedMsg{}
	}
}
