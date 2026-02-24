// Package cmd implements the command-line interface for anisan-cli.
package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/anisan-cli/anisan/color"
	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/util"

	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/icon"
	"github.com/anisan-cli/anisan/provider"
	"github.com/anisan-cli/anisan/style"
	"github.com/anisan-cli/anisan/where"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sourcesCmd)
}

// sourcesCmd provides a parent command for managing scraping providers.
var sourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "Manage built-in and custom scraping providers",
}

func init() {
	sourcesCmd.AddCommand(sourcesListCmd)

	sourcesListCmd.Flags().BoolP("raw", "r", false, "Suppress header and metadata descriptions in the output")
	sourcesListCmd.Flags().BoolP("custom", "c", false, "Display only user-installed custom Lua sources")
	sourcesListCmd.Flags().BoolP("builtin", "b", false, "Display only pre-compiled built-in sources")

	sourcesListCmd.MarkFlagsMutuallyExclusive("custom", "builtin")
	sourcesListCmd.SetOut(os.Stdout)
}

// sourcesListCmd displays a summary of all registered scraping providers.
var sourcesListCmd = &cobra.Command{
	Use:   "list",
	Short: "Display a collection of all registered scraping providers",
	Run: func(cmd *cobra.Command, args []string) {
		printHeader := !lo.Must(cmd.Flags().GetBool("raw"))
		headerStyle := style.New().Foreground(color.HiBlue).Bold(true).Render
		h := func(s string) {
			if printHeader {
				cmd.Println(headerStyle(s))
			}
		}

		printBuiltin := func() {
			h("Builtin:")
			for _, p := range provider.Builtins() {
				cmd.Println(p.Name)
			}
		}

		printCustom := func() {
			h("Custom:")
			for _, p := range provider.Customs() {
				cmd.Println(p.Name)
			}
		}

		switch {
		case lo.Must(cmd.Flags().GetBool("builtin")):
			printBuiltin()
		case lo.Must(cmd.Flags().GetBool("custom")):
			printCustom()
		default:
			printBuiltin()
			if printHeader {
				cmd.Println()
			}
			printCustom()
		}
	},
}

func init() {
	sourcesCmd.AddCommand(sourcesRemoveCmd)

	sourcesRemoveCmd.Flags().StringArrayP("name", "n", []string{}, "Specify the name of the custom source(s) to uninstall")
	lo.Must0(sourcesRemoveCmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		sources, err := filesystem.API().ReadDir(where.Sources())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		return lo.FilterMap(sources, func(item os.FileInfo, _ int) (string, bool) {
			name := item.Name()
			if !strings.HasSuffix(name, provider.CustomProviderExtension) {
				return "", false
			}

			return util.FileStem(filepath.Base(name)), true
		}), cobra.ShellCompDirectiveNoFileComp
	}))
}

// sourcesRemoveCmd facilitates the uninstallation of custom Lua sources.
var sourcesRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Permanently uninstall specified custom Lua sources from the system",
	Run: func(cmd *cobra.Command, args []string) {
		for _, name := range lo.Must(cmd.Flags().GetStringArray("name")) {
			path := filepath.Join(where.Sources(), name+provider.CustomProviderExtension)
			handleErr(filesystem.API().Remove(path))
			fmt.Printf("%s successfully removed %s\n", icon.Get(icon.Success), style.Fg(color.Yellow)(name))
		}
	},
}

func init() {
	sourcesCmd.AddCommand(sourcesInstallCmd)
}

// sourcesInstallCmd facilitates the discovery and installation of community scrapers.
var sourcesInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Browse and install community-contributed scrapers",
	Long: `Initialize the interactive installation wizard for community scrapers from the official repository.
https://github.com/anisan-cli/scrapers`,
	Run: func(cmd *cobra.Command, args []string) {
		// handleErr(tui.Run(&tui.Options{Install: true}))
		fmt.Println("Community scraper installation is currently under development for a future release.")
	},
}

func init() {
	sourcesCmd.AddCommand(sourcesGenCmd)

	sourcesGenCmd.Flags().StringP("name", "n", "", "The display name of the new scraping provider")
	sourcesGenCmd.Flags().StringP("url", "u", "", "The base URL of the target website to be scraped")

	lo.Must0(sourcesGenCmd.MarkFlagRequired("name"))
	lo.Must0(sourcesGenCmd.MarkFlagRequired("url"))
}

// sourcesGenCmd scaffolds a boilerplate Lua provider script.
var sourcesGenCmd = &cobra.Command{
	Use:   "gen",
	Short: "Scaffold a new Lua provider script using a predefined template",
	Long:  `Generate a boilerplate Lua provider script with core functions and metadata.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.SetOut(os.Stdout)

		var author string
		usr, err := user.Current()
		if err == nil {
			author = usr.Username
		} else {
			author = "Anonymous"
		}

		s := struct {
			Name            string
			URL             string
			SearchAnimesFn  string
			AnimeEpisodesFn string
			Author          string
		}{
			Name:            lo.Must(cmd.Flags().GetString("name")),
			URL:             lo.Must(cmd.Flags().GetString("url")),
			SearchAnimesFn:  constant.SearchAnimesFn,
			AnimeEpisodesFn: constant.AnimeEpisodesFn,
			Author:          author,
		}

		funcMap := template.FuncMap{
			"repeat": strings.Repeat,
			"plus":   func(a, b int) int { return a + b },
			"max":    util.Max[int],
		}

		tmpl, err := template.New("source").Funcs(funcMap).Parse(constant.SourceTemplate)
		handleErr(err)

		target := filepath.Join(where.Sources(), util.SanitizeFilename(s.Name)+".lua")
		f, err := filesystem.API().Create(target)
		handleErr(err)

		defer util.Ignore(f.Close)

		err = tmpl.Execute(f, s)
		handleErr(err)

		cmd.Println(target)
	},
}
