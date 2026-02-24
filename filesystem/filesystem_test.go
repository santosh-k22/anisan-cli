package filesystem

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestApi(t *testing.T) {
	Convey("Filesystem API", t, func() {
		Convey("Should default to OsFs", func() {
			SetOsFs()
			fs := API()
			So(fs, ShouldNotBeNil)
			So(fs.Name(), ShouldEqual, "OsFs")
		})

		Convey("Should switch to MemMapFs", func() {
			SetMemMapFs()
			fs := API()
			So(fs, ShouldNotBeNil)
			So(fs.Name(), ShouldEqual, "MemMapFS")
		})
	})
}
