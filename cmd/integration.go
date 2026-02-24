// Package cmd implements the command-line interface for anisan-cli.
package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/anisan-cli/anisan/icon"
	"github.com/anisan-cli/anisan/integration/anilist"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/log"
	"github.com/anisan-cli/anisan/open"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(integrationCmd)
	integrationCmd.AddCommand(integrationAnilistCmd)
	integrationAnilistCmd.Flags().BoolP("disable", "d", false, "Statically disable the Anilist service integration")
}

// integrationCmd manages high-level integrations with external tracking services.
var integrationCmd = &cobra.Command{
	Use:   "integration",
	Short: "Manage high-level integrations with external tracking services",
	Long:  `Configure and manage integrations with external platforms like Anilist.`,
}

// integrationAnilistCmd configures and manages the Anilist service integration.
var integrationAnilistCmd = &cobra.Command{
	Use:   "anilist",
	Short: "Configure the Anilist service integration",
	Long: `Initialize or modify the Anilist service integration, including OAuth credentials and synchronization settings.
See https://github.com/anisan-cli/anisan/wiki/Anilist-Integration for more information`,
	Run: func(cmd *cobra.Command, args []string) {
		if lo.Must(cmd.Flags().GetBool("disable")) {
			viper.Set(key.AnilistEnable, false)
			viper.Set(key.AnilistCode, "")
			viper.Set(key.AnilistSecret, "")
			viper.Set(key.AnilistID, "")
			log.Info("Anilist integration disabled")
			handleErr(viper.WriteConfig())
		}

		if !viper.GetBool(key.AnilistEnable) {
			confirm := survey.Confirm{
				Message: "Anilist is disabled. Enable?",
				Default: false,
			}
			var response bool
			err := survey.AskOne(&confirm, &response)
			handleErr(err)

			if !response {
				return
			}

			viper.Set(key.AnilistEnable, response)
			err = viper.WriteConfig()
			if err != nil {
				switch err.(type) {
				case viper.ConfigFileNotFoundError:
					err = viper.SafeWriteConfig()
					handleErr(err)
				default:
					handleErr(err)
					log.Error(err)
				}
			}
		}

		if viper.GetString(key.AnilistID) == "" {
			input := survey.Input{
				Message: "Anilist client ID is not set. Please enter it:",
				Help:    "",
			}
			var response string
			err := survey.AskOne(&input, &response)
			handleErr(err)

			if response == "" {
				return
			}

			viper.Set(key.AnilistID, response)
			err = viper.WriteConfig()
			handleErr(err)
		}

		if viper.GetString(key.AnilistSecret) == "" {
			input := survey.Input{
				Message: "Anilist client secret is not set. Please enter it:",
				Help:    "",
			}
			var response string
			err := survey.AskOne(&input, &response)
			handleErr(err)

			if response == "" {
				return
			}

			viper.Set(key.AnilistSecret, response)
			err = viper.WriteConfig()
			handleErr(err)
		}

		if viper.GetString(key.AnilistCode) == "" {
			authURL := anilist.New().AuthURL()
			confirmOpenInBrowser := survey.Confirm{
				Message: "Open browser to authenticate with Anilist?",
				Default: false,
			}

			var openInBrowser bool
			err := survey.AskOne(&confirmOpenInBrowser, &openInBrowser)
			if err == nil && openInBrowser {
				err = open.Start(authURL)
			}

			if err != nil || !openInBrowser {
				fmt.Println("Please open the following URL in your browser:")
				fmt.Println(authURL)
			}

			input := survey.Input{
				Message: "Anilist code is not set. Please copy it from the link and paste in here:",
				Help:    "",
			}

			var response string
			err = survey.AskOne(&input, &response)
			handleErr(err)

			if response == "" {
				return
			}

			viper.Set(key.AnilistCode, response)
			err = viper.WriteConfig()
			handleErr(err)
		}

		fmt.Printf("%s Anilist integration was set up\n", icon.Get(icon.Success))
	},
}
