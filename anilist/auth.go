// Package anilist provides a client for the Anilist GraphQL API.
package anilist

import (
	"fmt"

	"github.com/anisan-cli/anisan/log"
	"github.com/zalando/go-keyring"
)

const (
	// keyringService is the generic service identifier for the system keyring.
	keyringService = "anisan"
	// keyringUser is the specific key used for storing the Anilist OAuth token.
	keyringUser = "anilist_token"
)

// SetToken saves the Anilist token to the system keyring.
func SetToken(token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}
	err := keyring.Set(keyringService, keyringUser, token)
	if err != nil {
		log.Error("Failed to save token to keyring: " + err.Error())
		return err
	}
	return nil
}

// GetToken retrieves the Anilist token from the system keyring.
func GetToken() (string, error) {
	token, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		// Log debug only, as it's common to not have a token yet
		log.Infof("No token found in keyring: %v", err)
		return "", err
	}
	return token, nil
}

// DeleteToken removes the Anilist token from the system keyring.
func DeleteToken() error {
	err := keyring.Delete(keyringService, keyringUser)
	if err != nil {
		log.Error("Failed to delete token from keyring: " + err.Error())
		return err
	}
	return nil
}
