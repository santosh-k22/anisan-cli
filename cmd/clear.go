// Package cmd implements the command-line interface for anisan-cli.
package cmd

import (
	"fmt"

	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/icon"
	"github.com/anisan-cli/anisan/util"
	"github.com/anisan-cli/anisan/where"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/spf13/cobra"
)

// clearTarget defines a filesystem resource eligible for automated cleanup.
type clearTarget struct {
	name     string
	argLong  string
	argShort mo.Option[string]
	location func() string
}

// clearTargets registry of all application artifacts that can be selectively cleared.
var clearTargets = []clearTarget{
	{"cache directory", "cache", mo.Some("c"), where.Cache},
	{"history file", "history", mo.Some("s"), where.History},
	{"anilist binds", "anilist", mo.Some("a"), where.AnilistBinds},
	{"queries history", "queries", mo.Some("q"), where.Queries},
}

func init() {
	rootCmd.AddCommand(clearCmd)

	for _, target := range clearTargets {
		help := fmt.Sprintf("clear %s", target.name)
		if target.argShort.IsPresent() {
			clearCmd.Flags().BoolP(target.argLong, target.argShort.MustGet(), false, help)
		} else {
			clearCmd.Flags().Bool(target.argLong, false, help)
		}
	}
}

// clearCmd manages the cleanup of temporary and cached application artifacts.
var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear temporary and cached application artifacts",
	Run: func(cmd *cobra.Command, args []string) {
		var anyCleared bool

		doClear := func(what string) bool {
			return lo.Must(cmd.Flags().GetBool(what))
		}

		for _, target := range clearTargets {
			if doClear(target.argLong) {
				anyCleared = true
				e := util.PrintErasable(fmt.Sprintf("%s Clearing %s...", icon.Get(icon.Progress), util.Capitalize(target.name)))
				_ = util.Delete(target.location())
				e()
				fmt.Printf("%s %s cleared\n", icon.Get(icon.Success), util.Capitalize(target.name))
				handleErr(filesystem.API().RemoveAll(target.location()))
			}
		}

		if !anyCleared {
			handleErr(cmd.Help())
		}
	},
}
