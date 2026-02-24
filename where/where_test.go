package where

import (
	"testing"

	"github.com/anisan-cli/anisan/filesystem"
	"github.com/samber/lo"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	// Use in-memory filesystem for tests to avoid creating real directories
	filesystem.SetMemMapFs()
}

func TestPaths(t *testing.T) {
	Convey("Path functions", t, func() {
		Convey("Config()", func() {
			path := Config()
			So(path, ShouldNotBeEmpty)
			So(lo.Must(filesystem.API().IsDir(path)), ShouldBeTrue)
		})

		Convey("Cache()", func() {
			path := Cache()
			So(path, ShouldNotBeEmpty)
			So(lo.Must(filesystem.API().IsDir(path)), ShouldBeTrue)
		})

		Convey("Logs()", func() {
			path := Logs()
			So(path, ShouldNotBeEmpty)
			So(lo.Must(filesystem.API().IsDir(path)), ShouldBeTrue)
		})
	})
}
