package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/the20100/g-search-console-cli/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Google Search Console authentication",
}

var (
	credentialsFile  string
	clientIDFlag     string
	clientSecretFlag string
	authNoBrowser    bool
)

var authSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Authenticate with Google Search Console via OAuth2",
	Long: `Authenticate using OAuth2 (Desktop app flow).

Setup steps:
  1. Go to https://console.cloud.google.com/apis/credentials
  2. Create credentials → OAuth client ID → Desktop app
  3. Add http://localhost:8080 as an authorized redirect URI
  4. Download the credentials.json file

Then run:
  gsc auth setup --credentials /path/to/credentials.json
  # or provide client ID/secret interactively

The CLI will open your browser for Google authorization.

On a remote server (VPS) where no browser is available:
  gsc auth setup --credentials /path/to/credentials.json --no-browser

  This prints the auth URL for you to open locally. After authorizing, your
  browser will redirect to localhost:8080 (which will fail to load — that's ok).
  Copy the full URL from the address bar and paste it into the terminal.`,
	RunE: runAuthSetup,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE:  runAuthStatus,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved credentials from the config file",
	RunE:  runAuthLogout,
}

func init() {
	authSetupCmd.Flags().StringVar(&credentialsFile, "credentials", "", "Path to Google OAuth2 credentials.json file")
	authSetupCmd.Flags().StringVar(&clientIDFlag, "client-id", "", "OAuth2 client ID")
	authSetupCmd.Flags().StringVar(&clientSecretFlag, "client-secret", "", "OAuth2 client secret")
	authSetupCmd.Flags().BoolVar(&authNoBrowser, "no-browser", false, "Manual auth flow for remote/VPS: print the URL, prompt for the redirect URL")

	authCmd.AddCommand(authSetupCmd, authStatusCmd, authLogoutCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthSetup(cmd *cobra.Command, args []string) error {
	clientID := clientIDFlag
	clientSecret := clientSecretFlag

	// Parse credentials.json if provided.
	if credentialsFile != "" {
		data, err := os.ReadFile(credentialsFile)
		if err != nil {
			return fmt.Errorf("reading credentials file: %w", err)
		}
		creds, err := parseCredentialsJSON(data)
		if err != nil {
			return fmt.Errorf("parsing credentials.json: %w", err)
		}
		clientID = creds.ClientID
		clientSecret = creds.ClientSecret
	}

	// Fall back to env vars.
	if clientID == "" {
		clientID = os.Getenv("GOOGLE_CLIENT_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	}

	// Fall back to existing config.
	if clientID == "" || clientSecret == "" {
		cfg, _ := config.Load()
		if cfg != nil {
			if clientID == "" {
				clientID = cfg.ClientID
			}
			if clientSecret == "" {
				clientSecret = cfg.ClientSecret
			}
		}
	}

	// Prompt interactively.
	if clientID == "" {
		fmt.Print("Enter OAuth2 Client ID: ")
		fmt.Scan(&clientID)
	}
	if clientSecret == "" {
		fmt.Print("Enter OAuth2 Client Secret: ")
		fmt.Scan(&clientSecret)
	}

	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("client ID and client secret are required")
	}

	oauthCfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8080",
		Scopes: []string{
			"https://www.googleapis.com/auth/webmasters",
		},
	}

	authURL := oauthCfg.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	var code string
	if authNoBrowser {
		var err error
		code, err = runOAuthFlowManual(authURL)
		if err != nil {
			return err
		}
	} else {
		// Start a local HTTP server to capture the OAuth2 redirect.
		codeCh := make(chan string, 1)
		errCh := make(chan error, 1)

		mux := http.NewServeMux()
		server := &http.Server{Addr: ":8080", Handler: mux}

		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if code == "" {
				errMsg := r.URL.Query().Get("error")
				if errMsg == "" {
					errMsg = "unknown error"
				}
				errCh <- fmt.Errorf("authorization failed: %s", errMsg)
				fmt.Fprintf(w, "<html><body><h2>Authorization failed: %s</h2><p>You can close this tab.</p></body></html>", errMsg)
				return
			}
			codeCh <- code
			fmt.Fprintf(w, "<html><body><h2>Authorization successful!</h2><p>You can close this tab and return to the terminal.</p></body></html>")
		})

		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- fmt.Errorf("local callback server error: %w", err)
			}
		}()

		fmt.Printf("\nOpening browser for authorization...\n")
		fmt.Printf("If the browser doesn't open, visit this URL manually:\n\n  %s\n\n", authURL)
		openBrowser(authURL)

		fmt.Println("Waiting for authorization (timeout: 5 minutes)...")

		select {
		case code = <-codeCh:
		case err := <-errCh:
			server.Shutdown(context.Background())
			return err
		case <-time.After(5 * time.Minute):
			server.Shutdown(context.Background())
			return fmt.Errorf("authorization timed out")
		}

		server.Shutdown(context.Background())
	}

	// Exchange authorization code for tokens.
	token, err := oauthCfg.Exchange(context.Background(), code)
	if err != nil {
		return fmt.Errorf("exchanging code for token: %w", err)
	}

	// Save credentials and tokens to config.
	cfg := &config.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		TokenExpiry:  token.Expiry,
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("\nAuthentication successful! Config saved to:\n  %s\n", config.Path())
	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Printf("Config: %s\n\n", config.Path())

	if envID := os.Getenv("GOOGLE_CLIENT_ID"); envID != "" {
		fmt.Println("Client ID: (from GOOGLE_CLIENT_ID env var)")
	} else if cfg.ClientID != "" {
		fmt.Printf("Client ID: %s\n", maskString(cfg.ClientID))
	} else {
		fmt.Println("Client ID: (not set)")
	}

	if cfg.RefreshToken != "" || cfg.AccessToken != "" {
		fmt.Println("Auth status: authenticated")
		if !cfg.TokenExpiry.IsZero() {
			if cfg.TokenExpiry.Before(time.Now()) {
				fmt.Println("Access token: expired (will refresh automatically using refresh token)")
			} else {
				fmt.Printf("Access token: valid until %s\n", cfg.TokenExpiry.Format(time.RFC3339))
			}
		}
		if cfg.RefreshToken != "" {
			fmt.Printf("Refresh token: %s\n", maskString(cfg.RefreshToken))
		}
	} else {
		fmt.Println("Auth status: not authenticated")
		fmt.Println("\nRun: gsc auth setup")
	}
	return nil
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	if err := config.Clear(); err != nil {
		return fmt.Errorf("removing config: %w", err)
	}
	fmt.Println("Credentials removed from config.")
	return nil
}

// credentialsJSONEntry holds the client credentials fields within credentials.json.
type credentialsJSONEntry struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// credentialsJSONFile represents the top-level structure of a downloaded credentials.json.
type credentialsJSONFile struct {
	Installed *credentialsJSONEntry `json:"installed"`
	Web       *credentialsJSONEntry `json:"web"`
}

func parseCredentialsJSON(data []byte) (*credentialsJSONEntry, error) {
	var wrapper credentialsJSONFile
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	if wrapper.Installed != nil {
		return wrapper.Installed, nil
	}
	if wrapper.Web != nil {
		return wrapper.Web, nil
	}
	return nil, fmt.Errorf("could not find 'installed' or 'web' credentials in file")
}

func runOAuthFlowManual(authURL string) (string, error) {
	fmt.Printf("\nOpen the following URL in your browser:\n\n%s\n\n", authURL)
	fmt.Println("After authorizing, your browser will be redirected to localhost:8080.")
	fmt.Println("That page will fail to load — that's expected on a remote server.")
	fmt.Println("Copy the full URL from the browser's address bar and paste it below.")
	fmt.Print("\nRedirect URL: ")

	reader := bufio.NewReader(os.Stdin)
	rawURL, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading redirect URL: %w", err)
	}
	rawURL = strings.TrimSpace(rawURL)

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parsing redirect URL: %w", err)
	}

	if errMsg := parsed.Query().Get("error"); errMsg != "" {
		return "", fmt.Errorf("authorization failed: %s", errMsg)
	}

	code := parsed.Query().Get("code")
	if code == "" {
		return "", fmt.Errorf("no authorization code found in URL — make sure you copied the full redirect URL")
	}

	return code, nil
}
