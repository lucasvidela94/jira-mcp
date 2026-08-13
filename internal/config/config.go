package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds Jira Cloud configuration sourced from environment variables.
type Config struct {
	URL          string   // JIRA_URL
	Username     string   // JIRA_USERNAME
	APIToken     string   // JIRA_API_TOKEN
	AuthMode     string   // "basic" when JIRA_API_TOKEN is set, "oauth" otherwise
	EnabledTools []string // ENABLED_TOOLS (comma-separated allowlist)
	ConfirmWrite bool     // JIRA_MCP_CONFIRM (whether destructive write tools require confirmation; default on)
}

// Load reads Jira configuration from environment variables.
// If JIRA_API_TOKEN is set, all three credentials are required (Basic mode).
// Otherwise, it returns without error for OAuth mode (token from disk).
func Load() (Config, error) {
	cfg := Config{
		URL:          os.Getenv("JIRA_URL"),
		Username:     os.Getenv("JIRA_USERNAME"),
		APIToken:     os.Getenv("JIRA_API_TOKEN"),
		ConfirmWrite: true,
	}

	switch strings.ToLower(strings.TrimSpace(os.Getenv("JIRA_MCP_CONFIRM"))) {
	case "off", "false", "0", "no":
		cfg.ConfirmWrite = false
	}

	if cfg.APIToken != "" {
		cfg.AuthMode = "basic"
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
	} else {
		cfg.AuthMode = "oauth"
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
