// Package cmd implements the command-line interface for anisan-cli.
package cmd

import (
	"github.com/anisan-cli/anisan/mal"
	"github.com/spf13/cobra"
)

const malSuccessHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authentication Successful</title>
    <style>
        body { margin: 0; padding: 0; background-color: #0f0f11; color: #ffffff; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; text-align: center; }
        .container { animation: fadeIn 0.8s ease-out; }
        h1 { font-size: 24px; font-weight: 500; margin-bottom: 8px; letter-spacing: -0.5px; }
        p { font-size: 15px; color: #88888b; font-weight: 400; }
        @keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
    </style>
</head>
<body>
    <div class="container">
        <h1>Authentication Successful</h1>
        <p>You may safely close this tab and return to the terminal.</p>
    </div>
</body>
</html>`

const malErrorHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authentication Failed</title>
    <style>
        body { margin: 0; padding: 0; background-color: #0f0f11; color: #ffffff; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; text-align: center; }
        .container { animation: fadeIn 0.8s ease-out; }
        h1 { font-size: 24px; font-weight: 500; margin-bottom: 8px; color: #ff5555; }
        p { font-size: 15px; color: #88888b; }
        @keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
    </style>
</head>
<body>
    <div class="container">
        <h1>Authentication Failed</h1>
        <p>%s</p>
    </div>
</body>
</html>`

func init() {
	rootCmd.AddCommand(malCmd)
	malCmd.AddCommand(malAuthCmd)
}

// malCmd manages high-level integration settings and synchronization for MyAnimeList.
var malCmd = &cobra.Command{
	Use:   "mal",
	Short: "Manage MyAnimeList service integration and synchronization",
}

// malAuthCmd initiates the OAuth2 PKCE authentication flow for MyAnimeList.
var malAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with the MyAnimeList service via OAuth2 PKCE",
	Long: `Initialize the OAuth2 PKCE authentication flow for MyAnimeList.
This command launches a local callback server and opens the system browser for secure authorization.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return mal.Login()
	},
}
