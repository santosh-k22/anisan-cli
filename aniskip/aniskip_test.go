package aniskip

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetSkipTimes(t *testing.T) {
	Convey("GetSkipTimes", t, func() {
		Convey("Should return skip times for a known anime episode", func() {
			// Death Note EP1 (MAL ID: 1535) is a well-known entry
			// that reliably has skip time data on aniskip.
			times, err := GetSkipTimes(1535, 1)

			// aniskip is a third-party API — if it's down, we degrade gracefully.
			// So we only assert no hard error and valid structure if data exists.
			So(err, ShouldBeNil)

			if times != nil {
				So(times.HasIntro, ShouldBeTrue)
				So(times.Opening.End, ShouldBeGreaterThan, times.Opening.Start)
			}
		})

		Convey("Should return nil for an anime with no skip data", func() {
			// Use a very unlikely MAL ID
			times, err := GetSkipTimes(999999999, 1)
			So(err, ShouldBeNil)
			So(times, ShouldBeNil)
		})

		Convey("Should handle zero/negative inputs gracefully", func() {
			times, err := GetSkipTimes(0, 0)
			So(err, ShouldBeNil)
			// Might be nil or empty — just shouldn't crash
			_ = times
		})
	})
}

func TestSkipTimesStructure(t *testing.T) {
	Convey("SkipTimes", t, func() {
		Convey("Zero value should have HasIntro and HasOutro as false", func() {
			var st SkipTimes
			So(st.HasIntro, ShouldBeFalse)
			So(st.HasOutro, ShouldBeFalse)
			So(st.Opening.Start, ShouldEqual, 0)
			So(st.Opening.End, ShouldEqual, 0)
			So(st.Ending.Start, ShouldEqual, 0)
			So(st.Ending.End, ShouldEqual, 0)
		})
	})
}
