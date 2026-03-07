// Package custom provides a bridge between the Go core and Lua-based scraper scripts.
package custom

import (
	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/source"
	lua "github.com/yuin/gopher-lua"
)

func (s *luaSource) VideosOf(episode *source.Episode) ([]*source.Video, error) {
	// No caching for videos (links expire)

	val, err := s.call(constant.EpisodeVideosFn, lua.LTTable, episodeToTable(s.state, episode))
	if err != nil {
		return nil, err
	}

	table := val.(*lua.LTable)
	// Pre-allocate slice capacity to reduce memory pressure during extraction.
	videos := make([]*source.Video, 0, table.Len())
	var errs []error

	table.ForEach(func(k, v lua.LValue) {
		if k.Type() != lua.LTNumber || v.Type() != lua.LTTable {
			return
		}

		// Direct primitive cast.
		idx := uint16(k.(lua.LNumber))

		vid, err := videoFromTable(v.(*lua.LTable), idx)
		if err != nil {
			errs = append(errs, err)
			return
		}

		videos = append(videos, vid)
	})

	if len(videos) == 0 && len(errs) > 0 {
		return nil, errs[0]
	}

	return videos, nil
}
