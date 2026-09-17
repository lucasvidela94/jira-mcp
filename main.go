package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/debug"
	"strings"

	"github.com/lucasvidela94/jira-mcp/internal/auth"
	"github.com/lucasvidela94/jira-mcp/internal/cli"
	"github.com/lucasvidela94/jira-mcp/internal/config"
	"github.com/lucasvidela94/jira-mcp/internal/jira"
	"github.com/lucasvidela94/jira-mcp/internal/mcp"
	"github.com/lucasvidela94/jira-mcp/internal/update"
	"golang.org/x/oauth2"
)

// version is set by goreleaser at build time.
var version = "dev"

// ClientID and ClientSecret are the Atlassian OAuth 2.0 app credentials,
// embedded at build time via ldflags.
var (
	ClientID     string
	ClientSecret string
)

// resolveVersion returns the embedded version, or falls back to Go module build info.
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return version
}

type runner interface {
	ServeStdio() error
	ServeSSE(addr string) error
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "update" {
		if err := runUpdate(); err != nil {
			log.Fatalf("update failed: %s", err)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "auth" {
		if err := runAuth(os.Args[2:]); err != nil {
			log.Fatalf("auth failed: %s", err)
		}
		return
	}

	climode, args := detectCLI(os.Args[1:])
	if climode {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "config error: %s\n", err)
			os.Exit(1)
		}
		provider, err := auth.Detect(cfg.URL, cfg.Username, cfg.APIToken, buildOAuthConfig(cfg))
		if err != nil {
			fmt.Fprintf(os.Stderr, "auth error: %s\n", err)
			os.Exit(1)
		}
		client := jira.New(provider, nil)
		if err := cli.Run(args, client); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n\nAvailable commands:\n", err)
			os.Exit(1)
		}
		return
	}

	transport := flag.String("transport", "stdio", "transport type: stdio or http")
	port := flag.String("port", "8080", "HTTP listen port (used with --transport http)")
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println(resolveVersion())
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %s\n", err)
		os.Exit(1)
	}

	provider, err := auth.Detect(cfg.URL, cfg.Username, cfg.APIToken, buildOAuthConfig(cfg))
	if err != nil {
		log.Fatalf("auth error: %s", err)
	}

	client := jira.New(provider, nil)
	server := mcp.NewServer(client, cfg.EnabledTools, mcp.WithConfirmation(cfg.ConfirmWrite))

	if err := dispatchTransport(server, *transport, *port); err != nil {
		log.Fatalf("server error: %s", err)
	}
}

func runUpdate() error {
	u := update.NewSelfUpdater()
	result, err := u.Run(context.Background(), resolveVersion())
	if err != nil {
		return err
	}
	fmt.Println(result.Message)
	if result.Updated {
		fmt.Println("Restart your MCP client to use the new version.")
	}
	return nil
}

// authOptions is the parsed form of `jira-mcp auth`.
type authOptions struct {
	status  bool
	logout  bool
	help    bool
	siteURL string // from --url; empty when not given
}

// parseAuthArgs parses the auth subcommand flags strictly. Unknown flags and a
// missing --url value are errors instead of being silently ignored.
func parseAuthArgs(args []string) (authOptions, error) {
	var opts authOptions
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; arg {
		case "--status":
			opts.status = true
		case "--logout":
			opts.logout = true
		case "--help", "-h":
			opts.help = true
		case "--url":
			if i+1 >= len(args) {
				return authOptions{}, fmt.Errorf("--url requires a value")
			}
			i++
			opts.siteURL = args[i]
		default:
			return authOptions{}, fmt.Errorf("unknown flag for 'auth': %s", arg)
		}
	}
	return opts, nil
}

func printAuthUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: jira-mcp auth [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Log in to Jira Cloud with Atlassian OAuth 2.0.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --url <site>   Jira site URL (optional; also read from JIRA_URL)")
	fmt.Fprintln(w, "  --status       Show the current token status and exit")
	fmt.Fprintln(w, "  --logout       Remove the stored token and exit")
	fmt.Fprintln(w, "  -h, --help     Show this help")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "OAuth needs an Atlassian OAuth 2.0 (3LO) app. Provide your own with")
	fmt.Fprintln(w, "JIRA_OAUTH_CLIENT_ID and JIRA_OAUTH_CLIENT_SECRET, or use Basic Auth")
	fmt.Fprintln(w, "by setting JIRA_URL, JIRA_USERNAME and JIRA_API_TOKEN.")
}

// runAuth handles the `jira-mcp auth` subcommand and its flags.
func runAuth(args []string) error {
	opts, err := parseAuthArgs(args)
	if err != nil {
		return err
	}
	if opts.help {
		printAuthUsage(os.Stdout)
		return nil
	}

	tokenPath := auth.DefaultTokenPath()

	switch {
	case opts.logout:
		return cli.AuthLogout(os.Stdout, tokenPath)
	case opts.status:
		store := auth.NewTokenStore(tokenPath)
		return cli.AuthStatus(os.Stdout, store)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// The site URL is only a hint: the Cloud ID is resolved from Atlassian's
	// accessible-resources endpoint, so its absence must not block login.
	siteURL, err := resolveSiteURL(opts.siteURL, os.Getenv)
	if err != nil {
		return err
	}

	creds := resolveOAuthCredentials(cfg.OAuthClientID, cfg.OAuthClientSecret, ClientID, ClientSecret)
	if !creds.configured() {
		return errors.New("OAuth is not configured: no Atlassian app credentials were found.\n" +
			"Set your own app credentials with JIRA_OAUTH_CLIENT_ID and JIRA_OAUTH_CLIENT_SECRET\n" +
			"(create an OAuth 2.0 (3LO) app at https://developer.atlassian.com/console/myapps/ and\n" +
			"register the redirect URL http://127.0.0.1:<port>/callback),\n" +
			"or use Basic Auth by setting JIRA_URL, JIRA_USERNAME and JIRA_API_TOKEN.")
	}

	if siteURL != "" {
		fmt.Fprintf(os.Stderr, "Jira site: %s\n", siteURL)
	}
	fmt.Fprintf(os.Stderr, "OAuth app credentials: %s\n", creds.source)

	store := auth.NewTokenStore(tokenPath)
	provider := auth.NewOAuthProvider("", siteURL, store, nil)

	ctx := context.Background()
	if err := provider.Login(ctx, oauthConfigFrom(creds)); err != nil {
		return authLoginError(err, siteURL)
	}

	fmt.Printf("✓ Login successful!\n")
	fmt.Printf("  Cloud ID: %s\n", provider.CloudID())
	fmt.Printf("  Token saved to: %s\n", auth.DefaultTokenPath())
	fmt.Printf("\nYou can now run jira-mcp without JIRA_API_TOKEN.\n")
	return nil
}

// authLoginError translates an OAuth login failure into a user-facing error.
// A callback timeout is the only signal the CLI gets when Atlassian rejects the
// OAuth app on its own page, so it gets dedicated guidance; everything else
// keeps the plain "login failed" shape.
func authLoginError(err error, siteURL string) error {
	if errors.Is(err, auth.ErrCallbackTimeout) {
		return errors.New(authTimeoutHelp(siteURL))
	}
	return fmt.Errorf("login failed: %w", err)
}

// authTimeoutHelp explains the most likely cause of a missing callback and the
// two ways forward. It never includes any credential value.
func authTimeoutHelp(siteURL string) string {
	where := "your Jira site"
	if siteURL != "" {
		where = siteURL
	}
	return fmt.Sprintf(`No authorization was received for %s before the time limit.

The most likely cause is that the OAuth app used for this login is not authorized
for that site: Atlassian shows "We couldn't identify the app requesting access"
and never redirects back. Unmodified release binaries embed the maintainer's app,
which only works for the maintainer's own site.

Two ways forward:
  1. Use your own Atlassian OAuth 2.0 (3LO) app. Create one at
     https://developer.atlassian.com/console/myapps/, register the redirect URL
     http://127.0.0.1:<port>/callback, then export JIRA_OAUTH_CLIENT_ID and
     JIRA_OAUTH_CLIENT_SECRET and run 'jira-mcp auth' again.
  2. Use Basic Auth instead: export JIRA_URL, JIRA_USERNAME and JIRA_API_TOKEN.`, where)
}

// resolveSiteURL resolves the optional site URL hint. Precedence: the --url
// flag, then JIRA_URL, then empty. An empty result is valid: the OAuth flow
// derives the Cloud ID from Atlassian's accessible-resources endpoint, so the
// site URL is never required. A non-empty result must carry an http(s) scheme.
//
// There is deliberately no interactive prompt: the value is optional, and
// prompting would hang in pipes and CI.
func resolveSiteURL(flagValue string, getenv func(string) string) (string, error) {
	value := strings.TrimSpace(flagValue)
	if value == "" && getenv != nil {
		value = strings.TrimSpace(getenv("JIRA_URL"))
	}
	if value == "" {
		return "", nil
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		return "", fmt.Errorf("site URL must start with http:// or https://, got %q", value)
	}
	return strings.TrimRight(value, "/"), nil
}

// detectCLI checks whether the invocation looks like a CLI subcommand.
// A bare positional argument (not starting with "-") activates CLI mode.
// The --cli flag also forces CLI mode and is stripped from the arg slice.
func detectCLI(rawArgs []string) (bool, []string) {
	var filtered []string
	cliMode := false
	for _, a := range rawArgs {
		if a == "--cli" {
			cliMode = true
			continue
		}
		filtered = append(filtered, a)
	}
	if !cliMode && len(filtered) > 0 && !strings.HasPrefix(filtered[0], "-") {
		cliMode = true
	}
	return cliMode, filtered
}

func dispatchTransport(server runner, transport, port string) error {
	switch transport {
	case "http":
		addr := fmt.Sprintf(":%s", port)
		return server.ServeSSE(addr)
	case "stdio":
		return server.ServeStdio()
	default:
		return fmt.Errorf("invalid transport: %s (must be stdio or http)", transport)
	}
}

// oauthCredentials is the resolved Atlassian OAuth app credential pair plus its
// provenance, so the CLI can report where the credentials came from without
// ever printing the secret.
type oauthCredentials struct {
	clientID     string
	clientSecret string
	source       string // "environment", "embedded", or "" when unconfigured
}

// configured reports whether a complete credential pair was resolved.
func (c oauthCredentials) configured() bool {
	return c.clientID != "" && c.clientSecret != ""
}

// resolveOAuthCredentials applies the OAuth credential precedence: environment
// over build-time embedded values. The environment pair is taken atomically, so
// a partial pair is ignored in favour of the embedded one rather than being
// sent to Atlassian as a broken combination.
func resolveOAuthCredentials(envID, envSecret, embeddedID, embeddedSecret string) oauthCredentials {
	envID = strings.TrimSpace(envID)
	envSecret = strings.TrimSpace(envSecret)
	if envID != "" && envSecret != "" {
		return oauthCredentials{clientID: envID, clientSecret: envSecret, source: "environment"}
	}

	embeddedID = strings.TrimSpace(embeddedID)
	embeddedSecret = strings.TrimSpace(embeddedSecret)
	if embeddedID != "" && embeddedSecret != "" {
		return oauthCredentials{clientID: embeddedID, clientSecret: embeddedSecret, source: "embedded"}
	}

	return oauthCredentials{}
}

// buildOAuthConfig returns an oauth2.Config for the resolved credentials, or nil
// when OAuth is not configured (for example a Docker build with no embedded
// credentials and no environment variables).
func buildOAuthConfig(cfg config.Config) *oauth2.Config {
	return oauthConfigFrom(resolveOAuthCredentials(cfg.OAuthClientID, cfg.OAuthClientSecret, ClientID, ClientSecret))
}

// oauthConfigFrom builds the Atlassian OAuth 2.0 (3LO) client configuration.
func oauthConfigFrom(creds oauthCredentials) *oauth2.Config {
	if !creds.configured() {
		return nil
	}
	return &oauth2.Config{
		ClientID:     creds.clientID,
		ClientSecret: creds.clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://auth.atlassian.com/authorize",
			TokenURL: "https://auth.atlassian.com/oauth/token",
		},
		Scopes: []string{"read:jira-work", "write:jira-work", "read:jira-user"},
	}
}
