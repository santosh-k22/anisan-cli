// Package inline provides the implementation for the application's non-interactive, programmable execution mode.
package inline

import (
	"fmt"
	"io"
	"os"

	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/log"
	"github.com/anisan-cli/anisan/source"
	"github.com/anisan-cli/anisan/where"
	"github.com/spf13/viper"
)

func Run(options *Options) error {
	fmt.Fprintf(os.Stderr, "DEBUG: Looking for sources in: %s\n", where.Sources())
	if options.Out == nil {
		options.Out = os.Stdout
	}

	// Step 1: Execute concurrent searches across all configured providers.
	var animes []*source.Anime
	for _, src := range options.Sources {
		m, err := src.Search(options.Query)
		if err != nil {
			return fmt.Errorf("search failed for %s: %w", src.Name(), err)
		}
		animes = append(animes, m...)
	}

	// Step 2: Apply anime selection logic if a picker is defined.
	var selected []*source.Anime
	if options.AnimePicker.IsPresent() {
		picker := options.AnimePicker.MustGet()
		if choice := picker(animes); choice != nil {
			selected = []*source.Anime{choice}
		}
	} else {
		selected = animes
	}

	if len(selected) == 0 {
		if options.Json {
			return writeJson(options.Out, []*source.Anime{}, options)
		}
		return nil // Nothing found
	}

	// Step 3: Populate metadata and retrieve episodes for the selected subset of anime.
	for _, anime := range selected {
		if err := prepareAnime(anime, options); err != nil {
			return err
		}
	}

	// Step 4: Dispatch the processed results to the configured output writer.
	if options.Json {
		return writeJson(options.Out, selected, options)
	}

	// Plain text output (only makes sense if episodes are filtered/single anime)
	// If multiple animes, we probably wouldn't be here in text mode usually,
	// but let's print episodes for all selected animes.
	for _, anime := range selected {
		for _, ep := range anime.Episodes {
			log.Info("Found " + ep.Name)
			if options.Videos && len(ep.Videos) > 0 {
				for _, v := range ep.Videos {
					fmt.Fprintln(options.Out, v.URL)
				}
			} else {
				fmt.Fprintln(options.Out, ep.URL)
			}
		}
	}

	return nil
}

func prepareAnime(anime *source.Anime, options *Options) error {
	// Anilist binding
	if options.IncludeAnilistAnime || viper.GetBool(key.MetadataFetchAnilist) {
		if err := anime.BindWithAnilist(); err != nil {
			// Don't fail hard on metadata fetch unless critical?
			// Inline mode usually expects data.
			log.Warnf("failed to bind anilist for %s: %v", anime.Name, err)
		}
	}

	if viper.GetBool(key.MetadataFetchAnilist) {
		_ = anime.PopulateMetadata(func(string) {})
	}

	// Episodes
	episodes, err := anime.Source.EpisodesOf(anime)
	if err != nil {
		return err
	}

	// Filter Episodes
	if options.EpisodesFilter.IsPresent() {
		filter := options.EpisodesFilter.MustGet()
		filtered, err := filter(episodes)
		if err != nil {
			return err
		}
		anime.Episodes = filtered
	} else {
		anime.Episodes = episodes
	}

	// Videos
	if options.Videos {
		for _, ep := range anime.Episodes {
			videos, err := anime.Source.VideosOf(ep)
			if err != nil {
				log.Warnf("failed to fetch videos for %s: %v", ep.Name, err)
				continue
			}
			ep.Videos = videos
		}
	}

	return nil
}

func writeJson(out io.Writer, animes []*source.Anime, options *Options) error {
	data, err := asJson(animes, options.Query, options.IncludeAnilistAnime)
	if err != nil {
		return err
	}
	_, err = out.Write(data)
	return err
}
