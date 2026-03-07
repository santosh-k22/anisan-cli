// Package inline provides the implementation for the application's non-interactive, programmable execution mode.
package inline

import (
	"encoding/json"

	"github.com/anisan-cli/anisan/anilist"
	"github.com/anisan-cli/anisan/mal"
	"github.com/anisan-cli/anisan/source"
)

type Anime struct {
	// Source is the name of the provider.
	Source string `json:"source"`
	// Anime is the anime object from the source.
	Anime *source.Anime `json:"anime"`
	// Anilist is the matched Anilist entry (optional).
	Anilist *anilist.Anime `json:"anilist,omitempty"`
	// Mal is the matched MyAnimeList entry (optional).
	Mal *mal.Anime `json:"mal,omitempty"`
}

type Output struct {
	Query  string   `json:"query"`
	Result []*Anime `json:"result"`
}

func asJson(animes []*source.Anime, query string, includeAnilist, includeMal bool) ([]byte, error) {
	var result = make([]*Anime, len(animes))
	for i, a := range animes {
		var al *anilist.Anime
		if includeAnilist {
			al = a.Anilist.OrElse(nil)
		}

		var m *mal.Anime
		if includeMal {
			m = a.Mal.OrElse(nil)
		}

		result[i] = &Anime{
			Source:  a.Source.Name(),
			Anime:   a,
			Anilist: al,
			Mal:     m,
		}
	}

	return json.Marshal(&Output{
		Query:  query,
		Result: result,
	})
}
