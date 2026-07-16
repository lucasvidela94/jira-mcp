package mcp

import (
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/lucasvidela/jira-mcp/internal/jira"
)

// requiredString extracts a required string argument from the request.
func requiredString(args map[string]any, name string) (string, error) {
	v, ok := args[name]
	if !ok {
		return "", fmt.Errorf("missing required parameter: %s", name)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("parameter %s must be a string", name)
	}
	return s, nil
}

// optionalString extracts an optional string argument.
func optionalString(args map[string]any, name string) string {
	v, ok := args[name]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// optionalObject extracts an optional JSON object argument.
func optionalObject(args map[string]any, name string) (map[string]any, error) {
	v, ok := args[name]
	if !ok {
		return nil, nil
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("parameter %s must be an object", name)
	}
	return obj, nil
}

// optionalInt extracts an optional integer argument.
func optionalInt(args map[string]any, name string) (int, bool, error) {
	v, ok := args[name]
	if !ok {
		return 0, false, nil
	}
	switch n := v.(type) {
	case int:
		return n, true, nil
	case int64:
		return int(n), true, nil
	case float64:
		return int(n), true, nil
	default:
		return 0, false, fmt.Errorf("parameter %s must be a number", name)
	}
}

// resultJSON returns a successful CallToolResult with JSON content.
func resultJSON(data any) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultJSON(data)
}

// resultText returns a successful CallToolResult with plain text.
func resultText(text string) *mcp.CallToolResult {
	return mcp.NewToolResultText(text)
}

// resultError returns a CallToolResult marked as an error.
func resultError(msg string) *mcp.CallToolResult {
	// Ensure no credential-like strings leak through.
	msg = sanitizeError(msg)
	return mcp.NewToolResultError(msg)
}

// sanitizeError removes known credential placeholders and their values from error messages.
func sanitizeError(msg string) string {
	for _, secret := range []string{"JIRA_API_TOKEN", "JIRA_USERNAME", "JIRA_URL"} {
		// Replace `KEY=value` patterns and the bare key name.
		for {
			idx := strings.Index(msg, secret)
			if idx == -1 {
				break
			}
			end := idx + len(secret)
			if end < len(msg) && msg[end] == '=' {
				// Find the end of the value (next space or end of string).
				valEnd := strings.IndexAny(msg[end:], " \t\n\r")
				if valEnd == -1 {
					valEnd = len(msg)
				} else {
					valEnd += end
				}
				msg = msg[:idx] + "<redacted>" + msg[valEnd:]
			} else {
				msg = msg[:idx] + "<redacted>" + msg[end:]
			}
		}
	}
	return msg
}

// handleClientError maps client errors to MCP tool results.
func handleClientError(err error) *mcp.CallToolResult {
	if jerr, ok := err.(*jira.JiraError); ok {
		return resultError(jerr.Message)
	}
	return resultError(err.Error())
}
