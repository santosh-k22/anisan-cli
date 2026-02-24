// Package cmd implements the command-line interface for anisan-cli.
package cmd

import (
	"os"
	"strings"

	"github.com/anisan-cli/anisan/color"
	"github.com/anisan-cli/anisan/config"
	"github.com/anisan-cli/anisan/constant"
	"github.com/anisan-cli/anisan/style"
	"github.com/anisan-cli/anisan/where"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.Flags().BoolP("set-only", "s", false, "Display only environment variables that are currently defined")
	envCmd.Flags().BoolP("unset-only", "u", false, "Display only environment variables that are currently undefined")

	envCmd.MarkFlagsMutuallyExclusive("set-only", "unset-only")
}

// envCmd displays the current process values for all supported environment variables.
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Display the collection of supported environment variables",
	Long:  `Display the collection of supported environment variables and their current process values.`,
	Run: func(cmd *cobra.Command, args []string) {
		setOnly := lo.Must(cmd.Flags().GetBool("set-only"))
		unsetOnly := lo.Must(cmd.Flags().GetBool("unset-only"))

		config.EnvExposed = append(config.EnvExposed, where.EnvConfigPath)
		slices.Sort(config.EnvExposed)
		for _, env := range config.EnvExposed {
			if env != where.EnvConfigPath {
				env = strings.ToUpper(constant.Anisan + "_" + config.EnvKeyReplacer.Replace(env))
			}
			value := os.Getenv(env)
			present := value != ""

			if setOnly || unsetOnly {
				if !present && setOnly {
					continue
				}

				if present && unsetOnly {
					continue
				}
			}

			cmd.Print(style.New().Bold(true).Foreground(color.Purple).Render(env))
			cmd.Print("=")

			if present {
				cmd.Println(style.Fg(color.Green)(value))
			} else {
				cmd.Println(style.Fg(color.Red)("unset"))
			}
		}
	},
}
