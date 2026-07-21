package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/lucasvidela94/jira-mcp/internal/jira"
)

func (s *Server) handleCreateIssue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
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

	req := &jira.CreateIssueRequest{
		ProjectKey:  projectKey,
		IssueType:   issueType,
		Summary:     summary,
		Description: optionalString(args, "description"),
		Fields:      fields,
	}

	issue, err := s.client.CreateIssue(ctx, req)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(issue)
}

func (s *Server) handleUpdateIssue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	key, err := requiredString(args, "issue_key")
	if err != nil {
		return nil, err
	}

	fields, err := optionalObject(args, "fields")
	if err != nil {
		return nil, err
	}

	req := &jira.UpdateIssueRequest{
		Summary:     optionalString(args, "summary"),
		IssueType:   optionalString(args, "issue_type"),
		Description: optionalString(args, "description"),
		Fields:      fields,
	}

	if err := s.client.UpdateIssue(ctx, key, req); err != nil {
		return handleClientError(err), nil
	}
	return resultText(fmt.Sprintf("Issue %s updated successfully", key)), nil
}

func (s *Server) handleTransitionIssue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	key, err := requiredString(args, "issue_key")
	if err != nil {
		return nil, err
	}
	transitionID, err := requiredString(args, "transition_id")
	if err != nil {
		return nil, err
	}

	if err := s.client.TransitionIssue(ctx, key, &jira.TransitionIssueRequest{TransitionID: transitionID}); err != nil {
		return handleClientError(err), nil
	}
	return resultText(fmt.Sprintf("Issue %s transitioned successfully", key)), nil
}

func (s *Server) handleAssignIssue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	key, err := requiredString(args, "issue_key")
	if err != nil {
		return nil, err
	}
	accountID, err := requiredString(args, "account_id")
	if err != nil {
		return nil, err
	}

	if err := s.client.AssignIssue(ctx, key, &jira.AssignIssueRequest{AccountID: accountID}); err != nil {
		return handleClientError(err), nil
	}
	return resultText(fmt.Sprintf("Issue %s assigned successfully", key)), nil
}

func (s *Server) handleDeleteIssue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := requiredString(request.GetArguments(), "issue_key")
	if err != nil {
		return nil, err
	}

	if err := s.client.DeleteIssue(ctx, key); err != nil {
		return handleClientError(err), nil
	}
	return resultText(fmt.Sprintf("Issue %s deleted successfully", key)), nil
}
