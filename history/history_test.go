package history

import (
	"fmt"
	"testing"

	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/source"
	. "github.com/smartystreets/goconvey/convey"
)

type testSource struct{}

func (testSource) Name() string {
	panic("")
}

func (testSource) Search(_ string) ([]*source.Anime, error) {
	panic("")
}

func (testSource) EpisodesOf(_ *source.Anime) ([]*source.Episode, error) {
	panic("")
}

func (testSource) VideosOf(_ *source.Episode) ([]*source.Video, error) {
	panic("")
}

func (testSource) ID() string {
	return "test source"
}

func init() {
	filesystem.SetMemMapFs()
}

func TestHistory(t *testing.T) {
	Convey("Given a episode", t, func() {
		episode := source.Episode{
			Name:  "adwad",
			URL:   "dwaofa",
			Index: 42069,
			ID:    "fawfa",
		}
		anime := source.Anime{
			Name:     "dawf",
			URL:      "fwa",
			Index:    1337,
			ID:       "wjakfkawgjj",
			Source:   testSource{},
			Episodes: []*source.Episode{&episode},
		}
		episode.Anime = &anime

		Convey("When saving the episode", func() {
			err := Save(&episode, 0.0)
			Convey("Then the error should be nil", func() {
				So(err, ShouldBeNil)

				Convey("And the episode should be saved", func() {
					episodes, err := Get()
					So(err, ShouldBeNil)
					So(len(episodes), ShouldBeGreaterThan, 0)
					So(episodes[fmt.Sprintf("%s (%s)", episode.Anime.Name, episode.Source().ID())].Name, ShouldEqual, episode.Name)
				})
			})
		})
	})
}
