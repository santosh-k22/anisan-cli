// Package custom provides a bridge between the Go core and Lua-based scraper scripts.
package custom

import (
	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/internal/cache"
	"github.com/anisan-cli/anisan/source"
	lua "github.com/yuin/gopher-lua"
)

func (s *luaSource) Search(query string) ([]*source.Anime, error) {
	cacheKey := cache.GenerateKey(query, s.Name())
	var cachedAnimes []*source.Anime
	if cache.Read(cacheKey, &cachedAnimes) {
		for _, a := range cachedAnimes {
			a.Source = s
		}
		return cachedAnimes, nil
	}

	val, err := s.call(constant.SearchAnimesFn, lua.LTTable, lua.LString(query))
	if err != nil {
		return nil, err
	}

	table := val.(*lua.LTable)
	// Pre-allocate slice capacity based on the Lua table length to minimize reallocations.
	animes := make([]*source.Anime, 0, table.Len())

	var errs []error
	table.ForEach(func(k, v lua.LValue) {
		if k.Type() != lua.LTNumber || v.Type() != lua.LTTable {
			return // Skip invalid entries
		}

		// Direct primitive cast to avoid unnecessary string allocations.
		idx := uint16(k.(lua.LNumber))

		anime, err := animeFromTable(v.(*lua.LTable), idx)
		if err != nil {
			errs = append(errs, err)
			return
		}

		anime.Source = s
		animes = append(animes, anime)
	})

	if len(animes) == 0 && len(errs) > 0 {
		return nil, errs[0]
	}

	if len(animes) > 0 {
		_ = cache.Write(cacheKey, animes)
	}

	return animes, nil
}
