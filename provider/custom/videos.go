// Package custom provides a bridge between the Go core and Lua-based scraper scripts.
package custom

import (
	"strconv"

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
	var videos []*source.Video
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

		vid, err := videoFromTable(v.(*lua.LTable), uint16(idx))
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
