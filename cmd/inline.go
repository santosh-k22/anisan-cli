// Package cmd implements the command-line interface for anisan-cli.
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/anisan-cli/anisan/anilist"
	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/inline"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/mal"
	"github.com/anisan-cli/anisan/provider"
	"github.com/anisan-cli/anisan/query"
	"github.com/anisan-cli/anisan/source"

	"github.com/invopop/jsonschema"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(inlineCmd)

	inlineCmd.Flags().StringP("query", "q", "", "The search query to execute for anime discovery")
	inlineCmd.Flags().StringP("anime", "a", "", "Criteria for selecting specific anime from the search results")
	inlineCmd.Flags().StringP("episodes", "e", "", "Criteria for selecting specific episodes from the chosen anime")
	inlineCmd.Flags().BoolP("json", "j", false, "Format the command output as a JSON object")
	inlineCmd.Flags().BoolP("fetch-metadata", "f", false, "Fetch and include detailed anime metadata in the output")
	inlineCmd.Flags().BoolP("include-anilist-anime", "A", false, "Include Anilist record data in the structured output")
	inlineCmd.Flags().BoolP("include-mal-anime", "M", false, "Include MAL record data in the structured output")
	inlineCmd.Flags().BoolP("include-videos", "V", false, "Execute provider scraping to include video stream URLs for selected episodes")
	lo.Must0(viper.BindPFlag(key.TrackerFetchMetadata, inlineCmd.Flags().Lookup("fetch-metadata")))

	inlineCmd.Flags().StringP("output", "o", "", "Specify a file path to write the command output")

	inlineCmd.RegisterFlagCompletionFunc("query", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return query.SuggestMany(toComplete), cobra.ShellCompDirectiveNoFileComp
	})
}

// inlineCmd executes the application in non-interactive, scriptable inline mode.
var inlineCmd = &cobra.Command{
	Use:   "inline",
	Short: "Execute the application in non-interactive, scriptable inline mode",
	Long: `Initialize the application for automated execution and data extraction using inline mode.

Anime selectors:
  first - first anime in the list
  last - last anime in the list
  [number] - select anime by index (starting from 0)

Episode selectors:
  first - first episode in the list
  last - last episode in the list
  all - all episodes in the list
  [number] - select episode by index (starting from 0)
  [from]-[to] - select episodes by range
  @[substring]@ - select episodes by name substring

When using the json flag anime selector could be omitted. That way, it will select all animes`,

	Example: "https://github.com/anisan-cli/anisan/wiki/Inline-mode",
	PreRun: func(cmd *cobra.Command, args []string) {
		json, _ := cmd.Flags().GetBool("json")

		if !json && !cmd.Flags().Changed("anime") {
			handleErr(errors.New("the --anime flag is required when not outputting to JSON"))
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			sources []source.Source
			err     error
		)

		for _, name := range viper.GetStringSlice(key.DefaultSources) {
			if name == "" {
				handleErr(errors.New("source not set"))
			}

			p, ok := provider.Get(name)
			if !ok {
				handleErr(fmt.Errorf("source not found: %s", name))
			}

			src, err := p.CreateSource()
			handleErr(err)

			sources = append(sources, src)
		}

		query := lo.Must(cmd.Flags().GetString("query"))

		output := lo.Must(cmd.Flags().GetString("output"))
		var writer io.Writer
		if output != "" {
			writer, err = filesystem.API().Create(output)
			handleErr(err)
		} else {
			writer = os.Stdout
		}

		animeFlag := lo.Must(cmd.Flags().GetString("anime"))
		animePicker := mo.None[inline.AnimePicker]()
		if animeFlag != "" {
			fn, err := inline.ParseAnimePicker(lo.Must(cmd.Flags().GetString("anime")), query)
			handleErr(err)
			animePicker = mo.Some(fn)
		}

		episodeFlag := lo.Must(cmd.Flags().GetString("episodes"))
		episodesFilter := mo.None[inline.EpisodesFilter]()
		if episodeFlag != "" {
			fn, err := inline.ParseEpisodesFilter(episodeFlag)
			handleErr(err)
			episodesFilter = mo.Some(fn)
		}

		options := &inline.Options{
			Sources:             sources,
			Json:                lo.Must(cmd.Flags().GetBool("json")),
			Query:               query,
			IncludeAnilistAnime: lo.Must(cmd.Flags().GetBool("include-anilist-anime")),
			IncludeMalAnime:     lo.Must(cmd.Flags().GetBool("include-mal-anime")),
			AnimePicker:         animePicker,
			EpisodesFilter:      episodesFilter,
			Out:                 writer,
			Videos:              lo.Must(cmd.Flags().GetBool("include-videos")),
		}

		handleErr(inline.Run(options))
	},
}

func init() {
	inlineCmd.AddCommand(inlineAnilistCmd)
}

// inlineAnilistCmd manages Anilist record operations in inline mode.
var inlineAnilistCmd = &cobra.Command{
	Use:   "anilist",
	Short: "Manage Anilist record operations in inline mode",
}

func init() {
	inlineAnilistCmd.AddCommand(inlineAnilistSearchCmd)

	inlineAnilistSearchCmd.Flags().StringP("name", "n", "", "The anime title to search for on Anilist")
	inlineAnilistSearchCmd.Flags().IntP("id", "i", 0, "The specific Anilist ID to retrieve metadata for")

	inlineAnilistSearchCmd.MarkFlagsMutuallyExclusive("name", "id")
}

var inlineAnilistSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Perform an Anilist search by anime title and return the results",
	PreRun: func(cmd *cobra.Command, args []string) {
		if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("id") {
			handleErr(errors.New("name or id flag is required"))
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		animeName := lo.Must(cmd.Flags().GetString("name"))
		animeId := lo.Must(cmd.Flags().GetInt("id"))

		var toEncode any

		if animeName != "" {
			animes, err := anilist.SearchByName(animeName)
			handleErr(err)
			toEncode = animes
		} else {
			anime, err := anilist.GetByID(animeId)
			handleErr(err)
			toEncode = anime
		}

		handleErr(json.NewEncoder(os.Stdout).Encode(toEncode))
	},
}

func init() {
	inlineAnilistCmd.AddCommand(inlineAnilistGetCmd)

	inlineAnilistGetCmd.Flags().StringP("name", "n", "", "The local anime name to retrieve the mapped Anilist relation for")
	lo.Must0(inlineAnilistGetCmd.MarkFlagRequired("name"))
}

var inlineAnilistGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Retrieve the Anilist record currently associated with a specific local anime title",
	Run: func(cmd *cobra.Command, args []string) {
		name := lo.Must(cmd.Flags().GetString("name"))
		a := anilist.GetCachedRelation(name)

		if a == nil {
			anime, err := anilist.FindClosest(name)
			handleErr(err)
			a = anime
		}

		handleErr(json.NewEncoder(os.Stdout).Encode(a))
	},
}

func init() {
	inlineAnilistCmd.AddCommand(inlineAnilistBindCmd)

	inlineAnilistBindCmd.Flags().StringP("name", "n", "", "The local anime title to establish a mapping for")
	inlineAnilistBindCmd.Flags().IntP("id", "i", 0, "The Anilist ID to bind to the specified anime title")

	lo.Must0(inlineAnilistBindCmd.MarkFlagRequired("name"))
	lo.Must0(inlineAnilistBindCmd.MarkFlagRequired("id"))

	inlineAnilistBindCmd.MarkFlagsRequiredTogether("name", "id")
}

var inlineAnilistBindCmd = &cobra.Command{
	Use:   "set",
	Short: "Statically bind a local anime title to a specific Anilist record ID",
	Run: func(cmd *cobra.Command, args []string) {
		anilistAnime, err := anilist.GetByID(lo.Must(cmd.Flags().GetInt("id")))
		handleErr(err)

		animeName := lo.Must(cmd.Flags().GetString("name"))

		handleErr(anilist.SetRelation(animeName, anilistAnime))
	},
}

func init() {
	inlineCmd.AddCommand(inlineTrackerCmd)
}

// inlineTrackerCmd manages remote tracker record operations in inline mode.
var inlineTrackerCmd = &cobra.Command{
	Use:   "tracker",
	Short: "Manage remote media tracker operations in inline mode",
}

func init() {
	inlineTrackerCmd.AddCommand(inlineTrackerSearchCmd)

	inlineTrackerSearchCmd.Flags().StringP("name", "n", "", "The anime title to search on the active tracker")
	inlineTrackerSearchCmd.Flags().IntP("id", "i", 0, "The specific tracker record ID to retrieve metadata for")

	inlineTrackerSearchCmd.MarkFlagsMutuallyExclusive("name", "id")
}

// inlineTrackerSearchCmd performs an API search via the active tracking backend.
var inlineTrackerSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Perform a search via the active tracker by anime title and return the results",
	PreRun: func(cmd *cobra.Command, args []string) {
		if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("id") {
			handleErr(errors.New("name or id flag is required"))
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		animeName := lo.Must(cmd.Flags().GetString("name"))
		animeId := lo.Must(cmd.Flags().GetInt("id"))

		var toEncode any
		backend := viper.GetString(key.TrackerBackend)

		if animeName != "" {
			if backend == "mal" {
				animes, err := mal.SearchAnime(animeName)
				handleErr(err)
				toEncode = animes
			} else {
				animes, err := anilist.SearchByName(animeName)
				handleErr(err)
				toEncode = animes
			}
		} else {
			if backend == "mal" {
				anime, err := mal.GetByID(animeId)
				handleErr(err)
				toEncode = anime
			} else {
				anime, err := anilist.GetByID(animeId)
				handleErr(err)
				toEncode = anime
			}
		}

		handleErr(json.NewEncoder(os.Stdout).Encode(toEncode))
	},
}

func init() {
	inlineTrackerCmd.AddCommand(inlineTrackerGetCmd)

	inlineTrackerGetCmd.Flags().StringP("name", "n", "", "The local anime name to retrieve the mapped relation for")
	lo.Must0(inlineTrackerGetCmd.MarkFlagRequired("name"))
}

// inlineTrackerGetCmd retrieves mapped relations for local anime titles.
var inlineTrackerGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Retrieve the tracker record currently associated with a specific local anime title",
	Run: func(cmd *cobra.Command, args []string) {
		var toEncode any
		name := lo.Must(cmd.Flags().GetString("name"))
		backend := viper.GetString(key.TrackerBackend)

		if backend == "mal" {
			var a *mal.Anime
			a = mal.GetCachedRelation(name)
			if a == nil {
				animes, err := mal.SearchAnime(name)
				handleErr(err)
				if len(animes) > 0 {
					a = &animes[0]
				}
			}
			toEncode = a
		} else {
			var a *anilist.Anime
			a, err := anilist.FindClosest(name)
			if err != nil {
				handleErr(err)
			}
			toEncode = a
		}

		handleErr(json.NewEncoder(os.Stdout).Encode(toEncode))
	},
}

func init() {
	inlineTrackerCmd.AddCommand(inlineTrackerBindCmd)

	inlineTrackerBindCmd.Flags().StringP("name", "n", "", "The local anime title to establish a mapping for")
	inlineTrackerBindCmd.Flags().IntP("id", "i", 0, "The remote Tracker ID to bind to the specified anime title")

	lo.Must0(inlineTrackerBindCmd.MarkFlagRequired("name"))
	lo.Must0(inlineTrackerBindCmd.MarkFlagRequired("id"))

	inlineTrackerBindCmd.MarkFlagsRequiredTogether("name", "id")
}

// inlineTrackerBindCmd statically binds local anime titles to active tracker IDs.
var inlineTrackerBindCmd = &cobra.Command{
	Use:   "set",
	Short: "Statically bind a local anime title to a specific remote tracker ID",
	Run: func(cmd *cobra.Command, args []string) {
		backend := viper.GetString(key.TrackerBackend)
		animeName := lo.Must(cmd.Flags().GetString("name"))
		searchId := lo.Must(cmd.Flags().GetInt("id"))

		if backend == "mal" {
			malAnime, err := mal.GetByID(searchId)
			handleErr(err)
			handleErr(mal.SetRelation(animeName, malAnime))
		} else {
			anilistAnime, err := anilist.GetByID(searchId)
			handleErr(err)
			handleErr(anilist.SetRelation(animeName, anilistAnime))
		}
	},
}

func init() {
	inlineCmd.AddCommand(inlineSchemaCmd)

	inlineSchemaCmd.Flags().BoolP("anilist", "a", false, "Generate the JSON Schema for Anilist search result objects")
	inlineSchemaCmd.Flags().BoolP("mal", "m", false, "Generate the JSON Schema for MAL search result objects")
}

// inlineSchemaCmd generates JSON schemas for structured inline mode outputs.
var inlineSchemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Generate JSON schemas for structured inline mode outputs",
	Run: func(cmd *cobra.Command, args []string) {
		reflector := new(jsonschema.Reflector)
		reflector.Anonymous = true
		reflector.Namer = func(t reflect.Type) string {
			name := t.Name()
			switch strings.ToLower(name) {
			case "anime", "episode", "video", "date", "output":
				return filepath.Base(t.PkgPath()) + "." + name
			}

			return name
		}

		var schema *jsonschema.Schema

		switch {
		case lo.Must(cmd.Flags().GetBool("anilist")):
			schema = reflector.Reflect([]*anilist.Anime{})
		case lo.Must(cmd.Flags().GetBool("mal")):
			schema = reflector.Reflect([]*mal.Anime{})
		default:
			schema = reflector.Reflect(&inline.Output{})
		}

		handleErr(json.NewEncoder(os.Stdout).Encode(schema))
	},
}

func init() {
	inlineCmd.AddCommand(inlineMalCmd)
}

// inlineMalCmd manages MAL record operations in inline mode.
var inlineMalCmd = &cobra.Command{
	Use:   "mal",
	Short: "Manage MAL record operations in inline mode",
}

func init() {
	inlineMalCmd.AddCommand(inlineMalSearchCmd)

	inlineMalSearchCmd.Flags().StringP("name", "n", "", "The anime title to search for on MAL")
	inlineMalSearchCmd.Flags().IntP("id", "i", 0, "The specific MAL ID to retrieve metadata for")

	inlineMalSearchCmd.MarkFlagsMutuallyExclusive("name", "id")
}

// inlineMalSearchCmd performs a MAL search by anime title.
var inlineMalSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Perform a MAL search by anime title and return the results",
	PreRun: func(cmd *cobra.Command, args []string) {
		if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("id") {
			handleErr(errors.New("name or id flag is required"))
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		animeName := lo.Must(cmd.Flags().GetString("name"))
		animeId := lo.Must(cmd.Flags().GetInt("id"))

		var toEncode any

		if animeName != "" {
			animes, err := mal.SearchAnime(animeName)
			handleErr(err)
			toEncode = animes
		} else {
			anime, err := mal.GetByID(animeId)
			handleErr(err)
			toEncode = anime
		}

		handleErr(json.NewEncoder(os.Stdout).Encode(toEncode))
	},
}

func init() {
	inlineMalCmd.AddCommand(inlineMalGetCmd)

	inlineMalGetCmd.Flags().StringP("name", "n", "", "The local anime name to retrieve the mapped MAL relation for")
	lo.Must0(inlineMalGetCmd.MarkFlagRequired("name"))
}

// inlineMalGetCmd retrieves mapped MAL relations for local anime titles.
var inlineMalGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Retrieve the MAL record currently associated with a specific local anime title",
	Run: func(cmd *cobra.Command, args []string) {
		var (
			a *mal.Anime
		)

		name := lo.Must(cmd.Flags().GetString("name"))
		a = mal.GetCachedRelation(name)

		if a == nil {
			animes, err := mal.SearchAnime(name)
			handleErr(err)
			if len(animes) > 0 {
				a = &animes[0]
			}
		}

		handleErr(json.NewEncoder(os.Stdout).Encode(a))
	},
}

func init() {
	inlineMalCmd.AddCommand(inlineMalBindCmd)

	inlineMalBindCmd.Flags().StringP("name", "n", "", "The local anime title to establish a mapping for")
	inlineMalBindCmd.Flags().IntP("id", "i", 0, "The MAL ID to bind to the specified anime title")

	lo.Must0(inlineMalBindCmd.MarkFlagRequired("name"))
	lo.Must0(inlineMalBindCmd.MarkFlagRequired("id"))

	inlineMalBindCmd.MarkFlagsRequiredTogether("name", "id")
}

// inlineMalBindCmd statically binds local anime titles to MAL record IDs.
var inlineMalBindCmd = &cobra.Command{
	Use:   "set",
	Short: "Statically bind a local anime title to a specific MAL record ID",
	Run: func(cmd *cobra.Command, args []string) {
		malAnime, err := mal.GetByID(lo.Must(cmd.Flags().GetInt("id")))
		handleErr(err)

		animeName := lo.Must(cmd.Flags().GetString("name"))

		handleErr(mal.SetRelation(animeName, malAnime))
	},
}
