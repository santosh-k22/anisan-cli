// Package auth provides a high-level API for persisting and retrieving user credentials from the system keyring.
package auth

import (
	"github.com/zalando/go-keyring"
)

const (
	service = "anisan-cli"
	user    = "anilist-token"
)

// SetToken persists the Anilist OAuth token to the system keyring.
func SetToken(token string) error {
	return keyring.Set(service, user, token)
}

// GetToken retrieves the Anilist OAuth token from the system keyring.
func GetToken() (string, error) {
	return keyring.Get(service, user)
}

// DeleteToken removes the Anilist OAuth token from the system keyring.
func DeleteToken() error {
	return keyring.Delete(service, user)
}
