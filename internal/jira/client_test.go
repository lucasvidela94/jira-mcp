package jira

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lucasvidela/jira-mcp/internal/config"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	cfg := config.Config{
		URL:      server.URL,
		Username: "user@example.com",
		APIToken: "secret-token",
	}
	return New(cfg, server.Client()), server
}

func TestNew_SetsAuthHeader(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			t.Errorf("expected Basic auth, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"issues": []Issue{}})
	})
	defer server.Close()

	_, err := client.Search(context.Background(), "project = PROJ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_do_MapsError(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		body       string
		retryAfter string
		want       string
	}{
		{
			name:   "400",
			status: 400,
			body:   `{"errorMessages":["Invalid JQL"]}`,
			want:   "jira error 400: validation failed: Invalid JQL",
		},
		{
			name:   "401",
			status: 401,
			body:   `{}`,
			want:   "jira error 401: authentication failed",
		},
		{
			name:   "403",
			status: 403,
			body:   `{}`,
			want:   "jira error 403: permission denied",
		},
		{
			name:   "404",
			status: 404,
			body:   `{}`,
			want:   "jira error 404: issue not found",
		},
		{
			name:       "429",
			status:     429,
			body:       `{}`,
			retryAfter: "60",
			want:       "jira error 429: rate limited; retry after 60",
		},
		{
			name:   "500",
			status: 500,
			body:   `{}`,
			want:   "jira error 500: jira server error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if tc.retryAfter != "" {
					w.Header().Set("Retry-After", tc.retryAfter)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			defer server.Close()

			_, err := client.Search(context.Background(), "project = PROJ")
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tc.want {
				t.Errorf("got %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestClient_do_NetworkError(t *testing.T) {
	cfg := config.Config{
		URL:      "http://invalid.localhost.test",
		Username: "user",
		APIToken: "token",
	}
	client := New(cfg, &http.Client{Timeout: 0})

	_, err := client.Search(context.Background(), "project = PROJ")
	if err == nil {
		t.Fatal("expected error for network failure")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "request failed") {
		t.Errorf("unexpected error type: %v", err)
	}
}

func TestClient_Search(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/search" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("jql") != "project = PROJ" {
			t.Errorf("unexpected jql %q", q.Get("jql"))
		}
		if q.Get("startAt") != "0" {
			t.Errorf("unexpected startAt %q", q.Get("startAt"))
		}
		if q.Get("maxResults") != "50" {
			t.Errorf("unexpected maxResults %q", q.Get("maxResults"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SearchResult{
			Total:  1,
			Issues: []Issue{{Key: "PROJ-1", ID: "10001"}},
		})
	})
	defer server.Close()

	res, err := client.Search(context.Background(), "project = PROJ", WithStartAt(0), WithMaxResults(50))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Total != 1 || len(res.Issues) != 1 || res.Issues[0].Key != "PROJ-1" {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestClient_GetIssue(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Issue{Key: "PROJ-1", ID: "10001", Fields: json.RawMessage(`{"summary":"Test"}`)})
	})
	defer server.Close()

	issue, err := client.GetIssue(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue.Key != "PROJ-1" {
		t.Errorf("unexpected issue key %s", issue.Key)
	}
}

func TestClient_ListProjects(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/project" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Project{{Key: "PROJ", Name: "Project"}})
	})
	defer server.Close()

	projects, err := client.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 1 || projects[0].Key != "PROJ" {
		t.Errorf("unexpected projects: %+v", projects)
	}
}

func TestClient_GetTransitions(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1/transitions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TransitionsResponse{Transitions: []Transition{{ID: "21", Name: "In Progress"}}})
	})
	defer server.Close()

	transitions, err := client.GetTransitions(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(transitions) != 1 || transitions[0].ID != "21" {
		t.Errorf("unexpected transitions: %+v", transitions)
	}
}

func TestClient_CreateIssue(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %s", r.Method)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		fields := body["fields"].(map[string]any)
		if fields["project"].(map[string]any)["key"] != "PROJ" {
			t.Errorf("unexpected project: %v", fields["project"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Issue{Key: "PROJ-2", ID: "10002"})
	})
	defer server.Close()

	issue, err := client.CreateIssue(context.Background(), &CreateIssueRequest{
		ProjectKey:  "PROJ",
		IssueType:   "Task",
		Summary:     "New issue",
		Description: "Description",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue.Key != "PROJ-2" {
		t.Errorf("unexpected key %s", issue.Key)
	}
}

func TestClient_UpdateIssue(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	err := client.UpdateIssue(context.Background(), "PROJ-1", &UpdateIssueRequest{Summary: "Updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_TransitionIssue(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1/transitions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	err := client.TransitionIssue(context.Background(), "PROJ-1", &TransitionIssueRequest{TransitionID: "21"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_AssignIssue(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1/assignee" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	err := client.AssignIssue(context.Background(), "PROJ-1", &AssignIssueRequest{AccountID: "abc123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_DeleteIssue(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	err := client.DeleteIssue(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_AddComment(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1/comment" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(CommentResponse{ID: "10010", Self: "https://jira/comment/10010", Body: "A comment"})
	})
	defer server.Close()

	comment, err := client.AddComment(context.Background(), "PROJ-1", &AddCommentRequest{Body: "A comment"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comment.ID != "10010" {
		t.Errorf("unexpected comment id %s", comment.ID)
	}
}

func TestClient_AddWorklog(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1/worklog" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(WorklogResponse{ID: "10020", Self: "https://jira/worklog/10020", TimeSpent: "1h"})
	})
	defer server.Close()

	worklog, err := client.AddWorklog(context.Background(), "PROJ-1", &AddWorklogRequest{TimeSpent: "1h"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if worklog.ID != "10020" {
		t.Errorf("unexpected worklog id %s", worklog.ID)
	}
}

func TestClient_RetryOnce(t *testing.T) {
	attempts := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Issue{Key: "PROJ-1", ID: "10001"})
	})
	defer server.Close()

	issue, err := client.GetIssue(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue.Key != "PROJ-1" {
		t.Errorf("unexpected key %s", issue.Key)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestClient_RetryExhausted(t *testing.T) {
	attempts := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	defer server.Close()

	_, err := client.GetIssue(context.Background(), "PROJ-1")
	if err == nil {
		t.Fatal("expected error after retry exhausted")
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestClient_RetrySkipsNonRetryable(t *testing.T) {
	attempts := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"errorMessages": []string{"bad"}})
	})
	defer server.Close()

	_, err := client.GetIssue(context.Background(), "PROJ-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt for non-retryable status, got %d", attempts)
	}
}

func TestClient_doesNotRetryWrites(t *testing.T) {
	attempts := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	defer server.Close()

	_, err := client.CreateIssue(context.Background(), &CreateIssueRequest{
		ProjectKey: "PROJ",
		IssueType: "Task",
		Summary:   "New issue",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt for write, got %d", attempts)
	}
}

func TestClient_baseURLWithTrailingSlash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Issue{Key: "PROJ-1", ID: "10001"})
	}))
	defer server.Close()

	cfg := config.Config{
		URL:      server.URL + "/",
		Username: "user",
		APIToken: "token",
	}
	client := New(cfg, server.Client())

	_, err := client.GetIssue(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_baseURLWithPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Issue{Key: "PROJ-1", ID: "10001"})
	}))
	defer server.Close()

	cfg := config.Config{
		URL:      server.URL + "/jira",
		Username: "user",
		APIToken: "token",
	}
	client := New(cfg, server.Client())

	_, err := client.GetIssue(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_DoesNotLeakToken(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"errorMessages": []string{"bad"}})
	})
	defer server.Close()

	_, err := client.Search(context.Background(), "project = PROJ")
	if err == nil {
		t.Fatal("expected error")
	}

	errStr := err.Error()
	if strings.Contains(errStr, "secret-token") || strings.Contains(errStr, "user@example.com") {
		t.Errorf("error leaks credentials: %s", errStr)
	}
}

func TestClient_InvalidBaseURL(t *testing.T) {
	cfg := config.Config{
		URL:      "://invalid-url",
		Username: "user",
		APIToken: "token",
	}
	client := New(cfg, http.DefaultClient)

	_, err := client.GetIssue(context.Background(), "PROJ-1")
	if err == nil {
		t.Fatal("expected error for invalid base URL")
	}
}

func TestClient_WithMaxResults(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("maxResults") != "10" {
			t.Errorf("unexpected maxResults %q", q.Get("maxResults"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SearchResult{Total: 0, Issues: []Issue{}})
	})
	defer server.Close()

	_, err := client.Search(context.Background(), "project = PROJ", WithMaxResults(10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_WithStartAt(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("startAt") != "25" {
			t.Errorf("unexpected startAt %q", q.Get("startAt"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SearchResult{Total: 0, Issues: []Issue{}})
	})
	defer server.Close()

	_, err := client.Search(context.Background(), "project = PROJ", WithStartAt(25))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_SearchURLEncodesJQL(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("jql") != "project = PROJ AND status = \"In Progress\"" {
			t.Errorf("unexpected jql %q", q.Get("jql"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SearchResult{Total: 0, Issues: []Issue{}})
	})
	defer server.Close()

	_, err := client.Search(context.Background(), "project = PROJ AND status = \"In Progress\"")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_GetIssue_NotFound(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"errorMessages": []string{"Issue does not exist"}})
	})
	defer server.Close()

	_, err := client.GetIssue(context.Background(), "UNKNOWN-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "jira error 404: issue not found" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_ListProjects_AuthFailed(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{})
	})
	defer server.Close()

	_, err := client.ListProjects(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "jira error 401: authentication failed" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_CreateIssue_MissingRequired(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"errorMessages": []string{"Specify an issue type"}})
	})
	defer server.Close()

	_, err := client.CreateIssue(context.Background(), &CreateIssueRequest{
		ProjectKey: "PROJ",
		Summary:    "Missing type",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "jira error 400: validation failed: Specify an issue type" {
		t.Errorf("unexpected error: %v", err)
	}
}
