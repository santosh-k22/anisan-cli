// Package mal provides a client for the MyAnimeList REST API.
package mal

import (
	"fmt"
	"io"
	"net/http"
)

const (
	apiEndpoint = "https://api.myanimelist.net/v2"
)

// authenticatedRequest performs an HTTP request with the OAuth token.
func authenticatedRequest(method, urlStr string, body io.Reader) (*http.Response, error) {
	token, err := LoadToken()
	if err != nil {
		return nil, fmt.Errorf("mal auth required: %w", err)
	}

	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mal api request: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return resp, fmt.Errorf("unauthorized (token might be expired, run `anisan mal auth`)")
	}

	return resp, nil
}
