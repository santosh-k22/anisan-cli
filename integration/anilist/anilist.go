// Package anilist implements the Integrator interface for the Anilist service, providing watch list synchronization via GraphQL.
package anilist

import (
	"github.com/anisan-cli/anisan/key"
	"github.com/spf13/viper"
)

type Anilist struct {
	token string
}

// New initializes a new Anilist service integration instance.
func New() *Anilist {
	return &Anilist{}
}

func (a *Anilist) id() string {
	return viper.GetString(key.AnilistID)
}

// AuthURL returns the OAuth2 authorization endpoint for the Anilist service.
func (a *Anilist) AuthURL() string {
	return "https://anilist.co/api/v2/oauth/authorize?client_id=" + a.id() + "&response_type=code&redirect_uri=https://anilist.co/api/v2/oauth/pin"
}
