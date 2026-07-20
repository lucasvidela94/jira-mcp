package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/lucasvidela94/jira-mcp/internal/jira"
)

func (s *Server) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	jql, err := requiredString(args, "jql")
	if err != nil {
		return nil, err
	}

	opts := []jira.Option{}
	if maxResults, ok, err := optionalInt(args, "max_results"); err != nil {
		return nil, err
	} else if ok {
		opts = append(opts, jira.WithMaxResults(maxResults))
	}

	if token := optionalString(args, "next_page_token"); token != "" {
		opts = append(opts, jira.WithNextPageToken(token))
	}

	if fieldsArg, ok := args["fields"].([]any); ok {
		var fields []string
		for _, f := range fieldsArg {
			fields = append(fields, fmt.Sprint(f))
		}
		if len(fields) > 0 {
			opts = append(opts, jira.WithSearchFields(fields))
		}
	}

	result, err := s.client.Search(ctx, jql, opts...)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(result)
}

func (s *Server) handleGetIssue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	key, err := requiredString(args, "issue_key")
	if err != nil {
		return nil, err
	}

	var opts []jira.Option
	if fieldsArg, ok := args["fields"].([]any); ok {
		var fields []string
		for _, f := range fieldsArg {
			fields = append(fields, fmt.Sprint(f))
		}
		if len(fields) > 0 {
			opts = append(opts, jira.WithFields(fields))
		}
	}

	issue, err := s.client.GetIssue(ctx, key, opts...)
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

func (s *Server) handleListUsers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	query := optionalString(args, "query")

	opts := []jira.Option{}
	if maxResults, ok, err := optionalInt(args, "max_results"); err != nil {
		return nil, err
	} else if ok {
		opts = append(opts, jira.WithMaxResults(maxResults))
	}

	users, err := s.client.ListUsers(ctx, query, opts...)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(map[string]any{"users": users})
}

func (s *Server) handleListProjects(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projects, err := s.client.ListProjects(ctx)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(map[string]any{"projects": projects})
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
	return resultJSON(map[string]any{"transitions": transitions})
}
