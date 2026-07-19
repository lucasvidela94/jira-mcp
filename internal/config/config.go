package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds Jira Cloud credentials sourced from environment variables.
type Config struct {
	URL          string   // JIRA_URL
	Username     string   // JIRA_USERNAME
	APIToken     string   // JIRA_API_TOKEN
	EnabledTools []string // ENABLED_TOOLS (comma-separated allowlist)
}

// Load reads Jira configuration from environment variables.
// It returns an error naming the first missing required variable.
func Load() (Config, error) {
	cfg := Config{
		URL:      os.Getenv("JIRA_URL"),
		Username: os.Getenv("JIRA_USERNAME"),
		APIToken: os.Getenv("JIRA_API_TOKEN"),
	}

	for _, field := range []struct {
		name  string
		value string
	}{
		{"JIRA_URL", cfg.URL},
		{"JIRA_USERNAME", cfg.Username},
		{"JIRA_API_TOKEN", cfg.APIToken},
	} {
		if field.value == "" {
			return Config{}, fmt.Errorf("missing required environment variable: %s", field.name)
		}
	}

	if raw := os.Getenv("ENABLED_TOOLS"); raw != "" {
		cfg.EnabledTools = splitAndTrim(raw, ",")
	}

	return cfg, nil
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
