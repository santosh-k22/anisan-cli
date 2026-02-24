// Package cmd implements the command-line interface for anisan-cli.
package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/anisan-cli/anisan/log"
	"github.com/anisan-cli/anisan/mal"
	"github.com/anisan-cli/anisan/open"
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
		// Step 1: Generate a cryptographically secure PKCE code verifier.
		verifier, err := mal.GenerateCodeVerifier()
		if err != nil {
			return err
		}

		// Channel to receive the code
		codeCh := make(chan string)
		errCh := make(chan error)

		// Step 2: Initialize a temporary local HTTP server to handle the OAuth2 callback.
		mux := http.NewServeMux()
		mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			w.Header().Set("Content-Type", "text/html")
			if code == "" {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, malErrorHTML, "No code found in redirect URL")
				return
			}
			codeCh <- code
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, malSuccessHTML)
		})

		server := &http.Server{Addr: ":8080", Handler: mux}

		go func() {
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				errCh <- err
			}
		}()

		// Step 3: Launch the system browser to the MyAnimeList authorization endpoint.
		// We pass empty string for clientID to use default
		authURL := mal.GetAuthURL(verifier, "")

		fmt.Println("Opening browser to:", authURL)
		if err := open.Start(authURL); err != nil {
			log.Warn("Failed to open browser: " + err.Error())
		}

		// Step 4: Await the authorization code from the redirect callback.
		log.Info("Waiting for callback on port 8080...")

		var code string
		select {
		case code = <-codeCh:
		case err := <-errCh:
			return fmt.Errorf("callback server error: %w", err)
		case <-time.After(2 * time.Minute):
			return fmt.Errorf("authentication timed out")
		}

		// Terminate the local callback server using a graceful shutdown sequence.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)

		// Step 5: Exchange the authorization code for a secure access token.
		token, err := mal.ExchangeCode(code, verifier, "")
		if err != nil {
			return fmt.Errorf("failed to exchange token: %w", err)
		}

		// Step 6: Persist the retrieved token to the system keyring.
		if err := mal.SaveToken(token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		fmt.Println("Authentication with MyAnimeList completed successfully.")
		return nil
	},
}
