package mcp

import (
	"context"
	"fmt"

	"github.com/lucasvidela94/jira-mcp/internal/jira"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) handleCreateIssueLink(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	linkType, err := requiredString(args, "link_type")
	if err != nil {
		return nil, err
	}
	inwardKey, err := requiredString(args, "inward_issue_key")
	if err != nil {
		return nil, err
	}
	outwardKey, err := requiredString(args, "outward_issue_key")
	if err != nil {
		return nil, err
	}

	req := &jira.IssueLinkRequest{
		LinkTypeName: linkType,
		InwardKey:    inwardKey,
		OutwardKey:   outwardKey,
	}
	if err := s.client.CreateIssueLink(ctx, req); err != nil {
		return handleClientError(err), nil
	}
	return resultText(fmt.Sprintf("Link created between %s and %s", inwardKey, outwardKey)), nil
}

func (s *Server) handleGetRelatedIssues(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := requiredString(request.GetArguments(), "issue_key")
	if err != nil {
		return nil, err
	}
	issues, err := s.client.GetRelatedIssues(ctx, key)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(map[string]any{"related_issues": issues})
}
