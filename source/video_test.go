package source

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestVideo(t *testing.T) {
	Convey("Video", t, func() {
		v := &Video{
			URL:     "http://example.com/vid.mp4",
			Quality: "1080p",
		}

		Convey("String representation", func() {
			So(v.String(), ShouldEqual, "1080p")
			v.Quality = ""
			So(v.String(), ShouldEqual, "http://example.com/vid.mp4")
		})
	})
}
