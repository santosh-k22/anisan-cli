// Package mal provides a client for the MyAnimeList REST API.
package mal

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "anisan"
	keyringUser    = "mal-token"
	// ClientID is the public client ID (User needs to provide this or use a default)
	// For open source CLI tools, it's often tricky. We might need to ask user to set it.
	defaultClientID = "8cdf92d70fbd7228dab4098523f6be68"
	authEndpoint    = "https://myanimelist.net/v1/oauth2/authorize"
	tokenEndpoint   = "https://myanimelist.net/v1/oauth2/token"
	redirectURI     = "http://localhost:8080/callback"
)

// Token encapsulates the OAuth2 access and refresh tokens retrieved from MyAnimeList.
type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// GenerateCodeVerifier generates a cryptographically secure random string for the PKCE challenge.
func GenerateCodeVerifier() (string, error) {
	b := make([]byte, 32) // High entropy
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GetAuthURL constructs the authorization URI for the OAuth2 PKCE flow.
func GetAuthURL(codeVerifier string, clientID string) string {
	if clientID == "" {
		clientID = defaultClientID
	}

	// The 'plain' challenge method is used where the challenge equals the verifier.

	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("client_id", clientID)
	v.Set("code_challenge", codeVerifier) // plain
	v.Set("code_challenge_method", "plain")
	v.Set("redirect_uri", redirectURI)

	return authEndpoint + "?" + v.Encode()
}

// ExchangeCode trades the authorization code for a set of OAuth2 tokens.
func ExchangeCode(code string, codeVerifier string, clientID string) (*Token, error) {
	if clientID == "" {
		clientID = defaultClientID
	}

	values := url.Values{}
	values.Set("client_id", clientID)
	values.Set("code", code)
	values.Set("code_verifier", codeVerifier)
	values.Set("grant_type", "authorization_code")
	values.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mal authentication failed: %s", string(body))
	}

	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	return &token, nil
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

// RefreshAPI facilitates the renewal of an expired access token using the stored refresh token.
func RefreshAPI(clientID string) error {
	token, err := LoadToken()
	if err != nil {
		return err
	}

	if clientID == "" {
		clientID = defaultClientID
	}

	values := url.Values{}
	values.Set("client_id", clientID)
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", token.RefreshToken)

	req, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to refresh token: status %d", resp.StatusCode)
	}

	var newToken Token
	if err := json.NewDecoder(resp.Body).Decode(&newToken); err != nil {
		return err
	}

	return SaveToken(&newToken)
}
