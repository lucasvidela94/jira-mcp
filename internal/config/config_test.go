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
}

func TestLoad_MissingVariable(t *testing.T) {
	// Ensure all three are unset initially.
	unsetenv(t, "JIRA_URL")
	unsetenv(t, "JIRA_USERNAME")
	unsetenv(t, "JIRA_API_TOKEN")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing env vars")
	}
	// The error must name the missing variable without exposing any value.
	if err.Error() != "missing required environment variable: JIRA_URL" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLoad_MissingToken(t *testing.T) {
	setenv(t, "JIRA_URL", "https://example.atlassian.net")
	setenv(t, "JIRA_USERNAME", "user@example.com")
	unsetenv(t, "JIRA_API_TOKEN")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if err.Error() != "missing required environment variable: JIRA_API_TOKEN" {
		t.Errorf("unexpected error message: %v", err)
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
