// Package custom provides a bridge between the Go core and Lua-based scraper scripts.
package custom

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/anisan-cli/anisan/source"
	"github.com/samber/lo"
	lua "github.com/yuin/gopher-lua"
)

// Helper to get string from table with default
func getString(table *lua.LTable, key string) string {
	val := table.RawGetString(key)
	if val.Type() == lua.LTString {
		return val.String()
	}
	return ""
}

// Helper to get string list from table (comma-separated or table)
func getStringList(table *lua.LTable, key string) []string {
	val := table.RawGetString(key)
	if val.Type() == lua.LTString {
		return lo.Map(strings.Split(val.String(), ","), func(s string, _ int) string {
			return strings.TrimSpace(s)
		})
	}
	if val.Type() == lua.LTTable {
		var list []string
		val.(*lua.LTable).ForEach(func(_, v lua.LValue) {
			if v.Type() == lua.LTString {
				list = append(list, v.String())
			}
		})
		return list
	}
	return nil
}

func animeFromTable(table *lua.LTable, index uint16) (*source.Anime, error) {
	name := getString(table, "name")
	url := getString(table, "url")

	if name == "" || url == "" {
		return nil, fmt.Errorf("anime must have name and url")
	}

	anime := &source.Anime{
		Name:  name,
		URL:   url,
		Index: index,
		ID:    url, // Use URL as ID for custom providers usually
	}

	// Metadata
	anime.Metadata.Summary = getString(table, "summary")
	anime.Metadata.Cover.ExtraLarge = getString(table, "cover")
	anime.Metadata.BannerImage = getString(table, "banner")
	anime.Metadata.Genres = getStringList(table, "genres")
	anime.Metadata.Status = getString(table, "status")
	anime.Metadata.Synonyms = getStringList(table, "synonyms")

	return anime, nil
}

func episodeFromTable(table *lua.LTable, anime *source.Anime, index uint16) (*source.Episode, error) {
	name := getString(table, "name")
	url := getString(table, "url")

	if name == "" || url == "" {
		return nil, fmt.Errorf("episode must have name and url")
	}

	finalIndex := index

	// Always try to parse from name first (Most reliable for sorting)
	// Matches "25", "25.5", "Episode 25", etc.
	re := regexp.MustCompile(`(\d+(\.\d+)?)`)
	matches := re.FindAllString(name, -1)
	parsedFromName := false
	if len(matches) > 0 {
		// Take the last number found (usually the episode number in "Season 1 Episode 25")
		lastMatch := matches[len(matches)-1]
		if parsed, err := strconv.ParseFloat(lastMatch, 64); err == nil {
			finalIndex = uint16(parsed)
			parsedFromName = true
		}
	}

	// Only check Lua 'number' if we couldn't parse from name
	if !parsedFromName {
		val := table.RawGetString("number")
		if val.Type() == lua.LTNumber {
			finalIndex = uint16(val.(lua.LNumber))
		} else if val.Type() == lua.LTString {
			if parsed, err := strconv.ParseUint(val.String(), 10, 16); err == nil {
				finalIndex = uint16(parsed)
			}
		}
	}

	// Discovery verification can be performed here for diagnostic purposes.

	ep := &source.Episode{
		Name:   name,
		URL:    url,
		Index:  finalIndex,
		ID:     url,
		Volume: getString(table, "volume"),
		Anime:  anime,
	}

	return ep, nil
}

func videoFromTable(table *lua.LTable, index uint16) (*source.Video, error) {
	url := getString(table, "url")
	quality := getString(table, "quality")

	if url == "" {
		return nil, fmt.Errorf("video must have url")
	}

	video := &source.Video{
		URL:       url,
		Quality:   quality,
		Extension: getString(table, "extension"),
		Index:     index,
		Headers:   make(map[string]string),
	}

	// Headers
	headersTbl := table.RawGetString("headers")
	if headersTbl.Type() == lua.LTTable {
		headersTbl.(*lua.LTable).ForEach(func(k, v lua.LValue) {
			video.Headers[k.String()] = v.String()
		})
	}

	return video, nil
}

func animeToTable(L *lua.LState, anime *source.Anime) *lua.LTable {
	table := L.NewTable()
	table.RawSetString("name", lua.LString(anime.Name))
	table.RawSetString("url", lua.LString(anime.URL))
	return table
}

func episodeToTable(L *lua.LState, episode *source.Episode) *lua.LTable {
	table := L.NewTable()
	table.RawSetString("name", lua.LString(episode.Name))
	table.RawSetString("url", lua.LString(episode.URL))
	return table
}
