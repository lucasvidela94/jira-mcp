package mcp

import (
	"net/http"
	"testing"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

func TestServer_ServeSSE_Exists(t *testing.T) {
	// Compile-time assertion that the Server type exposes ServeSSE.
	var srv *Server
	_ = srv.ServeSSE
}

func TestSSEServer_Routes(t *testing.T) {
	srv := NewServer(&fakeJiraClient{}, nil)
	ts := mcpserver.NewTestServer(srv.mcp)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/sse")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestSSEServer_HealthReturns404(t *testing.T) {
	srv := NewServer(&fakeJiraClient{}, nil)
	ts := mcpserver.NewTestServer(srv.mcp)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestServer_ToolList_Filtered(t *testing.T) {
	srv := NewServer(&fakeJiraClient{}, []string{"jira_search", "jira_get_issue"})
	tools := srv.mcp.ListTools()

	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if _, ok := tools["jira_search"]; !ok {
		t.Error("expected jira_search to be registered")
	}
	if _, ok := tools["jira_create_issue"]; ok {
		t.Error("did not expect jira_create_issue to be registered")
	}
}
