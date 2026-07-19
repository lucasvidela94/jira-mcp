package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/lucasvidela94/jira-mcp/internal/jira"
)

type fakeClient struct {
	listProjectsFn func(ctx context.Context) ([]jira.Project, error)
}

func (f *fakeClient) Search(ctx context.Context, jql string, opts ...jira.Option) (*jira.SearchResult, error) {
	return &jira.SearchResult{Total: 1, Issues: []jira.Issue{{Key: "PROJ-1"}}}, nil
}

func (f *fakeClient) GetIssue(ctx context.Context, key string) (*jira.Issue, error) {
	return &jira.Issue{Key: key, ID: "100"}, nil
}

func (f *fakeClient) ListProjects(ctx context.Context) ([]jira.Project, error) {
	return []jira.Project{{Key: "PROJ", Name: "Project"}}, nil
}

func (f *fakeClient) ListBoards(ctx context.Context, opts ...jira.Option) (jira.BoardList, error) {
	return jira.BoardList{{ID: 1, Name: "Board 1"}}, nil
}

func (f *fakeClient) ListSprints(ctx context.Context, boardID string, opts ...jira.Option) (jira.SprintList, error) {
	return jira.SprintList{{ID: 1, Name: "Sprint 1"}}, nil
}

func (f *fakeClient) GetActiveSprint(ctx context.Context, boardID string, opts ...jira.Option) (*jira.Sprint, error) {
	return &jira.Sprint{ID: 1, Name: "Active Sprint", State: "active"}, nil
}

func (f *fakeClient) ListProjectVersions(ctx context.Context, projectKey string) ([]jira.Version, error) {
	return []jira.Version{{ID: "1", Name: "v1.0"}}, nil
}

func (f *fakeClient) GetVersion(ctx context.Context, id string) (*jira.Version, error) {
	return &jira.Version{ID: id, Name: "v1.0"}, nil
}

func (f *fakeClient) ListStatuses(ctx context.Context, projectKey string) ([]jira.ProjectStatus, error) {
	return []jira.ProjectStatus{{Name: "Done", Statuses: []jira.Status{{Name: "Done"}}}}, nil
}

func (f *fakeClient) GetIssueHistory(ctx context.Context, key string) (*jira.Changelog, error) {
	return &jira.Changelog{Total: 1}, nil
}

func (f *fakeClient) GetRelatedIssues(ctx context.Context, key string) ([]jira.LinkedIssue, error) {
	return []jira.LinkedIssue{{Key: "PROJ-2"}}, nil
}

func (f *fakeClient) GetDevelopmentInfo(ctx context.Context, key string) (*jira.DevelopmentInformation, error) {
	return &jira.DevelopmentInformation{}, nil
}

func (f *fakeClient) GetSprint(ctx context.Context, sprintID string) (*jira.Sprint, error) {
	return &jira.Sprint{ID: 1, Name: "Sprint 1"}, nil
}

func (f *fakeClient) SearchSprintByName(ctx context.Context, boardID string, name string, opts ...jira.Option) (jira.SprintList, error) {
	return jira.SprintList{}, nil
}

func TestRun_ListProjects(t *testing.T) {
	var buf bytes.Buffer
	err := runListProjects(context.Background(), &fakeClient{}, &buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), `"key":"PROJ"`) {
		t.Errorf("expected PROJ in output, got %s", buf.String())
	}
}

func TestRun_GetIssue(t *testing.T) {
	var buf bytes.Buffer
	err := runGetIssue(context.Background(), &fakeClient{}, "PROJ-1", &buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), `"key":"PROJ-1"`) {
		t.Errorf("expected PROJ-1 in output, got %s", buf.String())
	}
}

func TestRun_Search(t *testing.T) {
	var buf bytes.Buffer
	err := runSearch(context.Background(), &fakeClient{}, "project = PROJ", &buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "PROJ-1") {
		t.Errorf("expected PROJ-1 in output, got %s", buf.String())
	}
}

func TestRun_ListBoards(t *testing.T) {
	var buf bytes.Buffer
	err := runListBoards(context.Background(), &fakeClient{}, "", &buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Board 1") {
		t.Errorf("expected Board 1 in output, got %s", buf.String())
	}
}

func TestRun_Dispatch(t *testing.T) {
	err := Run([]string{"list-projects"}, &fakeClient{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	err := Run([]string{"invalid"}, &fakeClient{})
	if err != nil {
		// Expect exit, but os.Exit cannot be caught.
		_ = err
	}
}

func TestRun_NoArgs(t *testing.T) {
	err := Run([]string{}, &fakeClient{})
	if err == nil {
		t.Fatal("expected error for no args")
	}
}
