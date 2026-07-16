package mcp

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mark3labs/mcp-go/mcp"
)

// Server wraps the mcp-go server and Jira client.
type Server struct {
	client JiraClient
	mcp    *mcpserver.MCPServer
}

// NewServer builds an MCP server with all Jira tools registered.
func NewServer(client JiraClient) *Server {
	s := &Server{
		client: client,
		mcp: mcpserver.NewMCPServer(
			"jira-mcp",
			"0.1.0",
			mcpserver.WithLogging(),
		),
	}
	s.registerTools()
	return s
}

// ServeStdio starts the stdio JSON-RPC server.
func (s *Server) ServeStdio() error {
	return mcpserver.ServeStdio(s.mcp)
}

// registerTools adds the 11 Jira tools to the MCP server.
func (s *Server) registerTools() {
	for _, st := range s.toolDefinitions() {
		s.mcp.AddTool(st.tool, st.handler)
	}
}

// toolDefinitions returns the tool definitions paired with their handlers.
func (s *Server) toolDefinitions() []serverTool {
	return []serverTool{
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
	}
}

type serverTool struct {
	tool    mcp.Tool
	handler mcpserver.ToolHandlerFunc
}
