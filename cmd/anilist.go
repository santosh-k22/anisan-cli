// Package cmd implements the command-line interface for anisan-cli.
package cmd

import (
	"fmt"

	"github.com/anisan-cli/anisan/integration/anilist"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(anilistCmd)
	anilistCmd.AddCommand(authCmd)
}

// anilistCmd manages high-level integration settings and synchronization for the Anilist service.
var anilistCmd = &cobra.Command{
	Use:   "anilist",
	Short: "Manage Anilist service integration and synchronization settings",
}

// authCmd initiates the OAuth2 authentication flow for the Anilist service.
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with the Anilist service via OAuth",
	Long:  "Open your browser to securely log in to Anilist and save the OAuth token to the system keyring.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := anilist.AuthenticateWithBrowser(); err != nil {
			return err
		}
		fmt.Println("Authentication token successfully persisted to the system keyring.")
		return nil
	},
}
