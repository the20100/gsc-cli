package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"github.com/the20100/g-search-console-cli/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	gsc "google.golang.org/api/searchconsole/v1"
)

var (
	jsonFlag   bool
	prettyFlag bool

	// Global Search Console service, set in PersistentPreRunE.
	svc *gsc.Service
)

var rootCmd = &cobra.Command{
	Use:   "gsc",
	Short: "Google Search Console CLI",
	Long: `gsc is a CLI tool for the Google Search Console API.

It outputs JSON when piped (for agent use) and human-readable tables in a terminal.

Credential setup:
  1. Create a Google Cloud project and enable the Search Console API
  2. Create OAuth2 credentials (Desktop app) at:
     https://console.cloud.google.com/apis/credentials
  3. Run: gsc auth setup

Token resolution:
  1. GOOGLE_CLIENT_ID + GOOGLE_CLIENT_SECRET env vars (client creds)
  2. Config file (~/.config/g-search-console/config.json via: gsc auth setup)

Examples:
  gsc auth setup
  gsc sites list
  gsc analytics query --site https://example.com --start 2024-01-01 --end 2024-01-31
  gsc analytics query --site https://example.com --dimensions query,page --limit 100
  gsc sitemaps list https://example.com
  gsc sitemaps submit https://example.com https://example.com/sitemap.xml
  gsc inspect https://example.com/page --site https://example.com
  gsc mobile-test https://example.com`,
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Force JSON output")
	rootCmd.PersistentFlags().BoolVar(&prettyFlag, "pretty", false, "Force pretty-printed JSON output (implies --json)")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if isAuthCommand(cmd) || cmd.Name() == "info" || cmd.Name() == "update" {
			return nil
		}
		return initService(cmd.Context())
	}

	rootCmd.AddCommand(infoCmd)
}

// savingTokenSource wraps an oauth2.TokenSource and persists refreshed tokens to config.
type savingTokenSource struct {
	source oauth2.TokenSource
	cfg    *config.Config
}

func (s *savingTokenSource) Token() (*oauth2.Token, error) {
	token, err := s.source.Token()
	if err != nil {
		return nil, err
	}
	// Persist if the access token was refreshed.
	if token.AccessToken != s.cfg.AccessToken {
		s.cfg.AccessToken = token.AccessToken
		s.cfg.TokenExpiry = token.Expiry
		_ = config.Save(s.cfg)
	}
	return token, nil
}

// resolveEnv returns the value of the first non-empty environment variable from the given names.
func resolveEnv(names ...string) string {
	for _, name := range names {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}
	return ""
}

func maskOrEmpty(v string) string {
	if v == "" {
		return "(not set)"
	}
	if len(v) <= 8 {
		return "***"
	}
	return v[:4] + "..." + v[len(v)-4:]
}

func initService(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	clientID := resolveEnv(
		"GOOGLE_CLIENT_ID",
		"GOOGLE_OAUTH_CLIENT_ID",
		"GCP_CLIENT_ID",
		"GSC_CLIENT_ID",
		"GCLOUD_CLIENT_ID",
		"GOOGLE_CLIENT",
	)
	if clientID == "" {
		clientID = cfg.ClientID
	}
	clientSecret := resolveEnv(
		"GOOGLE_CLIENT_SECRET",
		"GOOGLE_OAUTH_CLIENT_SECRET",
		"GCP_CLIENT_SECRET",
		"GSC_CLIENT_SECRET",
		"GCLOUD_CLIENT_SECRET",
		"GOOGLE_SECRET",
	)
	if clientSecret == "" {
		clientSecret = cfg.ClientSecret
	}

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("not configured — run: gsc auth setup\nor set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET env vars")
	}

	if cfg.RefreshToken == "" && cfg.AccessToken == "" {
		return fmt.Errorf("not authenticated — run: gsc auth setup")
	}

	oauthCfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes: []string{
			"https://www.googleapis.com/auth/webmasters",
		},
	}

	token := &oauth2.Token{
		AccessToken:  cfg.AccessToken,
		RefreshToken: cfg.RefreshToken,
		TokenType:    cfg.TokenType,
		Expiry:       cfg.TokenExpiry,
	}

	ts := oauthCfg.TokenSource(ctx, token)
	savingTS := &savingTokenSource{source: ts, cfg: cfg}
	httpClient := oauth2.NewClient(ctx, savingTS)

	svc, err = gsc.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}
	return nil
}

// isAuthCommand returns true if cmd is the "auth" command or one of its children.
func isAuthCommand(cmd *cobra.Command) bool {
	if cmd.Name() == "auth" {
		return true
	}
	p := cmd.Parent()
	for p != nil {
		if p.Name() == "auth" {
			return true
		}
		p = p.Parent()
	}
	return false
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show tool info: config path, auth status, and environment",
	Run: func(cmd *cobra.Command, args []string) {
		printInfo()
	},
}

func printInfo() {
	fmt.Println("gsc — Google Search Console CLI")
	fmt.Println()

	exe, _ := os.Executable()
	fmt.Printf("  binary:  %s\n", exe)
	fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println()

	fmt.Println("  config paths by OS:")
	fmt.Println("    macOS:    ~/Library/Application Support/g-search-console/config.json")
	fmt.Println("    Linux:    ~/.config/g-search-console/config.json")
	fmt.Println("    Windows:  %AppData%\\g-search-console\\config.json")
	fmt.Printf("  config:   %s\n", config.Path())
	fmt.Println()

	cfg, _ := config.Load()
	clientIDSource := "(not set)"
	if os.Getenv("GOOGLE_CLIENT_ID") != "" {
		clientIDSource = "GOOGLE_CLIENT_ID env var"
	} else if cfg != nil && cfg.ClientID != "" {
		clientIDSource = "config file"
	}
	fmt.Printf("  client ID source: %s\n", clientIDSource)

	authStatus := "not authenticated"
	if cfg != nil && (cfg.RefreshToken != "" || cfg.AccessToken != "") {
		authStatus = "authenticated"
		if !cfg.TokenExpiry.IsZero() {
			if cfg.TokenExpiry.Before(time.Now()) {
				authStatus += " (access token expired, will refresh automatically)"
			} else {
				authStatus += fmt.Sprintf(" (token valid until %s)", cfg.TokenExpiry.Format(time.RFC3339))
			}
		}
	}
	fmt.Printf("  auth status: %s\n", authStatus)
	fmt.Println()
	fmt.Println("  credential resolution order:")
	fmt.Println("    1. GOOGLE_CLIENT_ID + GOOGLE_CLIENT_SECRET env vars")
	fmt.Println("    2. config file (gsc auth setup)")
}
