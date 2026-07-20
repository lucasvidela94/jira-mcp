// Package cli provides a standalone command-line interface for Jira operations.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lucasvidela94/jira-mcp/internal/jira"
)

// JiraClient is the subset of Jira operations the CLI needs.
type JiraClient interface {
	Search(ctx context.Context, jql string, opts ...jira.Option) (*jira.SearchResult, error)
	GetIssue(ctx context.Context, key string, opts ...jira.Option) (*jira.Issue, error)
	ListProjects(ctx context.Context) ([]jira.Project, error)
	ListBoards(ctx context.Context, opts ...jira.Option) (jira.BoardList, error)
	ListSprints(ctx context.Context, boardID string, opts ...jira.Option) (jira.SprintList, error)
	GetActiveSprint(ctx context.Context, boardID string, opts ...jira.Option) (*jira.Sprint, error)
	ListProjectVersions(ctx context.Context, projectKey string) ([]jira.Version, error)
	GetVersion(ctx context.Context, id string) (*jira.Version, error)
	ListStatuses(ctx context.Context, projectKey string) ([]jira.ProjectStatus, error)
	GetIssueHistory(ctx context.Context, key string) (*jira.Changelog, error)
	GetRelatedIssues(ctx context.Context, key string) ([]jira.LinkedIssue, error)
	GetDevelopmentInfo(ctx context.Context, key string) (*jira.DevelopmentInformation, error)
	GetSprint(ctx context.Context, sprintID string) (*jira.Sprint, error)
	SearchSprintByName(ctx context.Context, boardID string, name string, opts ...jira.Option) (jira.SprintList, error)
	ListUsers(ctx context.Context, query string, opts ...jira.Option) ([]jira.User, error)
}

// Run dispatches CLI subcommands.
// The first positional arg determines the command. Flags are parsed from args.
func Run(args []string, client JiraClient) error {
	if len(args) == 0 {
		return errors.New("usage: jira-mcp <command> [args...]\n\nAvailable commands:\n" + helpText())
	}

	ctx := context.Background()
	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "list-projects":
		return runListProjects(ctx, client, os.Stdout, hasPretty(cmdArgs))
	case "get-issue":
		if len(cmdArgs) < 1 {
			return errors.New("usage: jira-mcp get-issue <issue-key>")
		}
		return runGetIssue(ctx, client, cmdArgs[0], os.Stdout, hasPretty(cmdArgs))
	case "search":
		if len(cmdArgs) < 1 || strings.HasPrefix(cmdArgs[0], "-") {
			return errors.New("usage: jira-mcp search <jql> [--max-results=N]")
		}
		return runSearch(ctx, client, cmdArgs[0], os.Stdout, hasPretty(cmdArgs))
	case "list-boards":
		projectKey := ""
		if len(cmdArgs) > 0 && !strings.HasPrefix(cmdArgs[0], "-") {
			projectKey = cmdArgs[0]
		}
		return runListBoards(ctx, client, projectKey, os.Stdout, hasPretty(cmdArgs))
	case "list-sprints":
		if len(cmdArgs) < 1 {
			return errors.New("usage: jira-mcp list-sprints <board-id> [--max-results=N]")
		}
		return runListSprints(ctx, client, cmdArgs[0], os.Stdout, hasPretty(cmdArgs))
	case "get-active-sprint":
		if len(cmdArgs) < 1 {
			return errors.New("usage: jira-mcp get-active-sprint <board-id>")
		}
		return runGetActiveSprint(ctx, client, cmdArgs[0], os.Stdout, hasPretty(cmdArgs))
	case "list-project-versions":
		if len(cmdArgs) < 1 {
			return errors.New("usage: jira-mcp list-project-versions <project-key>")
		}
		return runListProjectVersions(ctx, client, cmdArgs[0], os.Stdout, hasPretty(cmdArgs))
	case "get-version":
		if len(cmdArgs) < 1 {
			return errors.New("usage: jira-mcp get-version <version-id>")
		}
		return runGetVersion(ctx, client, cmdArgs[0], os.Stdout, hasPretty(cmdArgs))
	case "list-statuses":
		if len(cmdArgs) < 1 {
			return errors.New("usage: jira-mcp list-statuses <project-key>")
		}
		return runListStatuses(ctx, client, cmdArgs[0], os.Stdout, hasPretty(cmdArgs))
	case "get-issue-history":
		if len(cmdArgs) < 1 {
			return errors.New("usage: jira-mcp get-issue-history <issue-key>")
		}
		return runGetIssueHistory(ctx, client, cmdArgs[0], os.Stdout, hasPretty(cmdArgs))
	case "get-related-issues":
		if len(cmdArgs) < 1 {
			return errors.New("usage: jira-mcp get-related-issues <issue-key>")
		}
		return runGetRelatedIssues(ctx, client, cmdArgs[0], os.Stdout, hasPretty(cmdArgs))
	case "get-development-info":
		if len(cmdArgs) < 1 {
			return errors.New("usage: jira-mcp get-development-info <issue-key>")
		}
		return runGetDevelopmentInfo(ctx, client, cmdArgs[0], os.Stdout, hasPretty(cmdArgs))
	case "list-users":
		return runListUsers(ctx, client, os.Stdout, hasPretty(cmdArgs))
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, helpText())
		return nil
	default:
		return errors.New("unknown command: " + cmd)
	}
}

func helpText() string {
	return `  list-projects                  List Jira projects
  get-issue <issue-key>          Get issue details
  search <jql>                   Search issues with JQL
  list-boards [project-key]      List Agile boards
  list-sprints <board-id>        List sprints for a board
  get-active-sprint <board-id>   Get active sprint for a board
  list-project-versions <key>    List versions in a project
  get-version <id>               Get a project version
  list-statuses <project-key>    List project statuses
  get-issue-history <key>        Get issue changelog
  get-related-issues <key>       Get linked issues
  get-development-info <key>     Get linked PRs/branches/commits
  list-users                     List/search Jira users
`
}

func hasPretty(args []string) bool {
	for _, a := range args {
		if a == "--pretty" {
			return true
		}
	}
	return false
}

func writeJSON(out io.Writer, data any, pretty bool) error {
	var buf []byte
	var err error
	if pretty {
		buf, err = json.MarshalIndent(data, "", "  ")
	} else {
		buf, err = json.Marshal(data)
	}
	if err != nil {
		return err
	}
	_, err = out.Write(buf)
	return err
}

func runListProjects(ctx context.Context, client JiraClient, out io.Writer, pretty bool) error {
	projects, err := client.ListProjects(ctx)
	if err != nil {
		return err
	}
	return writeJSON(out, projects, pretty)
}

func runGetIssue(ctx context.Context, client JiraClient, key string, out io.Writer, pretty bool) error {
	issue, err := client.GetIssue(ctx, key)
	if err != nil {
		return err
	}
	return writeJSON(out, issue, pretty)
}

func runSearch(ctx context.Context, client JiraClient, jql string, out io.Writer, pretty bool) error {
	result, err := client.Search(ctx, jql)
	if err != nil {
		return err
	}
	return writeJSON(out, result, pretty)
}

func runListBoards(ctx context.Context, client JiraClient, projectKey string, out io.Writer, pretty bool) error {
	var opts []jira.Option
	if projectKey != "" {
		opts = append(opts, jira.WithProjectKey(projectKey))
	}
	boards, err := client.ListBoards(ctx, opts...)
	if err != nil {
		return err
	}
	return writeJSON(out, boards, pretty)
}

func runListSprints(ctx context.Context, client JiraClient, boardID string, out io.Writer, pretty bool) error {
	sprints, err := client.ListSprints(ctx, boardID)
	if err != nil {
		return err
	}
	return writeJSON(out, sprints, pretty)
}

func runGetActiveSprint(ctx context.Context, client JiraClient, boardID string, out io.Writer, pretty bool) error {
	sprint, err := client.GetActiveSprint(ctx, boardID)
	if err != nil {
		return err
	}
	if sprint == nil {
		_, err := fmt.Fprintln(out, "No active sprint found")
		return err
	}
	return writeJSON(out, sprint, pretty)
}

func runListProjectVersions(ctx context.Context, client JiraClient, projectKey string, out io.Writer, pretty bool) error {
	versions, err := client.ListProjectVersions(ctx, projectKey)
	if err != nil {
		return err
	}
	return writeJSON(out, versions, pretty)
}

func runGetVersion(ctx context.Context, client JiraClient, id string, out io.Writer, pretty bool) error {
	version, err := client.GetVersion(ctx, id)
	if err != nil {
		return err
	}
	return writeJSON(out, version, pretty)
}

func runListStatuses(ctx context.Context, client JiraClient, projectKey string, out io.Writer, pretty bool) error {
	statuses, err := client.ListStatuses(ctx, projectKey)
	if err != nil {
		return err
	}
	return writeJSON(out, statuses, pretty)
}

func runGetIssueHistory(ctx context.Context, client JiraClient, key string, out io.Writer, pretty bool) error {
	history, err := client.GetIssueHistory(ctx, key)
	if err != nil {
		return err
	}
	return writeJSON(out, history, pretty)
}

func runGetRelatedIssues(ctx context.Context, client JiraClient, key string, out io.Writer, pretty bool) error {
	issues, err := client.GetRelatedIssues(ctx, key)
	if err != nil {
		return err
	}
	return writeJSON(out, issues, pretty)
}

func runGetDevelopmentInfo(ctx context.Context, client JiraClient, key string, out io.Writer, pretty bool) error {
	info, err := client.GetDevelopmentInfo(ctx, key)
	if err != nil {
		return err
	}
	return writeJSON(out, info, pretty)
}

func runListUsers(ctx context.Context, client JiraClient, out io.Writer, pretty bool) error {
	users, err := client.ListUsers(ctx, "")
	if err != nil {
		return err
	}
	return writeJSON(out, users, pretty)
}
