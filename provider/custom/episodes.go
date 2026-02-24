// Package custom provides a bridge between the Go core and Lua-based scraper scripts.
package custom

import (
	"strconv"

	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/internal/cache"
	"github.com/anisan-cli/anisan/source"
	lua "github.com/yuin/gopher-lua"
)

func (s *luaSource) EpisodesOf(anime *source.Anime) ([]*source.Episode, error) {
	cacheKey := cache.GenerateKey(anime.URL, s.Name()+"_episodes")
	var cachedEpisodes []*source.Episode
	if cache.Read(cacheKey, &cachedEpisodes) {
		for _, ep := range cachedEpisodes {
			ep.Anime = anime
		}
		anime.Episodes = cachedEpisodes
		return cachedEpisodes, nil
	}

	val, err := s.call(constant.AnimeEpisodesFn, lua.LTTable, animeToTable(s.state, anime))
	if err != nil {
		return nil, err
	}

	table := val.(*lua.LTable)
	var episodes []*source.Episode
	var errs []error

	table.ForEach(func(k, v lua.LValue) {
		if k.Type() != lua.LTNumber || v.Type() != lua.LTTable {
			return
		}

		idx, err := strconv.ParseUint(k.String(), 10, 16)
		if err != nil {
			errs = append(errs, err)
			return
		}

		ep, err := episodeFromTable(v.(*lua.LTable), anime, uint16(idx))
		if err != nil {
			errs = append(errs, err)
			return
		}

		episodes = append(episodes, ep)
	})

	if len(episodes) == 0 && len(errs) > 0 {
		return nil, errs[0]
	}

	if len(episodes) > 0 {
		_ = cache.Write(cacheKey, episodes)
	}

	anime.Episodes = episodes
	return episodes, nil
}
