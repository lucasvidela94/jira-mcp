package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/lucasvidela/jira-mcp/internal/jira"
)

type fakeJiraClient struct {
	searchFn          func(ctx context.Context, jql string, opts ...jira.Option) (*jira.SearchResult, error)
	getIssueFn        func(ctx context.Context, key string) (*jira.Issue, error)
	createIssueFn     func(ctx context.Context, req *jira.CreateIssueRequest) (*jira.Issue, error)
	updateIssueFn     func(ctx context.Context, key string, req *jira.UpdateIssueRequest) error
	transitionIssueFn func(ctx context.Context, key string, req *jira.TransitionIssueRequest) error
	assignIssueFn     func(ctx context.Context, key string, req *jira.AssignIssueRequest) error
	deleteIssueFn     func(ctx context.Context, key string) error
	listProjectsFn    func(ctx context.Context) ([]jira.Project, error)
	getTransitionsFn  func(ctx context.Context, key string) ([]jira.Transition, error)
	addCommentFn      func(ctx context.Context, key string, req *jira.AddCommentRequest) (*jira.CommentResponse, error)
	addWorklogFn      func(ctx context.Context, key string, req *jira.AddWorklogRequest) (*jira.WorklogResponse, error)
}

func (f *fakeJiraClient) Search(ctx context.Context, jql string, opts ...jira.Option) (*jira.SearchResult, error) {
	return f.searchFn(ctx, jql, opts...)
}
func (f *fakeJiraClient) GetIssue(ctx context.Context, key string) (*jira.Issue, error) {
	return f.getIssueFn(ctx, key)
}
func (f *fakeJiraClient) CreateIssue(ctx context.Context, req *jira.CreateIssueRequest) (*jira.Issue, error) {
	return f.createIssueFn(ctx, req)
}
func (f *fakeJiraClient) UpdateIssue(ctx context.Context, key string, req *jira.UpdateIssueRequest) error {
	return f.updateIssueFn(ctx, key, req)
}
func (f *fakeJiraClient) TransitionIssue(ctx context.Context, key string, req *jira.TransitionIssueRequest) error {
	return f.transitionIssueFn(ctx, key, req)
}
func (f *fakeJiraClient) AssignIssue(ctx context.Context, key string, req *jira.AssignIssueRequest) error {
	return f.assignIssueFn(ctx, key, req)
}
func (f *fakeJiraClient) DeleteIssue(ctx context.Context, key string) error {
	return f.deleteIssueFn(ctx, key)
}
func (f *fakeJiraClient) ListProjects(ctx context.Context) ([]jira.Project, error) {
	return f.listProjectsFn(ctx)
}
func (f *fakeJiraClient) GetTransitions(ctx context.Context, key string) ([]jira.Transition, error) {
	return f.getTransitionsFn(ctx, key)
}
func (f *fakeJiraClient) AddComment(ctx context.Context, key string, req *jira.AddCommentRequest) (*jira.CommentResponse, error) {
	return f.addCommentFn(ctx, key, req)
}
func (f *fakeJiraClient) AddWorklog(ctx context.Context, key string, req *jira.AddWorklogRequest) (*jira.WorklogResponse, error) {
	return f.addWorklogFn(ctx, key, req)
}

func newRequest(name string, args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}
}

func TestServer_ToolList(t *testing.T) {
	srv := NewServer(&fakeJiraClient{})
	tools := srv.mcp.ListTools()

	expected := []string{
		"jira_search",
		"jira_get_issue",
		"jira_create_issue",
		"jira_update_issue",
		"jira_transition_issue",
		"jira_assign_issue",
		"jira_delete_issue",
		"jira_list_projects",
		"jira_get_transitions",
		"jira_add_comment",
		"jira_add_worklog",
	}

	if len(tools) != len(expected) {
		t.Errorf("expected %d tools, got %d", len(expected), len(tools))
	}
	for _, name := range expected {
		if _, ok := tools[name]; !ok {
			t.Errorf("missing tool %s", name)
		}
	}
}

func TestHandleSearch(t *testing.T) {
	client := &fakeJiraClient{
		searchFn: func(ctx context.Context, jql string, opts ...jira.Option) (*jira.SearchResult, error) {
			if jql != "project = PROJ" {
				t.Errorf("unexpected jql %q", jql)
			}
			return &jira.SearchResult{Total: 1, Issues: []jira.Issue{{Key: "PROJ-1"}}}, nil
		},
	}
	srv := NewServer(client)

	result, err := srv.handleSearch(context.Background(), newRequest("jira_search", map[string]any{"jql": "project = PROJ"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %v", result)
	}

	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "PROJ-1") {
		t.Errorf("expected result to contain PROJ-1, got %s", text)
	}
}

func TestHandleSearch_MissingRequiredParam(t *testing.T) {
	client := &fakeJiraClient{}
	srv := NewServer(client)

	_, err := srv.handleSearch(context.Background(), newRequest("jira_search", map[string]any{}))
	if err == nil {
		t.Fatal("expected error for missing jql")
	}
}

func TestHandleGetIssue(t *testing.T) {
	client := &fakeJiraClient{
		getIssueFn: func(ctx context.Context, key string) (*jira.Issue, error) {
			return &jira.Issue{Key: key, ID: "10001", Fields: json.RawMessage(`{"summary":"Test"}`)}, nil
		},
	}
	srv := NewServer(client)

	result, err := srv.handleGetIssue(context.Background(), newRequest("jira_get_issue", map[string]any{"issue_key": "PROJ-1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "PROJ-1") {
		t.Errorf("expected result to contain PROJ-1, got %s", text)
	}
}

func TestHandleCreateIssue(t *testing.T) {
	client := &fakeJiraClient{
		createIssueFn: func(ctx context.Context, req *jira.CreateIssueRequest) (*jira.Issue, error) {
			if req.ProjectKey != "PROJ" || req.Summary != "New issue" {
				t.Errorf("unexpected request: %+v", req)
			}
			return &jira.Issue{Key: "PROJ-2", ID: "10002"}, nil
		},
	}
	srv := NewServer(client)

	result, err := srv.handleCreateIssue(context.Background(), newRequest("jira_create_issue", map[string]any{
		"project_key": "PROJ",
		"issue_type":  "Task",
		"summary":     "New issue",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "PROJ-2") {
		t.Errorf("expected result to contain PROJ-2, got %s", text)
	}
}

func TestHandleCreateIssue_MissingRequiredParam(t *testing.T) {
	client := &fakeJiraClient{}
	srv := NewServer(client)

	_, err := srv.handleCreateIssue(context.Background(), newRequest("jira_create_issue", map[string]any{"summary": "New issue"}))
	if err == nil {
		t.Fatal("expected error for missing project_key")
	}
}

func TestHandleUpdateIssue(t *testing.T) {
	client := &fakeJiraClient{
		updateIssueFn: func(ctx context.Context, key string, req *jira.UpdateIssueRequest) error {
			if key != "PROJ-1" || req.Summary != "Updated" {
				t.Errorf("unexpected request: %s %+v", key, req)
			}
			return nil
		},
	}
	srv := NewServer(client)

	_, err := srv.handleUpdateIssue(context.Background(), newRequest("jira_update_issue", map[string]any{"issue_key": "PROJ-1", "summary": "Updated"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleTransitionIssue(t *testing.T) {
	client := &fakeJiraClient{
		transitionIssueFn: func(ctx context.Context, key string, req *jira.TransitionIssueRequest) error {
			if req.TransitionID != "21" {
				t.Errorf("unexpected transition id %s", req.TransitionID)
			}
			return nil
		},
	}
	srv := NewServer(client)

	_, err := srv.handleTransitionIssue(context.Background(), newRequest("jira_transition_issue", map[string]any{"issue_key": "PROJ-1", "transition_id": "21"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleAssignIssue(t *testing.T) {
	client := &fakeJiraClient{
		assignIssueFn: func(ctx context.Context, key string, req *jira.AssignIssueRequest) error {
			if req.AccountID != "abc123" {
				t.Errorf("unexpected account id %s", req.AccountID)
			}
			return nil
		},
	}
	srv := NewServer(client)

	_, err := srv.handleAssignIssue(context.Background(), newRequest("jira_assign_issue", map[string]any{"issue_key": "PROJ-1", "account_id": "abc123"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleDeleteIssue(t *testing.T) {
	client := &fakeJiraClient{
		deleteIssueFn: func(ctx context.Context, key string) error {
			if key != "PROJ-1" {
				t.Errorf("unexpected key %s", key)
			}
			return nil
		},
	}
	srv := NewServer(client)

	_, err := srv.handleDeleteIssue(context.Background(), newRequest("jira_delete_issue", map[string]any{"issue_key": "PROJ-1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleListProjects(t *testing.T) {
	client := &fakeJiraClient{
		listProjectsFn: func(ctx context.Context) ([]jira.Project, error) {
			return []jira.Project{{Key: "PROJ", Name: "Project"}}, nil
		},
	}
	srv := NewServer(client)

	result, err := srv.handleListProjects(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "PROJ") {
		t.Errorf("expected result to contain PROJ, got %s", text)
	}
}

func TestHandleGetTransitions(t *testing.T) {
	client := &fakeJiraClient{
		getTransitionsFn: func(ctx context.Context, key string) ([]jira.Transition, error) {
			return []jira.Transition{{ID: "21", Name: "In Progress"}}, nil
		},
	}
	srv := NewServer(client)

	result, err := srv.handleGetTransitions(context.Background(), newRequest("jira_get_transitions", map[string]any{"issue_key": "PROJ-1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "In Progress") {
		t.Errorf("expected result to contain In Progress, got %s", text)
	}
}

func TestHandleAddComment(t *testing.T) {
	client := &fakeJiraClient{
		addCommentFn: func(ctx context.Context, key string, req *jira.AddCommentRequest) (*jira.CommentResponse, error) {
			if req.Body != "A comment" {
				t.Errorf("unexpected body %q", req.Body)
			}
			return &jira.CommentResponse{ID: "10010", Self: "https://jira/comment/10010"}, nil
		},
	}
	srv := NewServer(client)

	result, err := srv.handleAddComment(context.Background(), newRequest("jira_add_comment", map[string]any{"issue_key": "PROJ-1", "body": "A comment"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "10010") {
		t.Errorf("expected result to contain comment id, got %s", text)
	}
}

func TestHandleAddComment_EmptyBody(t *testing.T) {
	client := &fakeJiraClient{}
	srv := NewServer(client)

	result, err := srv.handleAddComment(context.Background(), newRequest("jira_add_comment", map[string]any{"issue_key": "PROJ-1", "body": ""}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for empty body")
	}
}

func TestHandleAddWorklog(t *testing.T) {
	client := &fakeJiraClient{
		addWorklogFn: func(ctx context.Context, key string, req *jira.AddWorklogRequest) (*jira.WorklogResponse, error) {
			if req.TimeSpent != "1h" {
				t.Errorf("unexpected time spent %q", req.TimeSpent)
			}
			return &jira.WorklogResponse{ID: "10020", Self: "https://jira/worklog/10020"}, nil
		},
	}
	srv := NewServer(client)

	result, err := srv.handleAddWorklog(context.Background(), newRequest("jira_add_worklog", map[string]any{"issue_key": "PROJ-1", "time_spent": "1h"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "10020") {
		t.Errorf("expected result to contain worklog id, got %s", text)
	}
}

func TestHandleJiraError_MapsToToolError(t *testing.T) {
	client := &fakeJiraClient{
		getIssueFn: func(ctx context.Context, key string) (*jira.Issue, error) {
			return nil, &jira.JiraError{StatusCode: 404, Message: "issue not found"}
		},
	}
	srv := NewServer(client)

	result, err := srv.handleGetIssue(context.Background(), newRequest("jira_get_issue", map[string]any{"issue_key": "PROJ-1"}))
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "not found") {
		t.Errorf("expected error message to contain 'not found', got %s", text)
	}
}

func TestHandleJiraError_DoesNotLeakToken(t *testing.T) {
	client := &fakeJiraClient{
		getIssueFn: func(ctx context.Context, key string) (*jira.Issue, error) {
			return nil, errors.New("request failed: JIRA_API_TOKEN=secret-token")
		},
	}
	srv := NewServer(client)

	result, err := srv.handleGetIssue(context.Background(), newRequest("jira_get_issue", map[string]any{"issue_key": "PROJ-1"}))
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if contains(text, "secret-token") {
		t.Errorf("error leaks token: %s", text)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
