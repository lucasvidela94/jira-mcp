package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) handleListProjectVersions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := requiredString(request.GetArguments(), "project_key")
	if err != nil {
		return nil, err
	}
	versions, err := s.client.ListProjectVersions(ctx, key)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(map[string]any{"versions": versions})
}

func (s *Server) handleGetVersion(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := requiredString(request.GetArguments(), "version_id")
	if err != nil {
		return nil, err
	}
	version, err := s.client.GetVersion(ctx, id)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(version)
}
