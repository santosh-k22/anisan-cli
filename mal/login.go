package mal

import (
	"context"
	"fmt"
	"net/http"

	"github.com/anisan-cli/anisan/log"
	"github.com/anisan-cli/anisan/open"
	"github.com/spf13/viper"
)

// Login orchestrates the MyAnimeList OAuth2 PKCE authentication flow, including local callback server lifecycle management.
func Login() error {
	clientID := viper.GetString("tracker.mal.client_id")
	if clientID == "" {
		return fmt.Errorf("MAL client_id is missing in config")
	}

	pkce, err := GeneratePKCE()
	if err != nil {
		return err
	}

	authURL := fmt.Sprintf("https://myanimelist.net/v1/oauth2/authorize?response_type=code&client_id=%s&code_challenge=%s", clientID, pkce)

	log.Info("Opening browser for MyAnimeList authentication...")
	if err := open.Start(authURL); err != nil {
		log.Warnf("Could not open browser. Please navigate manually: %s", authURL)
	}

	srv := &http.Server{Addr: "127.0.0.1:8080"}
	var authErr error

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprintln(w, "Authentication failed. No code returned. You may close this window.")
			authErr = fmt.Errorf("no authorization code returned from MAL")
			go srv.Shutdown(context.Background())
			return
		}

		fmt.Fprintln(w, "Authentication successful! You may close this window and return to your terminal.")
		authErr = ExchangeToken(code, pkce)
		go srv.Shutdown(context.Background())
	})

	log.Info("Waiting for authorization callback on http://127.0.0.1:8080/callback...")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("local server failed: %w", err)
	}

	if authErr != nil {
		return fmt.Errorf("OAuth exchange failed: %w", authErr)
	}

	log.Info("Successfully authenticated with MyAnimeList.")
	return nil
}
