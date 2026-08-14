package mcp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// elicitor is the subset of the MCP server needed to request user confirmation.
// *mcpserver.MCPServer satisfies it.
type elicitor interface {
	RequestElicitation(ctx context.Context, request mcp.ElicitationRequest) (*mcp.ElicitationResult, error)
}

// Server wraps the mcp-go server and Jira client.
type Server struct {
	client       JiraClient
	mcp          *mcpserver.MCPServer
	enabledTools []string
	confirmWrite bool
	elicitor     elicitor
}

// Option configures a Server.
type Option func(*Server)

// WithConfirmation enables or disables the destructive-write confirmation guardrail.
func WithConfirmation(enabled bool) Option {
	return func(s *Server) {
		s.confirmWrite = enabled
	}
}

// guardrailInstructions tells every client-side agent not to misuse destructive tools.
const guardrailInstructions = `jira-mcp connects to a real Jira Cloud instance. The tools jira_create_issue, jira_create_child_issue, and jira_delete_issue make permanent, user-visible changes to the user's board and require explicit user confirmation before running. Never call these tools to test, explore, or inspect how Jira fields work, and never create throwaway test issues. Only create or delete an issue when the user has explicitly asked for that exact action. When a client cannot prompt interactively, these tools accept a confirm parameter: set confirm=true only after the user has explicitly approved the action.`

// destructiveTools are the tools that require explicit user confirmation.
var destructiveTools = map[string]bool{
	"jira_create_issue":       true,
	"jira_create_child_issue": true,
	"jira_delete_issue":       true,
}

// isDestructiveTool reports whether name is a destructive write tool.
func isDestructiveTool(name string) bool {
	return destructiveTools[name]
}

// NewServer builds an MCP server with all Jira tools registered.
// If enabledTools is non-empty, only tools whose names appear in the list
// are exposed. Unset or empty enables all tools.
func NewServer(client JiraClient, enabledTools []string, opts ...Option) *Server {
	s := &Server{
		client: client,
		mcp: mcpserver.NewMCPServer(
			"jira-mcp",
			"0.1.0",
			mcpserver.WithLogging(),
			mcpserver.WithElicitation(),
			mcpserver.WithInstructions(guardrailInstructions),
		),
		enabledTools: enabledTools,
		confirmWrite: true,
	}
	s.elicitor = s.mcp
	for _, opt := range opts {
		opt(s)
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
		s.mcp.AddTool(st.tool, s.confirmationMiddleware(st.handler))
	}
}

// confirmationMiddleware guards destructive tools behind a server-initiated
// user confirmation via MCP elicitation. Direct handler calls (tests) bypass
// this wrapper.
func (s *Server) confirmationMiddleware(next mcpserver.ToolHandlerFunc) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !s.confirmWrite || !isDestructiveTool(request.Params.Name) {
			return next(ctx, request)
		}

		message := fmt.Sprintf("Confirm %s", request.Params.Name)
		args := request.GetArguments()
		for _, key := range []string{"project_key", "issue_key", "parent_key"} {
			if v, ok := args[key]; ok {
				message += fmt.Sprintf(" %s=%v", key, v)
				break
			}
		}

		req := mcp.ElicitationRequest{
			Params: mcp.ElicitationParams{
				Message: message,
				RequestedSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"confirm": map[string]any{
							"type":        "boolean",
							"description": "Confirm this action",
						},
					},
					"required": []string{"confirm"},
				},
			},
		}

		result, err := s.elicitor.RequestElicitation(ctx, req)
		if err != nil {
			// The client does not support interactive elicitation. Fall back to
			// the explicit confirm argument so well-behaved clients can still
			// create/delete once the user has approved the action.
			if confirm, _ := args["confirm"].(bool); confirm {
				return next(ctx, request)
			}
			return resultError("confirmation required: this MCP client does not support interactive confirmation. Pass confirm=true to proceed, or set JIRA_MCP_CONFIRM=off to disable the guardrail."), nil
		}

		switch result.Action {
		case mcp.ElicitationResponseActionAccept:
			return next(ctx, request)
		default:
			return resultText("Cancelled: user did not confirm the operation."), nil
		}
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
		{tool: listUsersTool(), handler: s.handleListUsers},
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
