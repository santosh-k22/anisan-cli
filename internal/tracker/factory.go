package tracker

import (
	"log"

	"github.com/anisan-cli/anisan/internal/tracker/anilist"
	"github.com/anisan-cli/anisan/internal/tracker/mal"
	"github.com/anisan-cli/anisan/key"
	"github.com/spf13/viper"
)

// InitializeTracker resolves the active tracking backend from configuration
// and returns its corresponding MediaTracker implementation.
func InitializeTracker() MediaTracker {
	backend := viper.GetString(key.TrackerBackend)
	switch backend {
	case "mal":
		return mal.NewClient()
	case "anilist":
		return anilist.NewClient()
	default:
		log.Printf("warning: unsupported tracker backend %q specified, falling back to anilist", backend)
		return anilist.NewClient()
	}
}
