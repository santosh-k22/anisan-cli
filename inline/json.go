// Package inline provides the implementation for the application's non-interactive, programmable execution mode.
package inline

import (
	"encoding/json"

	"github.com/anisan-cli/anisan/anilist"
	"github.com/anisan-cli/anisan/source"
)

type Anime struct {
	// Source is the name of the provider.
	Source string `json:"source"`
	// Anime is the anime object from the source.
	Anime *source.Anime `json:"animel"`
	// Anilist is the matched Anilist entry (optional).
	Anilist *anilist.Anime `json:"anilist,omitempty"`
}

type Output struct {
	Query  string   `json:"query"`
	Result []*Anime `json:"result"`
}

func asJson(animes []*source.Anime, query string, includeAnilist bool) ([]byte, error) {
	var result = make([]*Anime, len(animes))
	for i, a := range animes {
		var al *anilist.Anime
		if includeAnilist {
			al = a.Anilist.OrElse(nil)
		}

		result[i] = &Anime{
			Source:  a.Source.Name(),
			Anime:   a,
			Anilist: al,
		}
	}

	return json.Marshal(&Output{
		Query:  query,
		Result: result,
	})
}
