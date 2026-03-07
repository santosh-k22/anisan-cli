package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/anisan-cli/anisan/anilist"
	"github.com/anisan-cli/anisan/internal/tracker"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/log"
	"github.com/anisan-cli/anisan/mal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(markCmd)
	markCmd.Flags().StringP("query", "q", "", "Anime title to search for")
	markCmd.Flags().IntP("episode", "e", 0, "Episode number to mark as watched")
	_ = markCmd.MarkFlagRequired("query")
	_ = markCmd.MarkFlagRequired("episode")
}

var markCmd = &cobra.Command{
	Use:   "mark",
	Short: "Headless progress synchronization",
	Long:  "Automatically searches and updates progress using your active tracker (AniList or MAL) defined in config.toml.",
	RunE: func(cmd *cobra.Command, args []string) error {
		query, _ := cmd.Flags().GetString("query")
		episodeIdx, _ := cmd.Flags().GetInt("episode")

		activeTracker := tracker.InitializeTracker()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Pre-flight auth check
		if err := activeTracker.CheckAuth(ctx); err != nil {
			return err
		}

		backend := viper.GetString(key.TrackerBackend)
		var mediaID, totalEps int

		log.Infof("Resolving %q via %s...", query, backend)

		// Resolve via the correct metadata provider
		if backend == "mal" {
			anime, err := mal.FindClosest(query)
			if err != nil {
				return fmt.Errorf("failed to locate anime on MAL: %w", err)
			}
			mediaID = anime.ID
			totalEps = anime.NumEpisodes
			log.Infof("Found %s (MAL ID: %d)", anime.Title, mediaID)
		} else {
			anime, err := anilist.FindClosest(query)
			if err != nil {
				return fmt.Errorf("failed to locate anime on AniList: %w", err)
			}
			mediaID = anime.ID
			totalEps = 0 // Default to 0; AniList backend will ignore completion status if not provided
			log.Infof("Found %s (AniList ID: %d)", anime.Name(), mediaID)
		}

		// Perform the unified sync
		err := activeTracker.UpdateEpisodeProgress(ctx, mediaID, episodeIdx, totalEps)
		if err != nil {
			if err.Error() == "sync_queued" {
				fmt.Println("Network unavailable. Sync queued for background retry.")
				return nil
			}
			return err
		}

		fmt.Printf("Successfully marked episode %d on %s!\n", episodeIdx, backend)
		return nil
	},
}
