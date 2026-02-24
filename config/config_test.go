package config

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func TestSetup(t *testing.T) {
	Convey("Config Setup", t, func() {
		Convey("Should initialize without error", func() {
			err := Setup()
			So(err, ShouldBeNil)
		})

		Convey("Should have default values populated", func() {
			_ = Setup()
			// After setup, viper should have defaults from Default map
			for name, field := range Default {
				val := viper.Get(name)
				So(val, ShouldNotBeNil)
				_ = field // just ensuring iteration works
			}
		})

		Convey("EnvKeyReplacer should convert dots to underscores", func() {
			result := EnvKeyReplacer.Replace("metadata.fetch.anilist")
			So(result, ShouldEqual, "metadata_fetch_anilist")
		})
	})
}
