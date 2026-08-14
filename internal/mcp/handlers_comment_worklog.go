package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/lucasvidela94/jira-mcp/internal/jira"
)

func (s *Server) handleAddComment(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	key, err := requiredString(args, "issue_key")
	if err != nil {
		return nil, err
	}
	body, bodyPresent, err := parseDescription(args, "body")
	if err != nil {
		return resultError(err.Error()), nil
	}
	if !bodyPresent || isBlankBody(body) {
		return resultError("comment body cannot be empty"), nil
	}

	comment, err := s.client.AddComment(ctx, key, &jira.AddCommentRequest{Body: body})
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(comment)
}

// isBlankBody reports whether a polymorphic comment body is empty or only
// whitespace. The body arrives as json.RawMessage: a plain string is encoded
// with quotes, while a pre-built ADF object is never blank.
func isBlankBody(body json.RawMessage) bool {
	if len(body) == 0 {
		return true
	}
	if body[0] == '"' {
		var s string
		if err := json.Unmarshal(body, &s); err == nil {
			return strings.TrimSpace(s) == ""
		}
	}
	return false
}

func (s *Server) handleAddWorklog(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	key, err := requiredString(args, "issue_key")
	if err != nil {
		return nil, err
	}
	timeSpent, err := requiredString(args, "time_spent")
	if err != nil {
		return nil, err
	}

	req := &jira.AddWorklogRequest{
		TimeSpent: timeSpent,
		Comment:   optionalString(args, "comment"),
		Started:   optionalString(args, "started"),
	}

	worklog, err := s.client.AddWorklog(ctx, key, req)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(worklog)
}
