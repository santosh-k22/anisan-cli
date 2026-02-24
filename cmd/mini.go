// Package cmd implements the command-line interface for anisan-cli.
package cmd

import (
	"github.com/anisan-cli/anisan/mini"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(miniCmd)

	miniCmd.Flags().BoolP("continue", "c", false, "Resume playback from the most recent history entry")
}

// miniCmd launches the application in a lightweight, minimalist terminal interface.
var miniCmd = &cobra.Command{
	Use:   "mini",
	Short: "Launch the application in a lightweight, minimalist terminal interface",
	Long:  `Initialize a streamlined, minimalist terminal UI for anime selection and playback.`,
	Run: func(cmd *cobra.Command, args []string) {
		options := mini.Options{
			Continue: lo.Must(cmd.Flags().GetBool("continue")),
		}
		err := mini.Run(&options)

		if err != nil && err.Error() != "interrupt" {
			handleErr(err)
		}
	},
}
