package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/lucasvidela94/jira-mcp/internal/jira"
)

type fakeJiraClient struct {
	searchFn              func(ctx context.Context, jql string, opts ...jira.Option) (*jira.SearchResult, error)
	getIssueFn            func(ctx context.Context, key string) (*jira.Issue, error)
	createIssueFn         func(ctx context.Context, req *jira.CreateIssueRequest) (*jira.Issue, error)
	updateIssueFn         func(ctx context.Context, key string, req *jira.UpdateIssueRequest) error
	transitionIssueFn     func(ctx context.Context, key string, req *jira.TransitionIssueRequest) error
	assignIssueFn         func(ctx context.Context, key string, req *jira.AssignIssueRequest) error
	deleteIssueFn         func(ctx context.Context, key string) error
	listProjectsFn        func(ctx context.Context) ([]jira.Project, error)
	getTransitionsFn      func(ctx context.Context, key string) ([]jira.Transition, error)
	addCommentFn          func(ctx context.Context, key string, req *jira.AddCommentRequest) (*jira.CommentResponse, error)
	addWorklogFn          func(ctx context.Context, key string, req *jira.AddWorklogRequest) (*jira.WorklogResponse, error)
	listBoardsFn          func(ctx context.Context, opts ...jira.Option) (jira.BoardList, error)
	listSprintsFn         func(ctx context.Context, boardID string, opts ...jira.Option) (jira.SprintList, error)
	getSprintFn           func(ctx context.Context, sprintID string) (*jira.Sprint, error)
	getActiveSprintFn     func(ctx context.Context, boardID string, opts ...jira.Option) (*jira.Sprint, error)
	searchSprintByNameFn  func(ctx context.Context, boardID string, name string, opts ...jira.Option) (jira.SprintList, error)
	getIssueHistoryFn     func(ctx context.Context, key string) (*jira.Changelog, error)
	listProjectVersionsFn func(ctx context.Context, projectKey string) ([]jira.Version, error)
	getVersionFn          func(ctx context.Context, id string) (*jira.Version, error)
	getDevelopmentInfoFn  func(ctx context.Context, key string) (*jira.DevelopmentInformation, error)
	listStatusesFn        func(ctx context.Context, projectKey string) ([]jira.ProjectStatus, error)
	createIssueLinkFn     func(ctx context.Context, req *jira.IssueLinkRequest) error
	getRelatedIssuesFn    func(ctx context.Context, key string) ([]jira.LinkedIssue, error)
	createChildIssueFn    func(ctx context.Context, parentKey string, req *jira.CreateChildIssueRequest) (*jira.Issue, error)
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
func (f *fakeJiraClient) ListBoards(ctx context.Context, opts ...jira.Option) (jira.BoardList, error) {
	return f.listBoardsFn(ctx, opts...)
}
func (f *fakeJiraClient) ListSprints(ctx context.Context, boardID string, opts ...jira.Option) (jira.SprintList, error) {
	return f.listSprintsFn(ctx, boardID, opts...)
}
func (f *fakeJiraClient) GetSprint(ctx context.Context, sprintID string) (*jira.Sprint, error) {
	return f.getSprintFn(ctx, sprintID)
}
func (f *fakeJiraClient) GetActiveSprint(ctx context.Context, boardID string, opts ...jira.Option) (*jira.Sprint, error) {
	return f.getActiveSprintFn(ctx, boardID, opts...)
}
func (f *fakeJiraClient) SearchSprintByName(ctx context.Context, boardID string, name string, opts ...jira.Option) (jira.SprintList, error) {
	return f.searchSprintByNameFn(ctx, boardID, name, opts...)
}

func (f *fakeJiraClient) GetIssueHistory(ctx context.Context, key string) (*jira.Changelog, error) {
	return f.getIssueHistoryFn(ctx, key)
}

func (f *fakeJiraClient) ListProjectVersions(ctx context.Context, projectKey string) ([]jira.Version, error) {
	return f.listProjectVersionsFn(ctx, projectKey)
}

func (f *fakeJiraClient) GetVersion(ctx context.Context, id string) (*jira.Version, error) {
	return f.getVersionFn(ctx, id)
}

func (f *fakeJiraClient) GetDevelopmentInfo(ctx context.Context, key string) (*jira.DevelopmentInformation, error) {
	return f.getDevelopmentInfoFn(ctx, key)
}

func (f *fakeJiraClient) ListStatuses(ctx context.Context, projectKey string) ([]jira.ProjectStatus, error) {
	return f.listStatusesFn(ctx, projectKey)
}

func (f *fakeJiraClient) CreateIssueLink(ctx context.Context, req *jira.IssueLinkRequest) error {
	return f.createIssueLinkFn(ctx, req)
}

func (f *fakeJiraClient) GetRelatedIssues(ctx context.Context, key string) ([]jira.LinkedIssue, error) {
	return f.getRelatedIssuesFn(ctx, key)
}

func (f *fakeJiraClient) CreateChildIssue(ctx context.Context, parentKey string, req *jira.CreateChildIssueRequest) (*jira.Issue, error) {
	return f.createChildIssueFn(ctx, parentKey, req)
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
	srv := NewServer(&fakeJiraClient{}, nil)
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
		"jira_list_boards",
		"jira_list_sprints",
		"jira_get_sprint",
		"jira_get_active_sprint",
		"jira_search_sprint_by_name",
		"jira_get_issue_history",
		"jira_list_project_versions",
		"jira_get_version",
		"jira_get_development_info",
		"jira_list_statuses",
		"jira_create_issue_link",
		"jira_get_related_issues",
		"jira_create_child_issue",
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
	srv := NewServer(client, nil)

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
	srv := NewServer(client, nil)

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
	srv := NewServer(client, nil)

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
	srv := NewServer(client, nil)

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
	srv := NewServer(client, nil)

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
	srv := NewServer(client, nil)

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
	srv := NewServer(client, nil)

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
	srv := NewServer(client, nil)

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
	srv := NewServer(client, nil)

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
	srv := NewServer(client, nil)

	result, err := srv.handleListProjects(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "\"projects\"") {
		t.Errorf("expected result to wrap projects, got %s", text)
	}
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
	srv := NewServer(client, nil)

	result, err := srv.handleGetTransitions(context.Background(), newRequest("jira_get_transitions", map[string]any{"issue_key": "PROJ-1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "\"transitions\"") {
		t.Errorf("expected result to wrap transitions, got %s", text)
	}
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
	srv := NewServer(client, nil)

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
	srv := NewServer(client, nil)

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
	srv := NewServer(client, nil)

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
			return nil, &jira.JiraError{StatusCode: 404, Message: "resource not found"}
		},
	}
	srv := NewServer(client, nil)

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
	srv := NewServer(client, nil)

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

func TestHandleListBoards(t *testing.T) {
	client := &fakeJiraClient{
		listBoardsFn: func(ctx context.Context, opts ...jira.Option) (jira.BoardList, error) {
			return jira.BoardList{
				{ID: 1, Name: "Board 1", Type: "scrum"},
			}, nil
		},
	}
	srv := NewServer(client, nil)

	result, err := srv.handleListBoards(context.Background(), newRequest("jira_list_boards", map[string]any{"project_key": "MC"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "\"boards\"") {
		t.Errorf("expected result to wrap boards, got %s", text)
	}
	if !contains(text, "Board 1") {
		t.Errorf("expected result to contain Board 1, got %s", text)
	}
}

func TestHandleListBoards_NoFilter(t *testing.T) {
	client := &fakeJiraClient{
		listBoardsFn: func(ctx context.Context, opts ...jira.Option) (jira.BoardList, error) {
			return jira.BoardList{
				{ID: 2, Name: "Board 2", Type: "kanban"},
			}, nil
		},
	}
	srv := NewServer(client, nil)

	result, err := srv.handleListBoards(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "Board 2") {
		t.Errorf("expected result to contain Board 2, got %s", text)
	}
}

func TestHandleListSprints(t *testing.T) {
	client := &fakeJiraClient{
		listSprintsFn: func(ctx context.Context, boardID string, opts ...jira.Option) (jira.SprintList, error) {
			if boardID != "42" {
				t.Errorf("unexpected board id %s", boardID)
			}
			return jira.SprintList{{ID: 1, Name: "Sprint 1", State: "active"}}, nil
		},
	}
	srv := NewServer(client, nil)

	result, err := srv.handleListSprints(context.Background(), newRequest("jira_list_sprints", map[string]any{"board_id": "42"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "\"sprints\"") {
		t.Errorf("expected result to wrap sprints, got %s", text)
	}
	if !contains(text, "Sprint 1") {
		t.Errorf("expected result to contain Sprint 1, got %s", text)
	}
}

func TestHandleListSprints_MissingBoardID(t *testing.T) {
	client := &fakeJiraClient{}
	srv := NewServer(client, nil)

	_, err := srv.handleListSprints(context.Background(), newRequest("jira_list_sprints", map[string]any{}))
	if err == nil {
		t.Fatal("expected error for missing board_id")
	}
}

func TestHandleGetSprint(t *testing.T) {
	client := &fakeJiraClient{
		getSprintFn: func(ctx context.Context, sprintID string) (*jira.Sprint, error) {
			if sprintID != "123" {
				t.Errorf("unexpected sprint id %s", sprintID)
			}
			return &jira.Sprint{ID: 123, Name: "Sprint 1", State: "active"}, nil
		},
	}
	srv := NewServer(client, nil)

	result, err := srv.handleGetSprint(context.Background(), newRequest("jira_get_sprint", map[string]any{"sprint_id": "123"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "Sprint 1") {
		t.Errorf("expected result to contain Sprint 1, got %s", text)
	}
}

func TestHandleGetSprint_MissingSprintID(t *testing.T) {
	client := &fakeJiraClient{}
	srv := NewServer(client, nil)

	_, err := srv.handleGetSprint(context.Background(), newRequest("jira_get_sprint", map[string]any{}))
	if err == nil {
		t.Fatal("expected error for missing sprint_id")
	}
}

func TestHandleGetActiveSprint(t *testing.T) {
	client := &fakeJiraClient{
		getActiveSprintFn: func(ctx context.Context, boardID string, opts ...jira.Option) (*jira.Sprint, error) {
			if boardID != "42" {
				t.Errorf("unexpected board id %s", boardID)
			}
			return &jira.Sprint{ID: 1, Name: "Active Sprint", State: "active"}, nil
		},
	}
	srv := NewServer(client, nil)

	result, err := srv.handleGetActiveSprint(context.Background(), newRequest("jira_get_active_sprint", map[string]any{"board_id": "42"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "Active Sprint") {
		t.Errorf("expected result to contain Active Sprint, got %s", text)
	}
}

func TestHandleGetActiveSprint_NoActiveSprint(t *testing.T) {
	client := &fakeJiraClient{
		getActiveSprintFn: func(ctx context.Context, boardID string, opts ...jira.Option) (*jira.Sprint, error) {
			return nil, nil
		},
	}
	srv := NewServer(client, nil)

	result, err := srv.handleGetActiveSprint(context.Background(), newRequest("jira_get_active_sprint", map[string]any{"board_id": "42"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "No active sprint") {
		t.Errorf("expected result to indicate no active sprint, got %s", text)
	}
}

func TestHandleGetActiveSprint_MissingBoardID(t *testing.T) {
	client := &fakeJiraClient{}
	srv := NewServer(client, nil)

	_, err := srv.handleGetActiveSprint(context.Background(), newRequest("jira_get_active_sprint", map[string]any{}))
	if err == nil {
		t.Fatal("expected error for missing board_id")
	}
}

func TestHandleSearchSprintByName(t *testing.T) {
	client := &fakeJiraClient{
		searchSprintByNameFn: func(ctx context.Context, boardID string, name string, opts ...jira.Option) (jira.SprintList, error) {
			if boardID != "42" || name != "sprint" {
				t.Errorf("unexpected request: board_id=%s name=%s", boardID, name)
			}
			return jira.SprintList{
				{ID: 1, Name: "Sprint 1", State: "active"},
				{ID: 2, Name: "sprint 2", State: "future"},
			}, nil
		},
	}
	srv := NewServer(client, nil)

	result, err := srv.handleSearchSprintByName(context.Background(), newRequest("jira_search_sprint_by_name", map[string]any{"board_id": "42", "name": "sprint"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "\"sprints\"") {
		t.Errorf("expected result to wrap sprints, got %s", text)
	}
	if !contains(text, "Sprint 1") || !contains(text, "sprint 2") {
		t.Errorf("expected result to contain both sprints, got %s", text)
	}
}

func TestHandleSearchSprintByName_MissingName(t *testing.T) {
	client := &fakeJiraClient{}
	srv := NewServer(client, nil)

	_, err := srv.handleSearchSprintByName(context.Background(), newRequest("jira_search_sprint_by_name", map[string]any{"board_id": "42"}))
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestHandleSearchSprintByName_MissingBoardID(t *testing.T) {
	client := &fakeJiraClient{}
	srv := NewServer(client, nil)

	_, err := srv.handleSearchSprintByName(context.Background(), newRequest("jira_search_sprint_by_name", map[string]any{"name": "sprint"}))
	if err == nil {
		t.Fatal("expected error for missing board_id")
	}
}

func TestHandleGetIssueHistory(t *testing.T) {
	client := &fakeJiraClient{
		getIssueHistoryFn: func(ctx context.Context, key string) (*jira.Changelog, error) {
			return &jira.Changelog{
				Total: 1,
				Histories: []jira.ChangelogEntry{
					{ID: "1", Items: []jira.ChangelogItem{{Field: "status", FromString: "To Do", ToString: "Done"}}},
				},
			}, nil
		},
	}
	srv := NewServer(client, nil)
	result, err := srv.handleGetIssueHistory(context.Background(), newRequest("jira_get_issue_history", map[string]any{"issue_key": "PROJ-1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "To Do") || !contains(text, "Done") {
		t.Errorf("expected changelog content, got %s", text)
	}
}

func TestHandleGetIssueHistory_MissingKey(t *testing.T) {
	srv := NewServer(&fakeJiraClient{}, nil)
	_, err := srv.handleGetIssueHistory(context.Background(), newRequest("jira_get_issue_history", map[string]any{}))
	if err == nil {
		t.Fatal("expected error for missing issue_key")
	}
}

func TestHandleListProjectVersions(t *testing.T) {
	client := &fakeJiraClient{
		listProjectVersionsFn: func(ctx context.Context, projectKey string) ([]jira.Version, error) {
			return []jira.Version{{ID: "1", Name: "v1.0"}}, nil
		},
	}
	srv := NewServer(client, nil)
	result, err := srv.handleListProjectVersions(context.Background(), newRequest("jira_list_project_versions", map[string]any{"project_key": "PROJ"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "\"versions\"") || !contains(text, "v1.0") {
		t.Errorf("expected versions wrapper and data, got %s", text)
	}
}

func TestHandleGetVersion(t *testing.T) {
	client := &fakeJiraClient{
		getVersionFn: func(ctx context.Context, id string) (*jira.Version, error) {
			return &jira.Version{ID: "1", Name: "v1.0", Released: true}, nil
		},
	}
	srv := NewServer(client, nil)
	result, err := srv.handleGetVersion(context.Background(), newRequest("jira_get_version", map[string]any{"version_id": "1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "v1.0") || !contains(text, "\"released\":true") {
		t.Errorf("expected version data, got %s", text)
	}
}

func TestHandleGetVersion_MissingID(t *testing.T) {
	srv := NewServer(&fakeJiraClient{}, nil)
	_, err := srv.handleGetVersion(context.Background(), newRequest("jira_get_version", map[string]any{}))
	if err == nil {
		t.Fatal("expected error for missing version_id")
	}
}

func TestHandleGetDevelopmentInfo(t *testing.T) {
	client := &fakeJiraClient{
		getDevelopmentInfoFn: func(ctx context.Context, key string) (*jira.DevelopmentInformation, error) {
			return &jira.DevelopmentInformation{}, nil
		},
	}
	srv := NewServer(client, nil)
	result, err := srv.handleGetDevelopmentInfo(context.Background(), newRequest("jira_get_development_info", map[string]any{"issue_key": "PROJ-1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "{") {
		t.Errorf("expected JSON output, got %s", text)
	}
}

func TestHandleListStatuses(t *testing.T) {
	client := &fakeJiraClient{
		listStatusesFn: func(ctx context.Context, projectKey string) ([]jira.ProjectStatus, error) {
			return []jira.ProjectStatus{
				{Self: "https://jira/statuses/PROJ", ID: "1", Name: "Done",
					Statuses: []jira.Status{{ID: "3", Name: "Done", StatusCategory: jira.StatusCategory{Name: "Done"}}}},
			}, nil
		},
	}
	srv := NewServer(client, nil)
	result, err := srv.handleListStatuses(context.Background(), newRequest("jira_list_statuses", map[string]any{"project_key": "PROJ"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !contains(text, "\"statuses\"") || !contains(text, "Done") {
		t.Errorf("expected statuses data, got %s", text)
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
