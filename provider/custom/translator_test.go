package custom

import (
	"testing"

	"github.com/anisan-cli/anisan/source"
	. "github.com/smartystreets/goconvey/convey"
	lua "github.com/yuin/gopher-lua"
)

func TestAnimeFromTable(t *testing.T) {
	Convey("animeFromTable", t, func() {
		L := lua.NewState()
		defer L.Close()

		Convey("Should extract anime from valid Lua table", func() {
			tbl := L.NewTable()
			tbl.RawSetString("name", lua.LString("Bleach"))
			tbl.RawSetString("url", lua.LString("https://example.com/bleach"))
			tbl.RawSetString("cover", lua.LString("https://example.com/cover.jpg"))

			anime, err := animeFromTable(tbl, 0)
			So(err, ShouldBeNil)
			So(anime.Name, ShouldEqual, "Bleach")
			So(anime.URL, ShouldEqual, "https://example.com/bleach")
			So(anime.Metadata.Cover.ExtraLarge, ShouldEqual, "https://example.com/cover.jpg")
		})

		Convey("Should fail when required field 'name' is missing", func() {
			tbl := L.NewTable()
			tbl.RawSetString("url", lua.LString("https://example.com"))

			_, err := animeFromTable(tbl, 0)
			So(err, ShouldNotBeNil)
		})

		Convey("Should handle optional genres", func() {
			tbl := L.NewTable()
			tbl.RawSetString("name", lua.LString("Naruto"))
			tbl.RawSetString("url", lua.LString("https://example.com/naruto"))
			tbl.RawSetString("genres", lua.LString("Action, Adventure, Fantasy"))

			anime, err := animeFromTable(tbl, 0)
			So(err, ShouldBeNil)
			So(anime.Metadata.Genres, ShouldHaveLength, 3)
			So(anime.Metadata.Genres[0], ShouldEqual, "Action")
		})
	})
}

func TestVideoFromTable(t *testing.T) {
	Convey("videoFromTable", t, func() {
		L := lua.NewState()
		defer L.Close()

		Convey("Should extract video with URL", func() {
			tbl := L.NewTable()
			tbl.RawSetString("url", lua.LString("https://example.com/stream.m3u8"))
			tbl.RawSetString("quality", lua.LString("1080p"))

			video, err := videoFromTable(tbl, 0)
			So(err, ShouldBeNil)
			So(video.URL, ShouldEqual, "https://example.com/stream.m3u8")
			So(video.Quality, ShouldEqual, "1080p")
		})

		Convey("Should extract headers from Lua table", func() {
			tbl := L.NewTable()
			tbl.RawSetString("url", lua.LString("https://example.com/stream.m3u8"))

			headers := L.NewTable()
			headers.RawSetString("Referer", lua.LString("https://allanime.to"))
			headers.RawSetString("User-Agent", lua.LString("Mozilla/5.0"))
			tbl.RawSetString("headers", headers)

			video, err := videoFromTable(tbl, 0)
			So(err, ShouldBeNil)
			So(video.Headers, ShouldNotBeNil)
			So(video.Headers["Referer"], ShouldEqual, "https://allanime.to")
			So(video.Headers["User-Agent"], ShouldEqual, "Mozilla/5.0")
		})

		Convey("Should fail when URL is missing", func() {
			tbl := L.NewTable()
			tbl.RawSetString("quality", lua.LString("720p"))

			_, err := videoFromTable(tbl, 0)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestEpisodeFromTable(t *testing.T) {
	Convey("episodeFromTable", t, func() {
		L := lua.NewState()
		defer L.Close()

		Convey("Should extract episode with parent anime reference", func() {
			anime := &source.Anime{Name: "Bleach"}
			tbl := L.NewTable()
			tbl.RawSetString("name", lua.LString("Episode 1"))
			tbl.RawSetString("url", lua.LString("https://example.com/ep1"))

			episode, err := episodeFromTable(tbl, anime, 0)
			So(err, ShouldBeNil)
			So(episode.Name, ShouldEqual, "Episode 1")
			So(episode.URL, ShouldEqual, "https://example.com/ep1")
			So(episode.Anime, ShouldEqual, anime)
		})
	})
}
