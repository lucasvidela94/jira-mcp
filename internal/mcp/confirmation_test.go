package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

type fakeElicitor struct {
	result *mcp.ElicitationResult
	err    error
}

func (f *fakeElicitor) RequestElicitation(ctx context.Context, request mcp.ElicitationRequest) (*mcp.ElicitationResult, error) {
	return f.result, f.err
}

func TestIsDestructiveTool(t *testing.T) {
	tests := []struct {
		name string
		tool string
		want bool
	}{
		{"create issue", "jira_create_issue", true},
		{"create child issue", "jira_create_child_issue", true},
		{"delete issue", "jira_delete_issue", true},
		{"search", "jira_search", false},
		{"get issue", "jira_get_issue", false},
		{"update issue", "jira_update_issue", false},
		{"create issue link", "jira_create_issue_link", false},
		{"add comment", "jira_add_comment", false},
		{"unknown", "jira_does_not_exist", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDestructiveTool(tt.tool); got != tt.want {
				t.Errorf("isDestructiveTool(%q) = %v, want %v", tt.tool, got, tt.want)
			}
		})
	}
}

func TestConfirmationMiddleware(t *testing.T) {
	tests := []struct {
		name          string
		confirmWrite  bool
		tool          string
		elicitor      *fakeElicitor
		confirm       any
		wantNext      bool
		wantIsError   bool
		wantTextMatch string
	}{
		{
			name:         "confirmation disabled bypasses elicitation",
			confirmWrite: false,
			tool:         "jira_create_issue",
			elicitor:     &fakeElicitor{},
			wantNext:     true,
		},
		{
			name:         "non-destructive tool bypasses elicitation",
			confirmWrite: true,
			tool:         "jira_search",
			elicitor:     &fakeElicitor{},
			wantNext:     true,
		},
		{
			name:         "destructive tool accepted runs next",
			confirmWrite: true,
			tool:         "jira_create_issue",
			elicitor: &fakeElicitor{result: &mcp.ElicitationResult{
				ElicitationResponse: mcp.ElicitationResponse{Action: mcp.ElicitationResponseActionAccept},
			}},
			wantNext: true,
		},
		{
			name:         "destructive tool declined blocks next",
			confirmWrite: true,
			tool:         "jira_delete_issue",
			elicitor: &fakeElicitor{result: &mcp.ElicitationResult{
				ElicitationResponse: mcp.ElicitationResponse{Action: mcp.ElicitationResponseActionDecline},
			}},
			wantNext:      false,
			wantTextMatch: "Cancelled",
		},
		{
			name:          "elicitation error without confirm fails closed",
			confirmWrite:  true,
			tool:          "jira_create_child_issue",
			elicitor:      &fakeElicitor{err: errors.New("session does not support elicitation")},
			wantNext:      false,
			wantIsError:   true,
			wantTextMatch: "confirm=true",
		},
		{
			name:         "elicitation error with confirm=true proceeds",
			confirmWrite: true,
			tool:         "jira_create_child_issue",
			elicitor:     &fakeElicitor{err: errors.New("session does not support elicitation")},
			confirm:      true,
			wantNext:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(&fakeJiraClient{}, nil)
			s.confirmWrite = tt.confirmWrite
			s.elicitor = tt.elicitor

			nextCalled := false
			next := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				nextCalled = true
				return resultText("done"), nil
			}

			args := map[string]any{"project_key": "PROJ"}
			if tt.confirm != nil {
				args["confirm"] = tt.confirm
			}
			result, err := s.confirmationMiddleware(next)(context.Background(), newRequest(tt.tool, args))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if nextCalled != tt.wantNext {
				t.Errorf("next called = %v, want %v", nextCalled, tt.wantNext)
			}
			if tt.wantIsError != (result != nil && result.IsError) {
				t.Errorf("IsError = %v, want %v", result != nil && result.IsError, tt.wantIsError)
			}
			if tt.wantTextMatch != "" {
				if result == nil || len(result.Content) == 0 {
					t.Fatalf("expected text content matching %q, got no result content", tt.wantTextMatch)
				}
				text := result.Content[0].(mcp.TextContent).Text
				if !contains(text, tt.wantTextMatch) {
					t.Errorf("result text %q does not contain %q", text, tt.wantTextMatch)
				}
			}
		})
	}
}
