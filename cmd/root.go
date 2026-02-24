// Package cmd implements the command-line interface for anisan-cli.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/anisan-cli/anisan/color"
	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/icon"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/log"
	"github.com/anisan-cli/anisan/provider"
	"github.com/anisan-cli/anisan/style"
	"github.com/anisan-cli/anisan/tui"
	"github.com/anisan-cli/anisan/util"
	"github.com/anisan-cli/anisan/version"
	"github.com/anisan-cli/anisan/where"
	cc "github.com/ivanpirog/coloredcobra"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "Print the application version")

	rootCmd.PersistentFlags().StringP("icons", "I", "", "Set the visual icon variant (e.g., nerd, emoji, square)")
	lo.Must0(rootCmd.RegisterFlagCompletionFunc("icons", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return icon.AvailableVariants(), cobra.ShellCompDirectiveDefault
	}))
	lo.Must0(viper.BindPFlag(key.IconsVariant, rootCmd.PersistentFlags().Lookup("icons")))

	rootCmd.PersistentFlags().BoolP("write-history", "H", true, "Persist playback progress to the localized watch history")
	lo.Must0(viper.BindPFlag(key.HistorySaveOnRead, rootCmd.PersistentFlags().Lookup("write-history")))

	rootCmd.PersistentFlags().StringSliceP("source", "S", []string{}, "Specify the default search sources to prioritize")
	lo.Must0(rootCmd.RegisterFlagCompletionFunc("source", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var sources []string

		for _, p := range provider.Builtins() {
			sources = append(sources, p.Name)
		}

		for _, p := range provider.Customs() {
			sources = append(sources, p.Name)
		}

		return sources, cobra.ShellCompDirectiveDefault
	}))
	lo.Must0(viper.BindPFlag(key.DefaultSources, rootCmd.PersistentFlags().Lookup("source")))

	rootCmd.Flags().BoolP("continue", "c", false, "Resume playback from the most recent history entry")

	helpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		helpFunc(cmd, args)
		version.Notify()
	})

	// Initialize cleanup of localized temporary files on application startup.
	go func() {
		_ = util.Delete(where.Temp())
	}()
}

// rootCmd defines the entry point for the anisan-cli application.
var rootCmd = &cobra.Command{
	Use:   constant.Anisan,
	Short: "A minimalist command-line interface for anime discovery and playback",
	Long: constant.AsciiArtLogo + "\n" +
		style.New().Italic(true).Foreground(color.HiRed).Render("    - A minimalist command-line interface for anime discovery and playback"),
	Run: func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("version") {
			versionCmd.Run(versionCmd, args)
			return
		}

		CheckDependencies()

		options := tui.Options{
			Continue: lo.Must(cmd.Flags().GetBool("continue")),
		}
		handleErr(tui.Run(&options))
	},
}

// Execute initializes child command routing and processes the CLI entry point.
func Execute() {
	if viper.GetBool(key.CliColored) {
		cc.Init(&cc.Config{
			RootCmd:       rootCmd,
			Headings:      cc.HiCyan + cc.Bold + cc.Underline,
			Commands:      cc.HiYellow + cc.Bold,
			Example:       cc.Italic,
			ExecName:      cc.Bold,
			Flags:         cc.Bold,
			FlagsDataType: cc.Italic + cc.HiBlue,
		})
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func handleErr(err error) {
	if err != nil {
		log.Error(err)
		_, _ = fmt.Fprintf(os.Stderr, "%s %s\n", icon.Get(icon.Fail), strings.Trim(err.Error(), " \n"))
		os.Exit(1)
	}
}
