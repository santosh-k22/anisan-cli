package icon

import (
	"testing"

	"github.com/anisan-cli/anisan/key"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func TestGet(t *testing.T) {
	Convey("Given a registered icon", t, func() {
		target := Lua

		Convey("It renders correctly for each variant", func() {
			for _, variant := range AvailableVariants() {
				Convey("variant="+variant, func() {
					viper.Set(key.IconsVariant, variant)
					result := Get(target)
					So(result, ShouldNotBeEmpty)
				})
			}
		})

		Convey("It returns empty for an unknown variant", func() {
			viper.Set(key.IconsVariant, "")
			result := Get(target)
			So(result, ShouldBeEmpty)
		})
	})
}
