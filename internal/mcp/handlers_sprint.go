package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/lucasvidela94/jira-mcp/internal/jira"
)

func (s *Server) handleListSprints(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	boardID, err := requiredString(args, "board_id")
	if err != nil {
		return nil, err
	}

	opts := []jira.Option{}
	if maxResults, ok, err := optionalInt(args, "max_results"); err != nil {
		return nil, err
	} else if ok {
		opts = append(opts, jira.WithMaxResults(maxResults))
	}

	sprints, err := s.client.ListSprints(ctx, boardID, opts...)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(map[string]any{"sprints": sprints})
}

func (s *Server) handleGetSprint(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	sprintID, err := requiredString(args, "sprint_id")
	if err != nil {
		return nil, err
	}

	sprint, err := s.client.GetSprint(ctx, sprintID)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(sprint)
}

func (s *Server) handleGetActiveSprint(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	boardID, err := requiredString(args, "board_id")
	if err != nil {
		return nil, err
	}

	opts := []jira.Option{}
	if maxResults, ok, err := optionalInt(args, "max_results"); err != nil {
		return nil, err
	} else if ok {
		opts = append(opts, jira.WithMaxResults(maxResults))
	}

	sprint, err := s.client.GetActiveSprint(ctx, boardID, opts...)
	if err != nil {
		return handleClientError(err), nil
	}
	if sprint == nil {
		return resultText("No active sprint found for this board"), nil
	}
	return resultJSON(sprint)
}

func (s *Server) handleSearchSprintByName(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	boardID, err := requiredString(args, "board_id")
	if err != nil {
		return nil, err
	}
	name, err := requiredString(args, "name")
	if err != nil {
		return nil, err
	}

	opts := []jira.Option{}
	if maxResults, ok, err := optionalInt(args, "max_results"); err != nil {
		return nil, err
	} else if ok {
		opts = append(opts, jira.WithMaxResults(maxResults))
	}

	sprints, err := s.client.SearchSprintByName(ctx, boardID, name, opts...)
	if err != nil {
		return handleClientError(err), nil
	}
	return resultJSON(map[string]any{"sprints": sprints})
}
