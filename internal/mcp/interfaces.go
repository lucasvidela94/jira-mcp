package mcp

import (
	"context"

	"github.com/lucasvidela/jira-mcp/internal/jira"
)

// JiraClient is the contract the MCP server needs from a Jira client.
// The real *jira.Client satisfies this interface.
type JiraClient interface {
	Search(ctx context.Context, jql string, opts ...jira.Option) (*jira.SearchResult, error)
	GetIssue(ctx context.Context, key string) (*jira.Issue, error)
	CreateIssue(ctx context.Context, req *jira.CreateIssueRequest) (*jira.Issue, error)
	UpdateIssue(ctx context.Context, key string, req *jira.UpdateIssueRequest) error
	TransitionIssue(ctx context.Context, key string, req *jira.TransitionIssueRequest) error
	AssignIssue(ctx context.Context, key string, req *jira.AssignIssueRequest) error
	DeleteIssue(ctx context.Context, key string) error
	ListProjects(ctx context.Context) ([]jira.Project, error)
	GetTransitions(ctx context.Context, key string) ([]jira.Transition, error)
	AddComment(ctx context.Context, key string, req *jira.AddCommentRequest) (*jira.CommentResponse, error)
	AddWorklog(ctx context.Context, key string, req *jira.AddWorklogRequest) (*jira.WorklogResponse, error)
}

// Ensure the concrete client satisfies the interface.
var _ JiraClient = (*jira.Client)(nil)
