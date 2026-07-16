package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/lucasvidela/jira-mcp/internal/jira"
)

func (s *Server) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	jql, err := requiredString(args, "jql")
	if err != nil {
		return nil, err
	}

	opts := []jira.Option{}
	if startAt, ok, err := optionalInt(args, "startAt"); err != nil {
		return nil, err
	} else if ok {
		opts = append(opts, jira.WithStartAt(startAt))
	}
	if maxResults, ok, err := optionalInt(args, "maxResults"); err != nil {
		return nil, err
	} else if ok {
		opts = append(opts, jira.WithMaxResults(maxResults))
	}

	result, err := s.client.Search(ctx, jql, opts...)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(result)
}

func (s *Server) handleGetIssue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := requiredString(request.GetArguments(), "issue_key")
	if err != nil {
		return nil, err
	}

	issue, err := s.client.GetIssue(ctx, key)
	if err != nil {
		return handleClientError(err), nil
	}

	// Decode raw fields so the output is readable.
	var fields map[string]any
	if issue.Fields != nil {
		_ = json.Unmarshal(issue.Fields, &fields)
	}

	return resultJSON(map[string]any{
		"id":     issue.ID,
		"key":    issue.Key,
		"self":   issue.Self,
		"fields": fields,
	})
}

func (s *Server) handleListProjects(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projects, err := s.client.ListProjects(ctx)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(projects)
}

func (s *Server) handleGetTransitions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := requiredString(request.GetArguments(), "issue_key")
	if err != nil {
		return nil, err
	}

	transitions, err := s.client.GetTransitions(ctx, key)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(transitions)
}
