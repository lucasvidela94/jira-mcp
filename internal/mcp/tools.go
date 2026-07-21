package mcp

import "github.com/mark3labs/mcp-go/mcp"

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

func createIssueTool() mcp.Tool {
	return mcp.NewTool("jira_create_issue",
		mcp.WithDescription("Create a new Jira issue."),
		mcp.WithString("project_key", mcp.Description("Project key"), mcp.Required()),
		mcp.WithString("issue_type", mcp.Description("Issue type name"), mcp.Required()),
		mcp.WithString("summary", mcp.Description("Issue summary"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Optional plain-text description (automatically converted to ADF format)")),
		mcp.WithObject("fields", mcp.Description("Optional additional fields as a JSON object")),
	)
}

func updateIssueTool() mcp.Tool {
	return mcp.NewTool("jira_update_issue",
		mcp.WithDescription("Update an existing Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("summary", mcp.Description("New summary")),
		mcp.WithString("issue_type", mcp.Description("New issue type name")),
		mcp.WithString("description", mcp.Description("Optional plain-text description (automatically converted to ADF format)")),
		mcp.WithObject("fields", mcp.Description("Optional additional fields as a JSON object")),
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
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
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
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
	)
}

func addCommentTool() mcp.Tool {
	return mcp.NewTool("jira_add_comment",
		mcp.WithDescription("Add a comment to a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Comment body"), mcp.Required()),
	)
}

func addWorklogTool() mcp.Tool {
	return mcp.NewTool("jira_add_worklog",
		mcp.WithDescription("Add a worklog to a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("time_spent", mcp.Description("Time spent, e.g. 1h30m"), mcp.Required()),
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
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
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
		mcp.WithDescription("Get a Jira version by its ID."),
		mcp.WithString("version_id", mcp.Description("Version identifier"), mcp.Required()),
	)
}

func getDevelopmentInfoTool() mcp.Tool {
	return mcp.NewTool("jira_get_development_info",
		mcp.WithDescription("Get linked pull requests, branches, and commits for a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
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
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
	)
}

func createChildIssueTool() mcp.Tool {
	return mcp.NewTool("jira_create_child_issue",
		mcp.WithDescription("Create a sub-task or child issue under a parent issue."),
		mcp.WithString("parent_key", mcp.Description("Parent issue key"), mcp.Required()),
		mcp.WithString("project_key", mcp.Description("Project key"), mcp.Required()),
		mcp.WithString("issue_type", mcp.Description("Issue type name (usually Sub-task)"), mcp.Required()),
		mcp.WithString("summary", mcp.Description("Issue summary"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Optional plain-text description (automatically converted to ADF format)")),
		mcp.WithObject("fields", mcp.Description("Optional additional fields as a JSON object")),
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
