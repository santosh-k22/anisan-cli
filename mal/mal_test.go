package mal

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestStructTypes(t *testing.T) {
	Convey("Data Structures", t, func() {
		Convey("Token should have correct zero values", func() {
			var token Token
			So(token.AccessToken, ShouldBeEmpty)
			So(token.RefreshToken, ShouldBeEmpty)
			So(token.ExpiresIn, ShouldEqual, 0)
		})

		Convey("Anime should have correct zero values", func() {
			var anime Anime
			So(anime.ID, ShouldEqual, 0)
			So(anime.Title, ShouldBeEmpty)
		})
	})
}
