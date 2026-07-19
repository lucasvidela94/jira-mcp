package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) handleGetDevelopmentInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := requiredString(request.GetArguments(), "issue_key")
	if err != nil {
		return nil, err
	}
	info, err := s.client.GetDevelopmentInfo(ctx, key)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(info)
}
