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
	"github.com/anisan-cli/anisan/query"
	"github.com/samber/lo"
)

// searchByNameResponse defines the anticipated JSON response structure for anime-by-name searches.
type searchByNameResponse struct {
	Data struct {
		Page struct {
			Media []*Anime `json:"media"`
		} `json:"page"`
	} `json:"data"`
}

// searchByIDResponse defines the anticipated JSON response structure for anime-by-id lookups.
type searchByIDResponse struct {
	Data struct {
		Media *Anime `json:"media"`
	} `json:"data"`
}

// GetByID returns the anime with the given id.
// If the anime is not found, it returns nil.
func GetByID(id int) (*Anime, error) {
	if anime := idCacher.Get(id); anime.IsPresent() {
		return anime.MustGet(), nil
	}

	// Prepare request body for GraphQL query.
	log.Infof("Searching anilist for anime with id: %d", id)
	body := map[string]interface{}{
		"query": searchByIDQuery,
		"variables": map[string]interface{}{
			"id": id,
		},
	}

	// Marshal the request body to JSON.
	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Create and send the HTTP request to the Anilist API.
	log.Info("Sending request to Anilist")
	req, err := http.NewRequest(http.MethodPost, "https://graphql.anilist.co", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := network.Client.Do(req)

	if err != nil {
		log.Error(err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Error("Anilist returned status code " + strconv.Itoa(resp.StatusCode))
		return nil, fmt.Errorf("invalid response code %d", resp.StatusCode)
	}

	// Decode the JSON response into the response structure.
	var response searchByIDResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Error(err)
		return nil, err
	}

	anime := response.Data.Media
	log.Infof("Got response from Anilist, found anime with id %d", anime.ID)
	_ = idCacher.Set(id, anime)
	return anime, nil
}

// SearchByName returns a list of animes that match the given name.
func SearchByName(name string) ([]*Anime, error) {
	name = normalizedName(name)
	_ = query.Remember(name, 1)

	if _, failed := failCacher.Get(name).Get(); failed {
		return nil, fmt.Errorf("failed to search for %s", name)
	}

	if ids, ok := searchCacher.Get(name).Get(); ok {
		animes := lo.FilterMap(ids, func(item, _ int) (*Anime, bool) {
			return idCacher.Get(item).Get()
		})

		if len(animes) == 0 {
			_ = searchCacher.Delete(name)
			return SearchByName(name)
		}

		return animes, nil
	}

	// Prepare the request body for the GraphQL query.
	log.Infof("Searching anilist for anime %s", name)
	body := map[string]any{
		"query": searchByNameQuery,
		"variables": map[string]any{
			"query": name,
		},
	}

	// Marshal the request body to JSON.
	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Dispatch the GraphQL request to the Anilist API.
	log.Info("Sending request to Anilist")
	req, err := http.NewRequest(http.MethodPost, "https://graphql.anilist.co", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := network.Client.Do(req)

	if err != nil {
		log.Error(err)
		_ = failCacher.Set(name, true)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Error("Anilist returned status code " + strconv.Itoa(resp.StatusCode))
		_ = failCacher.Set(name, true)
		return nil, fmt.Errorf("invalid response code %d", resp.StatusCode)
	}

	// Decode the JSON response into the result structure.
	var response searchByNameResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Error(err)
		return nil, err
	}

	animes := response.Data.Page.Media
	log.Infof("Got response from Anilist, found %d results", len(animes))
	ids := make([]int, len(animes))
	for i, anime := range animes {
		ids[i] = anime.ID
		_ = idCacher.Set(anime.ID, anime)
	}
	_ = searchCacher.Set(name, ids)
	return animes, nil
}
