// Package mal provides a client for the MyAnimeList REST API.
package mal

import (
	"fmt"
	"strings"

	"github.com/anisan-cli/anisan/log"
	"github.com/anisan-cli/anisan/util"
	levenshtein "github.com/ka-weihe/fast-levenshtein"
	"github.com/samber/lo"
)

// normalizedName returns a lowercased, trimmed string for consistent comparison.
func normalizedName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// SetRelation persists a mapping between an anime name and its MyAnimeList identifier.
func SetRelation(name string, to *Anime) error {
	err := relationCacher.Set(name, to.ID)
	if err != nil {
		return err
	}

	if id := idCacher.Get(to.ID); id.IsAbsent() {
		return idCacher.Set(to.ID, to)
	}

	return nil
}

// FindClosest returns the closest anime to the given name.
// It will levenshtein compare the given name with all the anime names in the cache.
func FindClosest(name string) (*Anime, error) {
	name = normalizedName(name)
	return findClosest(name, name, 0, 3)
}

// findClosest returns the closest anime to the given name.
// It will levenshtein compare the given name with all the anime names in the cache.
func findClosest(name, originalName string, try, limit int) (*Anime, error) {
	if try >= limit {
		err := fmt.Errorf("no results found on MAL for anime %s", name)
		log.Error(err)
		_ = relationCacher.Set(originalName, -1)
		return nil, err
	}

	id := relationCacher.Get(name)
	if id.IsPresent() {
		if id.MustGet() == -1 {
			return nil, fmt.Errorf("no results found on MAL for anime %s", name)
		}

		if anime, ok := idCacher.Get(id.MustGet()).Get(); ok {
			if try > 0 {
				_ = relationCacher.Set(originalName, anime.ID)
			}
			return anime, nil
		}
	}

	// search for anime on mal
	animes, err := SearchAnime(name)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if id.IsPresent() {
		found, ok := lo.Find(animes, func(item Anime) bool { // SearchAnime returns []Anime
			return item.ID == id.MustGet()
		})

		if ok {
			return &found, nil
		}

		// there should be a anime with the id in the cache, but it wasn't found
		// this means that the anime was deleted from mal
		// remove the id from the cache
		_ = relationCacher.Delete(name)
		log.Infof("Anime with id %d was deleted from MAL", id.MustGet())
	}

	if len(animes) == 0 {
		// try again with a different name
		words := strings.Split(name, " ")
		if len(words) <= 2 {
			// API rate limit threshold reached; aborting further traversal to prevent escalation.
			return findClosest(name, originalName, limit, limit)
		}

		// one word less
		alternateName := strings.Join(words[:util.Max(len(words)-1, 1)], " ")
		log.Infof(`No results found on MAL for anime "%s", trying "%s"`, name, alternateName)
		return findClosest(alternateName, originalName, try+1, limit)
	}

	// find the closest match
	closest := lo.MinBy(animes, func(a, b Anime) bool {
		return levenshtein.Distance(
			name,
			normalizedName(a.Title),
		) < levenshtein.Distance(
			name,
			normalizedName(b.Title),
		)
	})

	log.Info("Found closest match: " + closest.Title)

	save := func(n string) {
		if id := relationCacher.Get(n); id.IsAbsent() {
			_ = relationCacher.Set(n, closest.ID)
		}
	}

	save(name)
	save(originalName)

	_ = idCacher.Set(closest.ID, &closest)
	return &closest, nil
}
