// Package cmd implements the command-line interface for anisan-cli.
package cmd

import (
	"os"

	"github.com/anisan-cli/anisan/color"
	"github.com/anisan-cli/anisan/style"
	"github.com/anisan-cli/anisan/where"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/spf13/cobra"
)

// whereTarget encapsulates a localized filesystem resource and its CLI representation.
type whereTarget struct {
	name     string
	where    func() string
	argLong  string
	argShort mo.Option[string]
	hidden   bool
}

// wherePaths registry of all application resources with resolvable filesystem paths.
var wherePaths = []*whereTarget{
	{"Config", where.Config, "config", mo.Some("c"), false},
	{"Sources", where.Sources, "sources", mo.Some("s"), false},
	{"Logs", where.Logs, "logs", mo.Some("l"), false},
	{"Cache", where.Cache, "cache", mo.None[string](), true},
	{"Temp", where.Temp, "temp", mo.None[string](), true},
	{"History", where.History, "history", mo.None[string](), true},
}

func init() {
	rootCmd.AddCommand(whereCmd)

	for _, n := range wherePaths {
		if n.argShort.IsPresent() {
			whereCmd.Flags().BoolP(n.argLong, n.argShort.MustGet(), false, n.name+" path")
		} else {
			whereCmd.Flags().Bool(n.argLong, false, n.name+" path")
		}

		if n.hidden {
			lo.Must0(whereCmd.Flags().MarkHidden(n.argLong))
		}

	}

	whereCmd.MarkFlagsMutuallyExclusive(lo.Map(wherePaths, func(t *whereTarget, _ int) string {
		return t.argLong
	})...)

	whereCmd.SetOut(os.Stdout)
}

// whereCmd displays localized filesystem paths for application resources.
var whereCmd = &cobra.Command{
	Use:   "where",
	Short: "Display the localized filesystem paths for application-specific resources",
	Run: func(cmd *cobra.Command, args []string) {
		headerStyle := style.New().Bold(true).Foreground(color.HiPurple).Render

		for _, n := range wherePaths {
			if lo.Must(cmd.Flags().GetBool(n.argLong)) {
				cmd.Println(n.where())
				return
			}
		}

		wherePaths = lo.Filter(wherePaths, func(t *whereTarget, _ int) bool {
			return !t.hidden
		})

		for i, n := range wherePaths {
			if n.hidden {
				continue
			}

			cmd.Printf("%s %s\n", headerStyle(n.name+"?"), style.Fg(color.Yellow)("--"+n.argLong))
			cmd.Println(n.where())

			if i < len(wherePaths)-1 {
				cmd.Println()
			}
		}
	},
}
