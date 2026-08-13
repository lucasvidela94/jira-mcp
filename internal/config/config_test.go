package config

import (
	"os"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	setenv(t, "JIRA_URL", "https://example.atlassian.net")
	setenv(t, "JIRA_USERNAME", "user@example.com")
	setenv(t, "JIRA_API_TOKEN", "secret-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.URL != "https://example.atlassian.net" {
		t.Errorf("expected URL https://example.atlassian.net, got %q", cfg.URL)
	}
	if cfg.Username != "user@example.com" {
		t.Errorf("expected Username user@example.com, got %q", cfg.Username)
	}
	if cfg.APIToken != "secret-token" {
		t.Errorf("expected APIToken secret-token, got %q", cfg.APIToken)
	}
	if cfg.AuthMode != "basic" {
		t.Errorf("expected AuthMode basic, got %q", cfg.AuthMode)
	}
}

func TestLoad_OAuthMode_NoEnvVars(t *testing.T) {
	unsetenv(t, "JIRA_URL")
	unsetenv(t, "JIRA_USERNAME")
	unsetenv(t, "JIRA_API_TOKEN")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error in OAuth mode, got %v", err)
	}
	if cfg.AuthMode != "oauth" {
		t.Errorf("expected AuthMode oauth, got %q", cfg.AuthMode)
	}
}

func TestLoad_BasicMode_MissingRequired(t *testing.T) {
	setenv(t, "JIRA_API_TOKEN", "some-token")
	unsetenv(t, "JIRA_URL")
	unsetenv(t, "JIRA_USERNAME")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when JIRA_API_TOKEN is set but JIRA_URL is missing")
	}
	if err.Error() != "missing required environment variable: JIRA_URL" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLoad_MissingToken(t *testing.T) {
	setenv(t, "JIRA_URL", "https://example.atlassian.net")
	setenv(t, "JIRA_USERNAME", "user@example.com")
	unsetenv(t, "JIRA_API_TOKEN")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error when token is missing (OAuth mode), got %v", err)
	}
	if cfg.AuthMode != "oauth" {
		t.Errorf("expected AuthMode oauth, got %q", cfg.AuthMode)
	}
}

func TestLoad_ConfirmWriteDefault(t *testing.T) {
	unsetenv(t, "JIRA_MCP_CONFIRM")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.ConfirmWrite {
		t.Error("expected ConfirmWrite to default to true")
	}
}

func TestLoad_ConfirmWriteOff(t *testing.T) {
	setenv(t, "JIRA_MCP_CONFIRM", "off")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.ConfirmWrite {
		t.Error("expected ConfirmWrite false for JIRA_MCP_CONFIRM=off")
	}
}

func TestLoad_ConfirmWriteCaseInsensitive(t *testing.T) {
	for _, value := range []string{"OFF", "False"} {
		t.Run(value, func(t *testing.T) {
			setenv(t, "JIRA_MCP_CONFIRM", value)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if cfg.ConfirmWrite {
				t.Errorf("expected ConfirmWrite false for JIRA_MCP_CONFIRM=%s", value)
			}
		})
	}
}

func TestLoad_ConfirmWriteInvalidStaysOn(t *testing.T) {
	setenv(t, "JIRA_MCP_CONFIRM", "banana")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.ConfirmWrite {
		t.Error("expected ConfirmWrite true for invalid JIRA_MCP_CONFIRM value")
	}
}

func setenv(t *testing.T, key, value string) {
	t.Helper()
	os.Setenv(key, value)
	t.Cleanup(func() { os.Unsetenv(key) })
}

func unsetenv(t *testing.T, key string) {
	t.Helper()
	os.Unsetenv(key)
	t.Cleanup(func() { os.Unsetenv(key) })
}
