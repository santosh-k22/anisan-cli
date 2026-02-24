// Package inline provides the implementation for the application's non-interactive, programmable execution mode.
package inline

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/anisan-cli/anisan/source"
	"github.com/anisan-cli/anisan/util"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

type (
	AnimePicker    func([]*source.Anime) *source.Anime
	EpisodesFilter func([]*source.Episode) ([]*source.Episode, error)
)

type Options struct {
	Out                 io.Writer
	Sources             []source.Source
	IncludeAnilistAnime bool
	Json                bool
	Query               string
	AnimePicker         mo.Option[AnimePicker]
	EpisodesFilter      mo.Option[EpisodesFilter]
	Videos              bool
}

func ParseAnimePicker(kind, value string) (AnimePicker, error) {
	switch kind {
	case "first":
		return func(animes []*source.Anime) *source.Anime {
			if len(animes) == 0 {
				return nil
			}
			return animes[0]
		}, nil
	case "last":
		return func(animes []*source.Anime) *source.Anime {
			if len(animes) == 0 {
				return nil
			}
			return animes[len(animes)-1]
		}, nil
	case "exact":
		return func(animes []*source.Anime) *source.Anime {
			for _, a := range animes {
				if a.Name == value {
					return a
				}
			}
			return nil
		}, nil
	case "index":
		idx, err := strconv.ParseUint(value, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid index: %s", value)
		}
		return func(animes []*source.Anime) *source.Anime {
			if len(animes) == 0 {
				return nil
			}
			i := util.Min(idx, uint64(len(animes)-1))
			return animes[i]
		}, nil
	default:
		return nil, fmt.Errorf("unknown picker type: %s", kind)
	}
}

// ParseEpisodesFilter parses legacy string description of filter
// Format: "first", "last", "all", "From 1 To 5", "Sub 'Search'"
// This logic is kept compatible with legacy CLI args for now
func ParseEpisodesFilter(description string) (EpisodesFilter, error) {
	if description == "first" {
		return func(episodes []*source.Episode) ([]*source.Episode, error) {
			if len(episodes) == 0 {
				return episodes, nil
			}
			return episodes[:1], nil
		}, nil
	}
	if description == "last" {
		return func(episodes []*source.Episode) ([]*source.Episode, error) {
			if len(episodes) == 0 {
				return episodes, nil
			}
			return episodes[len(episodes)-1:], nil
		}, nil
	}
	if description == "all" {
		return func(episodes []*source.Episode) ([]*source.Episode, error) {
			return episodes, nil
		}, nil
	}

	// Range: "1-5"
	if strings.Contains(description, "-") {
		parts := strings.Split(description, "-")
		if len(parts) == 2 {
			from, err1 := strconv.ParseUint(parts[0], 10, 16)
			to, err2 := strconv.ParseUint(parts[1], 10, 16)
			if err1 == nil && err2 == nil {
				return func(episodes []*source.Episode) ([]*source.Episode, error) {
					start := util.Min(from, uint64(len(episodes)))
					end := util.Min(to+1, uint64(len(episodes)))
					if start > end {
						return []*source.Episode{}, nil
					}
					return episodes[start:end], nil
				}, nil
			}
		}
	}

	// Substring: "@text@"
	if strings.HasPrefix(description, "@") && strings.HasSuffix(description, "@") {
		sub := description[1 : len(description)-1]
		return func(episodes []*source.Episode) ([]*source.Episode, error) {
			return lo.Filter(episodes, func(e *source.Episode, _ int) bool {
				return strings.Contains(strings.ToLower(e.Name), strings.ToLower(sub))
			}), nil
		}, nil
	}

	// Single index: "5"
	if idx, err := strconv.ParseUint(description, 10, 16); err == nil {
		return func(episodes []*source.Episode) ([]*source.Episode, error) {
			if uint64(len(episodes)) <= idx {
				return []*source.Episode{}, nil
			}
			return []*source.Episode{episodes[idx]}, nil
		}, nil
	}

	return nil, fmt.Errorf("invalid episode filter: %s", description)
}
