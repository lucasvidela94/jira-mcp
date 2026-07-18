package mcp

import (
	"testing"

	"github.com/lucasvidela94/jira-mcp/internal/jira"
)

func TestJiraClientInterface_SprintMethods(t *testing.T) {
	// Compile-time assertion that the concrete client satisfies the interface,
	// including the four sprint methods.
	var _ JiraClient = (*jira.Client)(nil)
}
