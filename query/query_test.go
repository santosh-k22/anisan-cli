package query

import (
	"testing"

	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/key"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func init() {
	filesystem.SetMemMapFs()
	// Ensure suggestions are enabled for tests
	viper.Set(key.SearchShowQuerySuggestions, true)
}

func TestQuery(t *testing.T) {
	Convey("Given query history", t, func() {
		// Clear cache for isolation (in memory fs makes this easier but global vars persist)
		// We can't easily clear the global var `suggestionCache` or `cacher`'s internal map without a helper.
		// However, MemMapFs reset generally handles file persistence, but in-memory map `suggestionCache` persists.
		// Let's rely on unique queries.

		q1 := "naruto"
		q2 := "bleach"

		Convey("When remembering queries", func() {
			err := Remember(q1, 1)
			So(err, ShouldBeNil)
			err = Remember(q2, 10) // Higher weight
			So(err, ShouldBeNil)

			Convey("Then suggestions should be sorted by rank", func() {
				// Clear memory cache to force read from file
				suggestionCache = make(map[string][]*queryRecord)

				// We need to ensure viper is set correctly
				viper.Set(key.SearchShowQuerySuggestions, true)

				s := SuggestMany("ble")
				So(len(s), ShouldBeGreaterThanOrEqualTo, 1)
				So(s[0], ShouldEqual, "bleach")
			})

			Convey("It sanitizes input", func() {
				So(sanitize("  NARUTO  "), ShouldEqual, "naruto")
			})
		})
	})
}
