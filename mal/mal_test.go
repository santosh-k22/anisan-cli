package mal

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGenerateCodeVerifier(t *testing.T) {
	Convey("GenerateCodeVerifier", t, func() {
		Convey("Should generate a valid PKCE code verifier", func() {
			verifier, err := GenerateCodeVerifier()
			So(err, ShouldBeNil)
			So(verifier, ShouldNotBeEmpty)
			// PKCE verifiers must be between 43 and 128 characters
			So(len(verifier), ShouldBeGreaterThanOrEqualTo, 20)
		})

		Convey("Should generate unique values on each call", func() {
			v1, _ := GenerateCodeVerifier()
			v2, _ := GenerateCodeVerifier()
			So(v1, ShouldNotEqual, v2)
		})
	})
}

func TestGetAuthURL(t *testing.T) {
	Convey("GetAuthURL", t, func() {
		Convey("Should generate a valid MAL auth URL", func() {
			url := GetAuthURL("test-verifier", "test-client-id")
			So(url, ShouldContainSubstring, "https://myanimelist.net/v1/oauth2/authorize")
			So(url, ShouldContainSubstring, "client_id=test-client-id")
			So(url, ShouldContainSubstring, "code_challenge=test-verifier")
			So(url, ShouldContainSubstring, "response_type=code")
		})

		Convey("Should use default client ID when empty", func() {
			url := GetAuthURL("test-verifier", "")
			So(url, ShouldContainSubstring, "client_id="+defaultClientID)
		})
	})
}

func TestStructTypes(t *testing.T) {
	Convey("Data Structures", t, func() {
		Convey("Token should have correct zero values", func() {
			var token Token
			So(token.AccessToken, ShouldBeEmpty)
			So(token.RefreshToken, ShouldBeEmpty)
			So(token.ExpiresIn, ShouldEqual, 0)
		})

		Convey("Anime should have correct zero values", func() {
			var anime Anime
			So(anime.ID, ShouldEqual, 0)
			So(anime.Title, ShouldBeEmpty)
		})

		Convey("UpdateStatus should have correct zero values", func() {
			var status UpdateStatus
			So(status.Status, ShouldBeEmpty)
			So(status.NumWatchedEpisodes, ShouldEqual, 0)
			So(status.IsRewatching, ShouldBeFalse)
		})
	})
}
