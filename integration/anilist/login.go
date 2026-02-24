package anilist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/anisan-cli/anisan/auth"
	"github.com/anisan-cli/anisan/log"
	"github.com/anisan-cli/anisan/open"
)

const (
	anilistOAuthURL     = "https://anilist.co/api/v2/oauth"
	anilistClientID     = "36439"
	anilistClientSecret = "F195CuZAnDfd5OjNkb00NUmPNoxbn7e4QBZnX2tc"
	anilistRedirectURI  = "http://localhost:8000/oauth/callback"
	anilistServerPort   = 8000
)

const successHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authentication Successful</title>
    <style>
        body {
            margin: 0;
            padding: 0;
            background-color: #0f0f11;
            color: #ffffff;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            text-align: center;
        }
        .container {
            animation: fadeIn 0.8s ease-out;
        }
        h1 {
            font-size: 24px;
            font-weight: 500;
            margin-bottom: 8px;
            letter-spacing: -0.5px;
        }
        p {
            font-size: 15px;
            color: #88888b;
            font-weight: 400;
        }
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Authentication Successful</h1>
        <p>You may safely close this tab and return to the terminal.</p>
    </div>
</body>
</html>`

const errorHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authentication Failed</title>
    <style>
        body { margin: 0; padding: 0; background-color: #0f0f11; color: #ffffff; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; text-align: center; }
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

// AuthenticateWithBrowser initiates the OAuth2 authorization code flow
func AuthenticateWithBrowser() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	callbackCh := make(chan string, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", anilistServerPort),
		Handler: mux,
	}

	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		errorParam := r.URL.Query().Get("error")

		w.Header().Set("Content-Type", "text/html")

		if errorParam != "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, errorHTML, errorParam)
			errCh <- fmt.Errorf("oauth error: %s", errorParam)
			return
		}

		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, errorHTML, "No authorization code received.")
			errCh <- fmt.Errorf("no authorization code received")
			return
		}

		// Exchange authorization code for token
		tokenURL := fmt.Sprintf("%s/token", anilistOAuthURL)
		data := url.Values{
			"grant_type":    {"authorization_code"},
			"client_id":     {anilistClientID},
			"client_secret": {anilistClientSecret},
			"redirect_uri":  {anilistRedirectURI},
			"code":          {code},
		}

		resp, err := http.PostForm(tokenURL, data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, errorHTML, "Failed to exchange code for token.")
			errCh <- fmt.Errorf("exchange failed: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, errorHTML, fmt.Sprintf("Token exchange failed with status: %d", resp.StatusCode))
			errCh <- fmt.Errorf("exchange failed status %d", resp.StatusCode)
			return
		}

		var tokenResp struct {
			AccessToken string `json:"access_token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			errCh <- err
			return
		}

		if tokenResp.AccessToken == "" {
			errCh <- fmt.Errorf("no access token in response")
			return
		}

		// Serve success beautifully
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, successHTML)

		callbackCh <- tokenResp.AccessToken
	})

	// Start server in background
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("failed to start local server: %w", err)
		}
	}()
	defer srv.Shutdown(ctx)

	authURL := fmt.Sprintf("%s/authorize?client_id=%s&redirect_uri=%s&response_type=code",
		anilistOAuthURL,
		anilistClientID,
		url.QueryEscape(anilistRedirectURI))

	fmt.Println("Opening browser for Anilist authentication...")
	if err := open.Start(authURL); err != nil {
		log.Warn("Failed to open browser: " + err.Error())
		fmt.Printf("Please manually visit: %s\n", authURL)
	}

	log.Info("Waiting for callback on port 8000...")

	var accessToken string
	select {
	case accessToken = <-callbackCh:
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return fmt.Errorf("authentication timed out")
	}

	// Save using the existing logic so keychain is preserved
	if err := auth.SetToken(accessToken); err != nil {
		return fmt.Errorf("failed to save token to keyring: %w", err)
	}

	return nil
}
