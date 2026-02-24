package source

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEpisode(t *testing.T) {
	Convey("Episode", t, func() {
		anime := &Anime{
			Name:   "Test Anime",
			Source: &testSource{},
		}

		ep := &Episode{
			Name:  "Episode 1",
			Anime: anime,
		}

		Convey("String", func() {
			So(ep.String(), ShouldEqual, "Episode 1")
		})

		Convey("Source", func() {
			So(ep.Source(), ShouldNotBeNil)
			So(ep.Source().Name(), ShouldEqual, "Test Source")
		})
	})
}

type testSource struct{}

func (testSource) Name() string                                { return "Test Source" }
func (testSource) ID() string                                  { return "test" }
func (testSource) Search(query string) ([]*Anime, error)       { return nil, nil }
func (testSource) EpisodesOf(anime *Anime) ([]*Episode, error) { return nil, nil }
func (testSource) VideosOf(episode *Episode) ([]*Video, error) { return nil, nil }
