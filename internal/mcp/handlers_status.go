package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) handleListStatuses(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := requiredString(request.GetArguments(), "project_key")
	if err != nil {
		return nil, err
	}
	statuses, err := s.client.ListStatuses(ctx, key)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(map[string]any{"statuses": statuses})
}
