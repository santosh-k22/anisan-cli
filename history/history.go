// Package history provides the implementation for tracking and persisting user media consumption state.
package history

import (
	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/source"
	"github.com/anisan-cli/anisan/where"
	"github.com/metafates/gache"
)

// cacher provides an abstracted, disk-backed registry for playback progress records.
var cacher = gache.New[map[string]*SavedEpisode](
	&gache.Options{
		Path:       where.History(),
		FileSystem: &filesystem.GacheFs{},
	},
)

// Get returns the complete collection of historical playback records from the persistent store.
func Get() (map[string]*SavedEpisode, error) {
	cached, expired, err := cacher.Get()
	if err != nil {
		return nil, err
	}
	if expired || cached == nil {
		return make(map[string]*SavedEpisode), nil
	}
	return cached, nil
}

// Save persists the playback progress of a specific episode to the history registry.
func Save(episode *source.Episode, percentage float64) error {
	saved, err := Get()
	if err != nil {
		return err
	}

	record := newSavedEpisode(episode)

	// Idempotency: Maintain the maximum observed playback percentage to prevent regressions on re-watch.
	if existing, exists := saved[record.encode()]; exists {
		if percentage < existing.WatchedPercentage {
			percentage = existing.WatchedPercentage
		}
	}
	record.WatchedPercentage = percentage

	saved[record.encode()] = record

	return cacher.Set(saved)
}

// Remove permanently deletes a specific playback record from the history registry.
func Remove(episode *SavedEpisode) error {
	saved, err := Get()
	if err != nil {
		return err
	}

	delete(saved, episode.encode())
	return cacher.Set(saved)
}
