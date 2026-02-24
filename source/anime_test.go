package source

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAnime(t *testing.T) {
	Convey("Anime", t, func() {
		a := &Anime{Name: "One Piece"}

		Convey("Dirname", func() {
			So(a.Dirname(), ShouldEqual, "One_Piece")
		})

		Convey("String", func() {
			So(a.String(), ShouldEqual, "One Piece")
		})

		Convey("GetCover - Empty", func() {
			_, err := a.GetCover()
			So(err, ShouldNotBeNil)
		})

		Convey("GetCover - Priority", func() {
			a.Metadata.Cover.Medium = "med"
			cover, err := a.GetCover()
			So(err, ShouldBeNil)
			So(cover, ShouldEqual, "med")

			a.Metadata.Cover.Large = "large"
			cover, err = a.GetCover()
			So(err, ShouldBeNil)
			So(cover, ShouldEqual, "large")

			a.Metadata.Cover.ExtraLarge = "xl"
			cover, err = a.GetCover()
			So(err, ShouldBeNil)
			So(cover, ShouldEqual, "xl")
		})
	})
}
