package mcp

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Server wraps the mcp-go server and Jira client.
type Server struct {
	client       JiraClient
	mcp          *mcpserver.MCPServer
	enabledTools []string
}

// NewServer builds an MCP server with all Jira tools registered.
// If enabledTools is non-empty, only tools whose names appear in the list
// are exposed. Unset or empty enables all tools.
func NewServer(client JiraClient, enabledTools []string) *Server {
	s := &Server{
		client: client,
		mcp: mcpserver.NewMCPServer(
			"jira-mcp",
			"0.1.0",
			mcpserver.WithLogging(),
		),
		enabledTools: enabledTools,
	}
	s.registerTools()
	return s
}

// ServeStdio starts the stdio JSON-RPC server.
func (s *Server) ServeStdio() error {
	return mcpserver.ServeStdio(s.mcp)
}

// ServeSSE starts the HTTP/SSE MCP server on the given address.
// It blocks until the server is interrupted or returns an error.
func (s *Server) ServeSSE(addr string) error {
	sseServer := mcpserver.NewSSEServer(s.mcp)

	errCh := make(chan error, 1)
	go func() {
		if err := sseServer.Start(addr); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-sigCh:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return sseServer.Shutdown(ctx)
	}
}

// registerTools adds all Jira tools to the MCP server.
func (s *Server) registerTools() {
	for _, st := range s.toolDefinitions() {
		s.mcp.AddTool(st.tool, st.handler)
	}
}

// toolDefinitions returns the tool definitions paired with their handlers.
// If enabledTools is set, only tools whose names appear in the list are returned.
func (s *Server) toolDefinitions() []serverTool {
	all := []serverTool{
		{tool: searchTool(), handler: s.handleSearch},
		{tool: getIssueTool(), handler: s.handleGetIssue},
		{tool: createIssueTool(), handler: s.handleCreateIssue},
		{tool: updateIssueTool(), handler: s.handleUpdateIssue},
		{tool: transitionIssueTool(), handler: s.handleTransitionIssue},
		{tool: assignIssueTool(), handler: s.handleAssignIssue},
		{tool: deleteIssueTool(), handler: s.handleDeleteIssue},
		{tool: listProjectsTool(), handler: s.handleListProjects},
		{tool: getTransitionsTool(), handler: s.handleGetTransitions},
		{tool: addCommentTool(), handler: s.handleAddComment},
		{tool: addWorklogTool(), handler: s.handleAddWorklog},
		{tool: listBoardsTool(), handler: s.handleListBoards},
		{tool: listSprintsTool(), handler: s.handleListSprints},
		{tool: getSprintTool(), handler: s.handleGetSprint},
		{tool: getActiveSprintTool(), handler: s.handleGetActiveSprint},
		{tool: searchSprintByNameTool(), handler: s.handleSearchSprintByName},
		{tool: getIssueHistoryTool(), handler: s.handleGetIssueHistory},
		{tool: listProjectVersionsTool(), handler: s.handleListProjectVersions},
		{tool: getVersionTool(), handler: s.handleGetVersion},
		{tool: getDevelopmentInfoTool(), handler: s.handleGetDevelopmentInfo},
		{tool: listStatusesTool(), handler: s.handleListStatuses},
		{tool: createIssueLinkTool(), handler: s.handleCreateIssueLink},
		{tool: getRelatedIssuesTool(), handler: s.handleGetRelatedIssues},
		{tool: createChildIssueTool(), handler: s.handleCreateChildIssue},
	}

	if len(s.enabledTools) == 0 {
		return all
	}

	allowed := make(map[string]bool, len(s.enabledTools))
	for _, name := range s.enabledTools {
		allowed[name] = true
	}

	var filtered []serverTool
	for _, st := range all {
		if allowed[st.tool.Name] {
			filtered = append(filtered, st)
		}
	}
	return filtered
}

type serverTool struct {
	tool    mcp.Tool
	handler mcpserver.ToolHandlerFunc
}
