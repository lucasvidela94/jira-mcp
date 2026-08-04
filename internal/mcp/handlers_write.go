package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/lucasvidela94/jira-mcp/internal/jira"
)

// parseDescription extracts a polymorphic description argument (string or
// pre-built ADF object) and returns it as json.RawMessage. The second return
// value is true if the caller supplied a description at all.
//
// When the value is a string, the returned RawMessage is the JSON-encoded
// string itself (e.g. `"hello"`), so the model layer can branch on the first
// byte to decide between plainTextToADF and verbatim embedding.
//
// When the value is a map, it is re-encoded unchanged (verbatim) and
// validated as a top-level ADF document (`type:"doc"`, `version:1`, `content`
// is an array). Inner node shapes are Jira's responsibility.
func parseDescription(args map[string]any, name string) (json.RawMessage, bool, error) {
	v, ok := args[name]
	if !ok {
		return nil, false, nil
	}
	switch val := v.(type) {
	case string:
		encoded, err := json.Marshal(val)
		if err != nil {
			return nil, true, fmt.Errorf("description: %w", err)
		}
		return encoded, true, nil
	case map[string]any:
		if err := validateTopLevelADF(val); err != nil {
			return nil, true, fmt.Errorf("description: %w", err)
		}
		encoded, err := json.Marshal(val)
		if err != nil {
			return nil, true, fmt.Errorf("description: %w", err)
		}
		return encoded, true, nil
	default:
		return nil, true, fmt.Errorf("description must be a string or an ADF object")
	}
}

// validateTopLevelADF checks the three required keys of an ADF document.
func validateTopLevelADF(obj map[string]any) error {
	if typ, _ := obj["type"].(string); typ != "doc" {
		return fmt.Errorf("ADF object must have type \"doc\"")
	}
	ver, ok := obj["version"]
	if !ok {
		return fmt.Errorf("ADF object must have version")
	}
	// JSON numbers arrive as float64 when decoded into map[string]any.
	switch n := ver.(type) {
	case float64:
		if n != 1 {
			return fmt.Errorf("ADF object must have version 1")
		}
	case int:
		if n != 1 {
			return fmt.Errorf("ADF object must have version 1")
		}
	default:
		return fmt.Errorf("ADF object version must be a number")
	}
	if _, ok := obj["content"]; !ok {
		return fmt.Errorf("ADF object must have content")
	}
	if _, isArray := obj["content"].([]any); !isArray {
		return fmt.Errorf("ADF object content must be an array")
	}
	return nil
}

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

	description, descPresent, err := parseDescription(args, "description")
	if err != nil {
		return resultError(err.Error()), nil
	}
	if descPresent && fields != nil {
		if _, has := fields["description"]; has {
			return resultError("description provided in both top-level argument and fields: choose one"), nil
		}
	}

	req := &jira.CreateIssueRequest{
		ProjectKey:  projectKey,
		IssueType:   issueType,
		Summary:     summary,
		Description: description,
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

	description, descPresent, err := parseDescription(args, "description")
	if err != nil {
		return resultError(err.Error()), nil
	}
	if descPresent && fields != nil {
		if _, has := fields["description"]; has {
			return resultError("description provided in both top-level argument and fields: choose one"), nil
		}
	}

	req := &jira.UpdateIssueRequest{
		Summary:     optionalString(args, "summary"),
		IssueType:   optionalString(args, "issue_type"),
		Description: description,
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
