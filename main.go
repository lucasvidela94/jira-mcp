package main

import (
	"context"
	"flag"
	"fmt"
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
		provider, err := auth.Detect(cfg.URL, cfg.Username, cfg.APIToken, buildOAuthConfig())
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

	provider, err := auth.Detect(cfg.URL, cfg.Username, cfg.APIToken, buildOAuthConfig())
	if err != nil {
		log.Fatalf("auth error: %s", err)
	}

	client := jira.New(provider, nil)
	server := mcp.NewServer(client, cfg.EnabledTools)

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

// runAuth handles the `jira-mcp auth` subcommand and its flags.
func runAuth(args []string) error {
	statusFlag := false
	logoutFlag := false

	for _, a := range args {
		switch a {
		case "--status":
			statusFlag = true
		case "--logout":
			logoutFlag = true
		}
	}

	tokenPath := auth.DefaultTokenPath()

	switch {
	case logoutFlag:
		return cli.AuthLogout(os.Stdout, tokenPath)
	case statusFlag:
		store := auth.NewTokenStore(tokenPath)
		return cli.AuthStatus(os.Stdout, store)
	default:
		// Run the OAuth login flow.
		if ClientID == "" || ClientSecret == "" {
			fmt.Println("OAuth credentials not embedded in this build.")
			fmt.Println("Set JIRA_API_TOKEN for Basic Auth, or rebuild with ldflags:")
			fmt.Println("  -X main.ClientID=... -X main.ClientSecret=...")
			return nil
		}

		siteURL := os.Getenv("JIRA_URL")
		if siteURL == "" {
			return fmt.Errorf("JIRA_URL environment variable is required for OAuth login")
		}

		fmt.Println("Opening browser for Jira authorization...")

		oauthConfig := &oauth2.Config{
			ClientID:     ClientID,
			ClientSecret: ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://auth.atlassian.com/authorize",
				TokenURL: "https://auth.atlassian.com/oauth/token",
			},
			Scopes: []string{"read:jira-work", "write:jira-work", "read:jira-user"},
		}

		store := auth.NewTokenStore(auth.DefaultTokenPath())
		provider := auth.NewOAuthProvider("", siteURL, store, nil)

		ctx := context.Background()
		if err := provider.Login(ctx, oauthConfig); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		fmt.Printf("✓ Login successful!\n")
		fmt.Printf("  Cloud ID: %s\n", provider.CloudID())
		fmt.Printf("  Token saved to: %s\n", auth.DefaultTokenPath())
		fmt.Printf("\nYou can now run jira-mcp without JIRA_API_TOKEN.\n")
		return nil
	}
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

// buildOAuthConfig returns an oauth2.Config if embedded credentials are available,
// or nil if OAuth is not configured (e.g., dev build without ldflags).
func buildOAuthConfig() *oauth2.Config {
	if ClientID == "" || ClientSecret == "" {
		return nil
	}
	return &oauth2.Config{
		ClientID:     ClientID,
		ClientSecret: ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://auth.atlassian.com/authorize",
			TokenURL: "https://auth.atlassian.com/oauth/token",
		},
		Scopes: []string{"read:jira-work", "write:jira-work", "read:jira-user"},
	}
}
