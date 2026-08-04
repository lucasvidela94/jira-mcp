package mcp

import (
	"context"

	"github.com/lucasvidela94/jira-mcp/internal/jira"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) handleCreateChildIssue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	parentKey, err := requiredString(args, "parent_key")
	if err != nil {
		return nil, err
	}
	projectKey, err := requiredString(args, "project_key")
	if err != nil {
		return nil, err
	}
	issueType, err := requiredString(args, "issue_type")
	if err != nil {
		return nil, err
	}
	summary, err := requiredString(args, "summary")
	if err != nil {
		return nil, err
	}
	fields, err := optionalObject(args, "fields")
	if err != nil {
		return nil, err
	}

	description, descPresent, err := parseDescription(args, "description")
	if err != nil {
		return resultError(err.Error()), nil
	}
	if descPresent && fields != nil {
		if _, has := fields["description"]; has {
			return resultError("description provided in both top-level argument and fields: choose one"), nil
		}
	}

	req := jira.CreateChildIssueRequest{
		ParentKey:   parentKey,
		ProjectKey:  projectKey,
		IssueType:   issueType,
		Summary:     summary,
		Description: description,
		Fields:      fields,
	}
	issue, err := s.client.CreateChildIssue(ctx, parentKey, &req)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(issue)
}
