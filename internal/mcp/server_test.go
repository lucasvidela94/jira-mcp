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
	srv := NewServer(&fakeJiraClient{})
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
	srv := NewServer(&fakeJiraClient{})
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
