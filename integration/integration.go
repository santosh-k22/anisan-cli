// Package integration provides high-level coordination between media scraping and third-party tracking services.
package integration

import (
	"github.com/anisan-cli/anisan/integration/anilist"
	"github.com/anisan-cli/anisan/source"
)

// Integrator defines the common interface for external service integrations that support activity synchronization.
type Integrator interface {
	// MarkWatched synchronizes the watch status of an episode with the external service.
	MarkWatched(episode *source.Episode) error
}

var (
	Anilist Integrator = anilist.New()
)
