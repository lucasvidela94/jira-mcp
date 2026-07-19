package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/lucasvidela94/jira-mcp/internal/jira"
)

func (s *Server) handleListBoards(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	opts := []jira.Option{}
	if projectKey := optionalString(args, "project_key"); projectKey != "" {
		opts = append(opts, jira.WithProjectKey(projectKey))
	}

	boards, err := s.client.ListBoards(ctx, opts...)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(map[string]any{"boards": boards})
}
