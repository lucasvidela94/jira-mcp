package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/lucasvidela94/jira-mcp/internal/auth"
	"github.com/lucasvidela94/jira-mcp/internal/config"
)

// swapEmbedded temporarily replaces the ldflags-injected OAuth credentials.
func swapEmbedded(id, secret string) func() {
	prevID, prevSecret := ClientID, ClientSecret
	ClientID, ClientSecret = id, secret
	return func() {
		ClientID, ClientSecret = prevID, prevSecret
	}
}

type fakeServer struct {
	serveStdioCalled bool
	serveSSECalled   bool
	serveSSEAddr     string
	returnErr        error
}

func (f *fakeServer) ServeStdio() error {
	f.serveStdioCalled = true
	return f.returnErr
}

func (f *fakeServer) ServeSSE(addr string) error {
	f.serveSSECalled = true
	f.serveSSEAddr = addr
	return f.returnErr
}

func TestDispatchTransport_DefaultsToStdio(t *testing.T) {
	server := &fakeServer{}
	if err := dispatchTransport(server, "stdio", "8080"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !server.serveStdioCalled {
		t.Error("expected ServeStdio to be called")
	}
	if server.serveSSECalled {
		t.Error("expected ServeSSE not to be called")
	}
}

func TestDispatchTransport_HTTP(t *testing.T) {
	server := &fakeServer{}
	if err := dispatchTransport(server, "http", "9090"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if server.serveStdioCalled {
		t.Error("expected ServeStdio not to be called")
	}
	if !server.serveSSECalled {
		t.Fatal("expected ServeSSE to be called")
	}
	if server.serveSSEAddr != ":9090" {
		t.Errorf("expected address :9090, got %s", server.serveSSEAddr)
	}
}

func TestDispatchTransport_InvalidTransport(t *testing.T) {
	server := &fakeServer{}
	err := dispatchTransport(server, "ws", "8080")
	if err == nil {
		t.Fatal("expected error")
	}
	if server.serveStdioCalled || server.serveSSECalled {
		t.Error("expected no server method to be called")
	}
}

func TestDispatchTransport_PropagatesError(t *testing.T) {
	want := errors.New("boom")
	server := &fakeServer{returnErr: want}

	err := dispatchTransport(server, "stdio", "8080")
	if !errors.Is(err, want) {
		t.Errorf("expected error %v, got %v", want, err)
	}
}

// --- OAuth credential resolution ---

func TestResolveOAuthCredentials(t *testing.T) {
	tests := []struct {
		name           string
		envID          string
		envSecret      string
		embeddedID     string
		embeddedSecret string
		wantID         string
		wantSecret     string
		wantSource     string
	}{
		{
			name:           "environment pair wins over embedded",
			envID:          "env-id",
			envSecret:      "env-secret",
			embeddedID:     "emb-id",
			embeddedSecret: "emb-secret",
			wantID:         "env-id",
			wantSecret:     "env-secret",
			wantSource:     "environment",
		},
		{
			name:       "environment only",
			envID:      "env-id",
			envSecret:  "env-secret",
			wantID:     "env-id",
			wantSecret: "env-secret",
			wantSource: "environment",
		},
		{
			name:           "embedded only",
			embeddedID:     "emb-id",
			embeddedSecret: "emb-secret",
			wantID:         "emb-id",
			wantSecret:     "emb-secret",
			wantSource:     "embedded",
		},
		{
			name:           "partial environment pair is ignored in favour of embedded",
			envID:          "env-id",
			embeddedID:     "emb-id",
			embeddedSecret: "emb-secret",
			wantID:         "emb-id",
			wantSecret:     "emb-secret",
			wantSource:     "embedded",
		},
		{
			name:       "partial environment pair with no embedded pair is unconfigured",
			envSecret:  "env-secret",
			wantSource: "",
		},
		{
			name:       "nothing configured",
			wantSource: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveOAuthCredentials(tt.envID, tt.envSecret, tt.embeddedID, tt.embeddedSecret)
			if got.clientID != tt.wantID {
				t.Errorf("clientID = %q, want %q", got.clientID, tt.wantID)
			}
			if got.clientSecret != tt.wantSecret {
				t.Errorf("clientSecret = %q, want %q", got.clientSecret, tt.wantSecret)
			}
			if got.source != tt.wantSource {
				t.Errorf("source = %q, want %q", got.source, tt.wantSource)
			}
			if wantConfigured := tt.wantSource != ""; got.configured() != wantConfigured {
				t.Errorf("configured() = %v, want %v", got.configured(), wantConfigured)
			}
		})
	}
}

func TestBuildOAuthConfig_UsesEnvironmentCredentials(t *testing.T) {
	defer swapEmbedded("emb-id", "emb-secret")()

	cfg := config.Config{OAuthClientID: "cfg-id", OAuthClientSecret: "cfg-secret"}
	got := buildOAuthConfig(cfg)
	if got == nil {
		t.Fatal("expected a non-nil oauth2.Config")
	}
	if got.ClientID != "cfg-id" || got.ClientSecret != "cfg-secret" {
		t.Errorf("expected cfg credentials, got %q/%q", got.ClientID, got.ClientSecret)
	}
}

func TestBuildOAuthConfig_FallsBackToEmbedded(t *testing.T) {
	defer swapEmbedded("emb-id", "emb-secret")()

	got := buildOAuthConfig(config.Config{})
	if got == nil {
		t.Fatal("expected a non-nil oauth2.Config")
	}
	if got.ClientID != "emb-id" || got.ClientSecret != "emb-secret" {
		t.Errorf("expected embedded credentials, got %q/%q", got.ClientID, got.ClientSecret)
	}
}

func TestBuildOAuthConfig_PreservesAtlassianEndpointAndScopes(t *testing.T) {
	defer swapEmbedded("emb-id", "emb-secret")()

	got := buildOAuthConfig(config.Config{})
	if got == nil {
		t.Fatal("expected a non-nil oauth2.Config")
	}
	if got.Endpoint.AuthURL != "https://auth.atlassian.com/authorize" {
		t.Errorf("unexpected AuthURL %q", got.Endpoint.AuthURL)
	}
	if got.Endpoint.TokenURL != "https://auth.atlassian.com/oauth/token" {
		t.Errorf("unexpected TokenURL %q", got.Endpoint.TokenURL)
	}
	wantScopes := []string{"read:jira-work", "write:jira-work", "read:jira-user"}
	if len(got.Scopes) != len(wantScopes) {
		t.Fatalf("expected %d scopes, got %d", len(wantScopes), len(got.Scopes))
	}
	for i, scope := range wantScopes {
		if got.Scopes[i] != scope {
			t.Errorf("scope[%d] = %q, want %q", i, got.Scopes[i], scope)
		}
	}
}

func TestBuildOAuthConfig_NilWhenUnconfigured(t *testing.T) {
	defer swapEmbedded("", "")()

	if got := buildOAuthConfig(config.Config{}); got != nil {
		t.Errorf("expected nil oauth2.Config when unconfigured, got %+v", got)
	}
}

// --- auth subcommand argument parsing ---

func TestParseAuthArgs(t *testing.T) {
	t.Run("no arguments", func(t *testing.T) {
		opts, err := parseAuthArgs(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.status || opts.logout || opts.siteURL != "" {
			t.Errorf("expected zero options, got %+v", opts)
		}
	})

	t.Run("status and logout flags", func(t *testing.T) {
		opts, err := parseAuthArgs([]string{"--status", "--logout"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !opts.status || !opts.logout {
			t.Errorf("expected both flags set, got %+v", opts)
		}
	})

	t.Run("url captures its value", func(t *testing.T) {
		opts, err := parseAuthArgs([]string{"--url", "https://magmalabs.atlassian.net"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.siteURL != "https://magmalabs.atlassian.net" {
			t.Errorf("expected site URL, got %q", opts.siteURL)
		}
	})

	t.Run("unknown flag is an error", func(t *testing.T) {
		_, err := parseAuthArgs([]string{"--logout-everything"})
		if err == nil {
			t.Fatal("expected an error for an unknown flag")
		}
		if !strings.Contains(err.Error(), "--logout-everything") {
			t.Errorf("expected the error to name the flag, got %v", err)
		}
	})

	t.Run("url without a value is an error", func(t *testing.T) {
		_, err := parseAuthArgs([]string{"--url"})
		if err == nil {
			t.Fatal("expected an error for a missing --url value")
		}
		if !strings.Contains(err.Error(), "--url") {
			t.Errorf("expected the error to mention --url, got %v", err)
		}
	})
}

// --- site URL resolution ---

func TestResolveSiteURL(t *testing.T) {
	noEnv := func(string) string { return "" }
	envWith := func(value string) func(string) string {
		return func(key string) string {
			if key == "JIRA_URL" {
				return value
			}
			return ""
		}
	}

	t.Run("flag wins over environment", func(t *testing.T) {
		got, err := resolveSiteURL("https://from-flag.atlassian.net", envWith("https://from-env.atlassian.net"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "https://from-flag.atlassian.net" {
			t.Errorf("expected flag URL, got %q", got)
		}
	})

	t.Run("environment fallback", func(t *testing.T) {
		got, err := resolveSiteURL("", envWith("https://from-env.atlassian.net"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "https://from-env.atlassian.net" {
			t.Errorf("expected environment URL, got %q", got)
		}
	})

	t.Run("trailing slash is trimmed", func(t *testing.T) {
		got, err := resolveSiteURL("https://x.atlassian.net/", noEnv)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "https://x.atlassian.net" {
			t.Errorf("expected trimmed URL, got %q", got)
		}
	})

	t.Run("absent is not an error", func(t *testing.T) {
		got, err := resolveSiteURL("", noEnv)
		if err != nil {
			t.Fatalf("expected no error when no site URL is available, got %v", err)
		}
		if got != "" {
			t.Errorf("expected empty URL, got %q", got)
		}
	})

	t.Run("nil getenv is tolerated", func(t *testing.T) {
		got, err := resolveSiteURL("", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("expected empty URL, got %q", got)
		}
	})

	t.Run("missing scheme is an error", func(t *testing.T) {
		_, err := resolveSiteURL("magmalabs.atlassian.net", noEnv)
		if err == nil {
			t.Fatal("expected an error for a URL without a scheme")
		}
		if !strings.Contains(err.Error(), "http://") {
			t.Errorf("expected the error to mention the scheme, got %v", err)
		}
	})
}

// --- auth diagnostics ---

func TestAuthLoginError_UsesGuidanceOnTimeout(t *testing.T) {
	err := authLoginError(auth.ErrCallbackTimeout, "https://magmalabs.atlassian.net")

	msg := err.Error()
	if !strings.Contains(msg, "JIRA_OAUTH_CLIENT_ID") {
		t.Errorf("expected guidance to mention JIRA_OAUTH_CLIENT_ID, got %q", msg)
	}
	if !strings.Contains(msg, "JIRA_API_TOKEN") {
		t.Errorf("expected guidance to mention JIRA_API_TOKEN, got %q", msg)
	}
	if !strings.Contains(msg, "https://magmalabs.atlassian.net") {
		t.Errorf("expected guidance to mention the site, got %q", msg)
	}
}

func TestAuthLoginError_WrapsTimeoutSentinel(t *testing.T) {
	base := fmt.Errorf("oauth login: %w", auth.ErrCallbackTimeout)
	if !errors.Is(base, auth.ErrCallbackTimeout) {
		t.Fatal("expected wrapped sentinel to be detected")
	}

	err := authLoginError(base, "")
	if !strings.Contains(err.Error(), "JIRA_OAUTH_CLIENT_ID") {
		t.Errorf("expected guidance, got %q", err.Error())
	}
}

func TestAuthLoginError_PlainFailureIsUnchanged(t *testing.T) {
	want := errors.New("access_denied")
	err := authLoginError(want, "https://x.atlassian.net")

	if !errors.Is(err, want) {
		t.Errorf("expected wrapped cause, got %v", err)
	}
	if !strings.Contains(err.Error(), "login failed") {
		t.Errorf("expected 'login failed' prefix, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "JIRA_OAUTH_CLIENT_ID") {
		t.Errorf("guidance must not be used for non-timeout errors, got %q", err.Error())
	}
}

func TestAuthTimeoutHelp_NoSiteURL(t *testing.T) {
	msg := authTimeoutHelp("")
	if !strings.Contains(msg, "JIRA_OAUTH_CLIENT_ID") || !strings.Contains(msg, "JIRA_API_TOKEN") {
		t.Errorf("expected both fallbacks in %q", msg)
	}
}

func TestDetectCLI_PositionalArgActivates(t *testing.T) {
	cliMode, args := detectCLI([]string{"search", "--jql", "project = PROJ"})

	if !cliMode {
		t.Fatal("expected CLI mode for a positional command")
	}
	if len(args) != 3 || args[0] != "search" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestDetectCLI_FlagsOnlyStayInServerMode(t *testing.T) {
	cliMode, args := detectCLI([]string{"--transport", "http"})

	if cliMode {
		t.Fatal("expected server mode when no positional command is present")
	}
	if len(args) != 2 {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestDetectCLI_ForceFlagStripsItself(t *testing.T) {
	cliMode, args := detectCLI([]string{"--cli", "--transport", "http"})

	if !cliMode {
		t.Fatal("expected --cli to force CLI mode")
	}
	for _, a := range args {
		if a == "--cli" {
			t.Errorf("--cli should be stripped from args, got %v", args)
		}
	}
}
