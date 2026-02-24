package anilist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/anisan-cli/anisan/anilist"
	"github.com/anisan-cli/anisan/auth"
	"github.com/anisan-cli/anisan/internal/sync"
	"github.com/anisan-cli/anisan/log"
	"github.com/anisan-cli/anisan/source"
)

var markWatchedQuery = `
mutation ($ID: Int, $progress: Int) {
	SaveMediaListEntry (mediaId: $ID, progress: $progress, status: CURRENT) {
		ID
	}
}
`

func (a *Anilist) MarkWatched(episode *source.Episode) error {
	if a.token == "" {
		token, err := auth.GetToken()
		if err != nil {
			log.Error(err)
			return err
		}
		a.token = token
	}

	anime, err := anilist.FindClosest(episode.Anime.Name)
	if err != nil {
		log.Error(err)
		return err
	}

	// prepare body
	body := map[string]interface{}{
		"query": markWatchedQuery,
		"variables": map[string]interface{}{
			"ID":       anime.ID,
			"progress": episode.Index,
		},
	}

	// parse body to json
	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Error(err)
		return err
	}

	// make request
	req, err := http.NewRequest(
		http.MethodPost,
		"https://graphql.anilist.co",
		bytes.NewBuffer(jsonBody),
	)

	if err != nil {
		log.Error(err)
		return err
	}

	// set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Accept", "application/json")

	// send request
	log.Info("Sending request to Anilist: " + string(jsonBody))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Warnf("Network failure, committing to offline sync queue: %v", err)
		if qErr := sync.QueueFailure(anime.ID, "MarkWatched", string(jsonBody)); qErr == nil {
			return fmt.Errorf("sync_queued") // Sentinel error string intercepted by the TUI for asynchronous notification.
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warnf("Request failed with status %d, committing to offline sync queue", resp.StatusCode)
		if qErr := sync.QueueFailure(anime.ID, "MarkWatched", string(jsonBody)); qErr == nil {
			return fmt.Errorf("sync_queued")
		}
		return fmt.Errorf("invalid response code %d", resp.StatusCode)
	}

	// decode response
	var response struct {
		Data struct {
			SaveMediaListEntry struct {
				ID int `json:"ID"`
			} `json:"SaveMediaListEntry"`
		} `json:"data"`
	}

	return json.NewDecoder(resp.Body).Decode(&response)
}
