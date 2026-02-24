// Package mal provides a client for the MyAnimeList REST API.
package mal

import (
	"path/filepath"
	"time"

	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/where"
	"github.com/metafates/gache"
	"github.com/samber/mo"
)

// cacheData defines the structured format for persisting cached MyAnimeList records to disk.
type cacheData[K comparable, T any] struct {
	Animes map[K]T `json:"animes"`
}

// cacher provides a generic, thread-safe wrapper for high-level caching operations.
type cacher[K comparable, T any] struct {
	internal   *gache.Cache[*cacheData[K, T]]
	keyWrapper func(K) K
}

// Get retrieves a value from the cache associated with the specified key.
func (c *cacher[K, T]) Get(key K) mo.Option[T] {
	data, expired, err := c.internal.Get()
	if err != nil || expired || data == nil {
		return mo.None[T]()
	}

	animes, ok := data.Animes[c.keyWrapper(key)]
	if ok {
		return mo.Some(animes)
	}

	return mo.None[T]()
}

// Set persists a key-value pair to the cache.
func (c *cacher[K, T]) Set(key K, t T) error {
	data, expired, err := c.internal.Get()
	if err != nil {
		return err
	}

	if !expired && data != nil {
		data.Animes[c.keyWrapper(key)] = t
		return c.internal.Set(data)
	} else {
		internal := &cacheData[K, T]{Animes: make(map[K]T)}
		internal.Animes[c.keyWrapper(key)] = t
		return c.internal.Set(internal)
	}
}

// Delete removes the entry associated with the specified key from the cache.
func (c *cacher[K, T]) Delete(key K) error {
	data, expired, err := c.internal.Get()
	if err != nil {
		return err
	}

	if !expired {
		delete(data.Animes, c.keyWrapper(key))
		return c.internal.Set(data)
	}

	return nil
}

// relationCacher provides persistence for anime title-to-ID mappings.
var relationCacher = &cacher[string, int]{
	internal: gache.New[*cacheData[string, int]](
		&gache.Options{
			Path:       filepath.Join(where.Config(), "mal.json"), // Using mal.json in config dir
			FileSystem: &filesystem.GacheFs{},
		},
	),
	keyWrapper: normalizedName,
}

// idCacher provides local persistence for comprehensive anime metadata lookups.
var idCacher = &cacher[int, *Anime]{
	internal: gache.New[*cacheData[int, *Anime]](
		&gache.Options{
			Path:       filepath.Join(where.Cache(), "mal_id_cache.json"),
			Lifetime:   time.Hour * 24 * 2,
			FileSystem: &filesystem.GacheFs{},
		},
	),
	keyWrapper: func(id int) int { return id },
}
