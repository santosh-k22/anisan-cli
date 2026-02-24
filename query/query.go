// Package query manages the persistence and retrieval of search query history and suggestions.
package query

import (
	"strings"

	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/where"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/metafates/gache"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

type queryRecord struct {
	Rank  int    `json:"rank"`
	Query string `json:"query"`
}

var cacher = gache.New[map[string]*queryRecord](
	&gache.Options{
		Path:       where.Queries(),
		FileSystem: &filesystem.GacheFs{},
	},
)

var suggestionCache = make(map[string][]*queryRecord)

// Remember records a search query in the persistent history or increments its popularity rank.
func Remember(q string, weight int) error {
	q = sanitize(q)
	cached, expired, err := cacher.Get()
	if expired || err != nil || cached == nil {
		cached = make(map[string]*queryRecord)
	}

	if record, ok := cached[q]; ok {
		record.Rank += weight
	} else {
		cached[q] = &queryRecord{Rank: weight, Query: q}
	}

	return cacher.Set(cached)
}

// Suggest returns the most relevant historical query suggestion for a partial input.
func Suggest(q string) mo.Option[string] {
	suggestions := SuggestMany(q)
	if len(suggestions) == 0 {
		return mo.None[string]()
	}
	return mo.Some(suggestions[0])
}

// SuggestMany returns a collection of historical query suggestions matching the partial input, sorted by popularity rank.
func SuggestMany(q string) []string {
	if !viper.GetBool(key.SearchShowQuerySuggestions) {
		return []string{}
	}

	q = sanitize(q)
	var records []*queryRecord

	if prev, ok := suggestionCache[q]; ok {
		records = prev
	} else {
		cached, expired, err := cacher.Get()
		if err != nil || expired || cached == nil {
			return []string{}
		}

		for _, record := range cached {
			if fuzzy.Match(q, record.Query) {
				records = append(records, record)
			}
		}

		slices.SortFunc(records, func(a, b *queryRecord) int {
			return b.Rank - a.Rank // Descending rank
		})

		suggestionCache[q] = records
	}

	return lo.Map(records, func(r *queryRecord, _ int) string {
		return r.Query
	})
}

func sanitize(q string) string {
	return strings.TrimSpace(strings.ToLower(q))
}
