// Package anilist provides a client for the Anilist GraphQL API.
package anilist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/anisan-cli/anisan/log"
	"github.com/anisan-cli/anisan/network"
)

// saveMediaListEntryMutation is the GraphQL mutation to update a user's list entry.
var saveMediaListEntryMutation = `
mutation ($mediaId: Int, $progress: Int, $status: MediaListStatus) {
    SaveMediaListEntry (mediaId: $mediaId, progress: $progress, status: $status) {
        id
        progress
        status
    }
}
`

// MediaListStatus represents the status of a media in the user's list.
type MediaListStatus string

const (
	MediaListStatusCurrent   MediaListStatus = "CURRENT"
	MediaListStatusPlanning  MediaListStatus = "PLANNING"
	MediaListStatusCompleted MediaListStatus = "COMPLETED"
	MediaListStatusDropped   MediaListStatus = "DROPPED"
	MediaListStatusPaused    MediaListStatus = "PAUSED"
	MediaListStatusRepeating MediaListStatus = "REPEATING"
)

// UpdateMediaListEntry updates the progress and status of an anime in the user's list.
func UpdateMediaListEntry(mediaId int, progress int, status MediaListStatus) error {
	token, err := GetToken()
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	variables := map[string]interface{}{
		"mediaId":  mediaId,
		"progress": progress,
	}
	if status != "" {
		variables["status"] = status
	}

	body := map[string]interface{}{
		"query":     saveMediaListEntryMutation,
		"variables": variables,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "https://graphql.anilist.co", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	log.Infof("Updating Anilist: MediaID=%d, Progress=%d, Status=%s", mediaId, progress, status)

	resp, err := network.Client.Do(req)
	if err != nil {
		log.Error(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error("Anilist update failed with status code " + strconv.Itoa(resp.StatusCode))

		// Try to read error body
		var errData map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errData)
		log.Errorf("Anilist Error: %+v", errData)

		return fmt.Errorf("anilist update failed: %d", resp.StatusCode)
	}

	log.Info("Anilist update successful")
	return nil
}
