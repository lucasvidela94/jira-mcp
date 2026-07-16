package mcp

import (
	"context"
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
	body, err := requiredString(args, "body")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(body) == "" {
		return resultError("comment body cannot be empty"), nil
	}

	comment, err := s.client.AddComment(ctx, key, &jira.AddCommentRequest{Body: body})
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(comment)
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
