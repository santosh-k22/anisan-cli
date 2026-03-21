// Package mal provides a client for the MyAnimeList REST API.
package mal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/anisan-cli/anisan/network"
	"github.com/spf13/viper"
)

const (
	apiEndpoint = "https://api.myanimelist.net/v2"
)

// AuthenticatedRequest executes an HTTP request using an OAuth2 Bearer token, automatically refreshing the token on 401 Unauthorized responses.
func AuthenticatedRequest(method, urlStr string, body string) (*http.Response, error) {
	token, err := LoadToken()
	if err != nil {
		return nil, fmt.Errorf("mal auth required: %w", err)
	}

	// Helper to create and authorize requests
	buildReq := func(t *Token) (*http.Request, error) {
		req, err := http.NewRequest(method, urlStr, strings.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+t.AccessToken)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req, nil
	}

	req, err := buildReq(token)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := network.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mal api request: %w", err)
	}

	// The Interceptor: If 401, attempt seamless refresh.
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close() // Close the dead response

		if err := performTokenRefresh(token.RefreshToken); err != nil {
			return nil, fmt.Errorf("unauthorized and token refresh failed: run `anisan mal auth`")
		}

		// Reload the fresh token and retry the exact same request
		freshToken, _ := LoadToken()
		retryReq, _ := buildReq(freshToken)

		return network.Client.Do(retryReq)
	}

	return resp, nil
}

// performTokenRefresh handles the OAuth2 refresh_token grant type.
func performTokenRefresh(refreshToken string) error {
	clientID := viper.GetString("tracker.mal.client_id")
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequest(http.MethodPost, "https://myanimelist.net/v1/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := network.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh rejected")
	}

	var tokenData Token
	if err := json.NewDecoder(resp.Body).Decode(&tokenData); err != nil {
		return err
	}
	return SaveToken(&tokenData)
}
