// Package mal provides a client for the MyAnimeList REST API.
package mal

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/anisan-cli/anisan/network"
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

const (
	keyringService = "anisan"
	keyringUser    = "mal-token"
)

// Token encapsulates the OAuth2 access and refresh tokens retrieved from MyAnimeList.
type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// SaveToken serializes and persists the OAuth2 token to the system keyring.
func SaveToken(token *Token) error {
	bytes, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return keyring.Set(keyringService, keyringUser, string(bytes))
}

// LoadToken retrieves and deserializes the OAuth2 token from the system keyring.
func LoadToken() (*Token, error) {
	str, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		return nil, err
	}

	var token Token
	if err := json.Unmarshal([]byte(str), &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// DeleteToken permanently removes the MyAnimeList token from the system keyring.
func DeleteToken() error {
	return keyring.Delete(keyringService, keyringUser)
}

// GeneratePKCE creates a 128-byte securely randomized Code Challenge for MAL OAuth2.
func GeneratePKCE() (string, error) {
	b := make([]byte, 128)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// MAL requires a URL-safe base64 string without padding.
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "="), nil
}

// ExchangeToken performs the OAuth2 code exchange using the PKCE verifier.
func ExchangeToken(authCode, codeVerifier string) error {
	clientID := viper.GetString("tracker.mal.client_id")
	if clientID == "" {
		return fmt.Errorf("MAL client_id is missing in configuration")
	}

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("code", authCode)
	data.Set("code_verifier", codeVerifier)
	data.Set("grant_type", "authorization_code")

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
		return fmt.Errorf("failed to exchange token: %d", resp.StatusCode)
	}

	var tokenData Token
	if err := json.NewDecoder(resp.Body).Decode(&tokenData); err != nil {
		return err
	}

	return SaveToken(&tokenData)
}
