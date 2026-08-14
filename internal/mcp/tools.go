package mcp

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

func searchTool() mcp.Tool {
	return mcp.NewTool("jira_search",
		mcp.WithDescription("Search Jira issues using JQL."),
		mcp.WithString("jql", mcp.Description("JQL query string"), mcp.Required()),
		mcp.WithNumber("max_results", mcp.Description("Maximum number of issues to return")),
		mcp.WithString("next_page_token", mcp.Description("Pagination token for next page of results")),
		mcp.WithArray("fields", mcp.Description("Optional list of fields to return (e.g. description, priority, created, labels)")),
	)
}

func getIssueTool() mcp.Tool {
	return mcp.NewTool("jira_get_issue",
		mcp.WithDescription("Get a Jira issue by key."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key, e.g. PROJ-1"), mcp.Required()),
		mcp.WithArray("fields", mcp.Description("Optional list of fields to return (e.g. description, priority, created, labels)")),
	)
}

// createIssueTool and updateIssueTool use NewToolWithRawSchema because the
// `description` field is polymorphic (string OR pre-built ADF object) and
// mark3labs/mcp-go v0.44.0 has no `oneOf`/`anyOf` combinator in the typed
// builder API. Hand-rolling the JSON Schema is the only supported path.
const createIssueToolSchema = `{
  "type": "object",
  "properties": {
    "project_key": {"type": "string", "description": "Project key"},
    "issue_type":  {"type": "string", "description": "Issue type name"},
    "summary":     {"type": "string", "description": "Issue summary"},
    "description": {
      "oneOf": [
        {"type": "string", "description": "Plain-text description (automatically converted to ADF)"},
        {
          "type": "object",
          "required": ["type", "version", "content"],
          "properties": {
            "type":    {"type": "string", "enum": ["doc"]},
            "version": {"type": "number", "enum": [1]},
            "content": {"type": "array"}
          },
          "description": "Pre-built ADF document object (rich text: headings, lists, links, etc.)"
        }
      ]
    },
    "fields": {"type": "object", "description": "Optional additional fields as a JSON object. Supported shapes: parent={\"key\":\"PROJ-1\"}, labels=[\"tag\"]. Set assignee via the separate jira_assign_issue tool, or fields={\"assignee\":{\"accountId\":\"...\"}}."},
    "confirm": {"type": "boolean", "description": "Set true to explicitly confirm creation. Required when the MCP client does not support interactive confirmation."}
  },
  "required": ["project_key", "issue_type", "summary"]
}`

const updateIssueToolSchema = `{
  "type": "object",
  "properties": {
    "issue_key":   {"type": "string", "description": "Issue key"},
    "summary":     {"type": "string", "description": "New summary"},
    "issue_type":  {"type": "string", "description": "New issue type name"},
    "description": {
      "oneOf": [
        {"type": "string", "description": "Plain-text description (automatically converted to ADF)"},
        {
          "type": "object",
          "required": ["type", "version", "content"],
          "properties": {
            "type":    {"type": "string", "enum": ["doc"]},
            "version": {"type": "number", "enum": [1]},
            "content": {"type": "array"}
          },
          "description": "Pre-built ADF document object (rich text: headings, lists, links, etc.)"
        }
      ]
    },
    "fields": {"type": "object", "description": "Optional additional fields as a JSON object. Supported shapes: parent={\"key\":\"PROJ-1\"}, labels=[\"tag\"]. Set assignee via the separate jira_assign_issue tool, or fields={\"assignee\":{\"accountId\":\"...\"}}."}
  },
  "required": ["issue_key"]
}`

func createIssueTool() mcp.Tool {
	return mcp.NewToolWithRawSchema(
		"jira_create_issue",
		"Create a new Jira issue. The description string is converted to ADF: paragraph breaks per line, #/##/### headings, - or * bullets, - [ ]/- [x] checkboxes, and **bold**; a pre-built ADF object is embedded verbatim. The optional fields object supports shapes like parent={\"key\":\"PROJ-1\"} and labels=[\"tag\"]; set assignee via the separate jira_assign_issue tool, or fields={\"assignee\":{\"accountId\":\"...\"}}.",
		json.RawMessage(createIssueToolSchema),
	)
}

func updateIssueTool() mcp.Tool {
	return mcp.NewToolWithRawSchema(
		"jira_update_issue",
		"Update an existing Jira issue. The description string is converted to ADF: paragraph breaks per line, #/##/### headings, - or * bullets, - [ ]/- [x] checkboxes, and **bold**; a pre-built ADF object is embedded verbatim. The optional fields object supports shapes like parent={\"key\":\"PROJ-1\"} and labels=[\"tag\"]; set assignee via the separate jira_assign_issue tool, or fields={\"assignee\":{\"accountId\":\"...\"}}.",
		json.RawMessage(updateIssueToolSchema),
	)
}

func transitionIssueTool() mcp.Tool {
	return mcp.NewTool("jira_transition_issue",
		mcp.WithDescription("Transition a Jira issue to a new status."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("transition_id", mcp.Description("Transition id"), mcp.Required()),
	)
}

func assignIssueTool() mcp.Tool {
	return mcp.NewTool("jira_assign_issue",
		mcp.WithDescription("Assign a Jira issue to a user."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("account_id", mcp.Description("Atlassian account id of the assignee"), mcp.Required()),
	)
}

func deleteIssueTool() mcp.Tool {
	return mcp.NewTool("jira_delete_issue",
		mcp.WithDescription("Delete a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Issue key"), mcp.Required()),
		mcp.WithBoolean("confirm", mcp.Description("Set true to explicitly confirm deletion. Required when the MCP client does not support interactive confirmation.")),
	)
}

func listProjectsTool() mcp.Tool {
	return mcp.NewTool("jira_list_projects",
		mcp.WithDescription("List Jira projects accessible to the authenticated user."),
	)
}

func getTransitionsTool() mcp.Tool {
	return mcp.NewTool("jira_get_transitions",
		mcp.WithDescription("List available transitions for a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Issue key"), mcp.Required()),
	)
}

// addCommentTool uses NewToolWithRawSchema because the `body` argument is
// polymorphic (string OR pre-built ADF object), the same rationale as the
// description field on create/update/child tools.
const addCommentToolSchema = `{
  "type": "object",
  "properties": {
    "issue_key": {"type": "string", "description": "Issue key"},
    "body": {
      "oneOf": [
        {"type": "string", "description": "Comment body (automatically converted to ADF)"},
        {
          "type": "object",
          "required": ["type", "version", "content"],
          "properties": {
            "type":    {"type": "string", "enum": ["doc"]},
            "version": {"type": "number", "enum": [1]},
            "content": {"type": "array"}
          },
          "description": "Pre-built ADF document object (rich text: headings, lists, links, etc.)"
        }
      ]
    }
  },
  "required": ["issue_key", "body"]
}`

func addCommentTool() mcp.Tool {
	return mcp.NewToolWithRawSchema(
		"jira_add_comment",
		"Add a comment to a Jira issue. The body string is converted to ADF: paragraph breaks per line, #/##/### headings, - or * bullets, - [ ]/- [x] checkboxes, and **bold**; a pre-built ADF object is embedded verbatim.",
		json.RawMessage(addCommentToolSchema),
	)
}

func addWorklogTool() mcp.Tool {
	return mcp.NewTool("jira_add_worklog",
		mcp.WithDescription("Add a worklog to a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Issue key"), mcp.Required()),
		mcp.WithString("time_spent", mcp.Description("Time spent, e.g. 1h"), mcp.Required()),
		mcp.WithString("comment", mcp.Description("Optional worklog comment")),
		mcp.WithString("started", mcp.Description("Optional start time in ISO-8601")),
	)
}

func listBoardsTool() mcp.Tool {
	return mcp.NewTool("jira_list_boards",
		mcp.WithDescription("List Jira Agile boards, optionally filtered by project key."),
		mcp.WithString("project_key", mcp.Description("Optional project key or id to filter boards")),
	)
}

func listSprintsTool() mcp.Tool {
	return mcp.NewTool("jira_list_sprints",
		mcp.WithDescription("List sprints for a Jira Agile board."),
		mcp.WithString("board_id", mcp.Description("Board identifier"), mcp.Required()),
		mcp.WithNumber("max_results", mcp.Description("Maximum number of sprints to return")),
	)
}

func getSprintTool() mcp.Tool {
	return mcp.NewTool("jira_get_sprint",
		mcp.WithDescription("Get a Jira Agile sprint by its ID."),
		mcp.WithString("sprint_id", mcp.Description("Sprint identifier"), mcp.Required()),
	)
}

func getActiveSprintTool() mcp.Tool {
	return mcp.NewTool("jira_get_active_sprint",
		mcp.WithDescription("Get the active sprint for a Jira Agile board."),
		mcp.WithString("board_id", mcp.Description("Board identifier"), mcp.Required()),
		mcp.WithNumber("max_results", mcp.Description("Maximum number of sprints to search")),
	)
}

func getIssueHistoryTool() mcp.Tool {
	return mcp.NewTool("jira_get_issue_history",
		mcp.WithDescription("Get the change history (changelog) of a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Issue key"), mcp.Required()),
	)
}

func listProjectVersionsTool() mcp.Tool {
	return mcp.NewTool("jira_list_project_versions",
		mcp.WithDescription("List versions in a Jira project."),
		mcp.WithString("project_key", mcp.Description("Project key"), mcp.Required()),
	)
}

func getVersionTool() mcp.Tool {
	return mcp.NewTool("jira_get_version",
		mcp.WithDescription("Get a Jira project version."),
		mcp.WithString("version_id", mcp.Description("Version identifier"), mcp.Required()),
	)
}

func getDevelopmentInfoTool() mcp.Tool {
	return mcp.NewTool("jira_get_development_info",
		mcp.WithDescription("Get linked pull requests, branches, and commits for a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Issue key"), mcp.Required()),
	)
}

func listStatusesTool() mcp.Tool {
	return mcp.NewTool("jira_list_statuses",
		mcp.WithDescription("List statuses available for a Jira project."),
		mcp.WithString("project_key", mcp.Description("Project key"), mcp.Required()),
	)
}

func createIssueLinkTool() mcp.Tool {
	return mcp.NewTool("jira_create_issue_link",
		mcp.WithDescription("Create a link between two Jira issues."),
		mcp.WithString("link_type", mcp.Description("Link type name, e.g. Relates, Blocks, Cloners"), mcp.Required()),
		mcp.WithString("inward_issue_key", mcp.Description("Key of the inward issue"), mcp.Required()),
		mcp.WithString("outward_issue_key", mcp.Description("Key of the outward issue"), mcp.Required()),
	)
}

func getRelatedIssuesTool() mcp.Tool {
	return mcp.NewTool("jira_get_related_issues",
		mcp.WithDescription("Get issues linked to a given Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Issue key"), mcp.Required()),
	)
}

// createChildIssueTool also accepts a polymorphic description. Reusing the raw
// schema approach for consistency with the create/update pair above.
const createChildIssueToolSchema = `{
  "type": "object",
  "properties": {
    "parent_key":  {"type": "string", "description": "Parent issue key"},
    "project_key": {"type": "string", "description": "Project key"},
    "issue_type":  {"type": "string", "description": "Issue type name (usually Sub-task)"},
    "summary":     {"type": "string", "description": "Issue summary"},
    "description": {
      "oneOf": [
        {"type": "string", "description": "Plain-text description (automatically converted to ADF)"},
        {
          "type": "object",
          "required": ["type", "version", "content"],
          "properties": {
            "type":    {"type": "string", "enum": ["doc"]},
            "version": {"type": "number", "enum": [1]},
            "content": {"type": "array"}
          },
          "description": "Pre-built ADF document object (rich text: headings, lists, links, etc.)"
        }
      ]
    },
    "fields": {"type": "object", "description": "Optional additional fields as a JSON object. Supported shapes: parent={\"key\":\"PROJ-1\"}, labels=[\"tag\"]. Set assignee via the separate jira_assign_issue tool, or fields={\"assignee\":{\"accountId\":\"...\"}}."},
    "confirm": {"type": "boolean", "description": "Set true to explicitly confirm creation. Required when the MCP client does not support interactive confirmation."}
  },
  "required": ["parent_key", "project_key", "issue_type", "summary"]
}`

func createChildIssueTool() mcp.Tool {
	return mcp.NewToolWithRawSchema(
		"jira_create_child_issue",
		"Create a sub-task or child issue under a parent issue. The description string is converted to ADF: paragraph breaks per line, #/##/### headings, - or * bullets, - [ ]/- [x] checkboxes, and **bold**; a pre-built ADF object is embedded verbatim. The optional fields object supports shapes like parent={\"key\":\"PROJ-1\"} and labels=[\"tag\"]; set assignee via the separate jira_assign_issue tool, or fields={\"assignee\":{\"accountId\":\"...\"}}.",
		json.RawMessage(createChildIssueToolSchema),
	)
}

func searchSprintByNameTool() mcp.Tool {
	return mcp.NewTool("jira_search_sprint_by_name",
		mcp.WithDescription("Search sprints by name on a Jira Agile board (case-insensitive substring)."),
		mcp.WithString("board_id", mcp.Description("Board identifier"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Substring to match against sprint names"), mcp.Required()),
		mcp.WithNumber("max_results", mcp.Description("Maximum number of sprints to search")),
	)
}

func listUsersTool() mcp.Tool {
	return mcp.NewTool("jira_list_users",
		mcp.WithDescription("Search Jira users by query string."),
		mcp.WithString("query", mcp.Description("Search query string (optional)")),
		mcp.WithNumber("max_results", mcp.Description("Maximum number of users to return")),
	)
}
