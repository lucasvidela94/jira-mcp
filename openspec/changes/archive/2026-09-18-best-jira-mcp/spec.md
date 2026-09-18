# Delta Spec: best-jira-mcp

## Overview

This change adds 8 new capabilities to `jira-mcp`: issue linking, issue history, project versions, development information, child issues, project statuses, read-only mode via `ENABLED_TOOLS`, and a standalone CLI. All new behavior follows the existing package seams (`internal/config`, `internal/jira`, `internal/mcp`, `main.go`).

---

## Feature 1: Issue Linking

### Requirement: Create issue link

The system MUST expose `jira_create_issue_link`, which creates a directional link between two Jira issues.

| Parameter | Required | Type | Notes |
|-----------|----------|------|-------|
| `link_type` | yes | string | Name of the link type (e.g. `Relates`, `Blocks`, `Cloners`) |
| `inward_issue_key` | yes | string | Key of the inward issue |
| `outward_issue_key` | yes | string | Key of the outward issue |

The handler MUST call the client method `CreateIssueLink(ctx, *IssueLinkRequest)` with the three arguments. The client MUST POST to `/rest/api/3/issueLink` a JSON body shaped like:

```json
{
  "type": {"name": "Relates"},
  "inwardIssue": {"key": "PROJ-1"},
  "outwardIssue": {"key": "PROJ-2"}
}
```

On success the handler MUST return a plain-text result confirming the link. On any client error the handler MUST return a sanitized tool error via `handleClientError`.

#### Scenario: Link created successfully

- GIVEN valid `link_type="Relates"`, `inward_issue_key="PROJ-1"`, `outward_issue_key="PROJ-2"`
- WHEN `jira_create_issue_link` is called
- THEN the client POSTs `/rest/api/3/issueLink` with the JSON body above
- AND the handler returns a text result containing both issue keys

#### Scenario: Missing link_type

- GIVEN a request without `link_type`
- WHEN `jira_create_issue_link` is called
- THEN the handler returns an error stating `missing required parameter: link_type`
- AND the client is not invoked

#### Scenario: Jira rejects the link

- GIVEN a request with two valid keys
- AND Jira responds with HTTP 400 and `{"errorMessages":["Issue link type not found"]}`
- WHEN `jira_create_issue_link` is called
- THEN the handler returns a tool error containing the Jira message
- AND the error MUST NOT leak any credential

### Requirement: Get related issues

The system MUST expose `jira_get_related_issues`, which returns every issue linked to a given issue.

| Parameter | Required | Type | Notes |
|-----------|----------|------|-------|
| `issue_key` | yes | string | Jira issue key |

The handler MUST call `GetRelatedIssues(ctx, key)`. The client MUST perform `GET /rest/api/3/issue/{key}?fields=issuelinks` and parse `fields.issuelinks` into `[]LinkedIssue`. The handler MUST return `{"related_issues": <issues>}`.

#### Scenario: Issue has linked issues

- GIVEN an issue `PROJ-1` with one outward link to `PROJ-2`
- WHEN `jira_get_related_issues` is called
- THEN the client requests `GET /rest/api/3/issue/PROJ-1?fields=issuelinks`
- AND the handler returns a JSON object containing `PROJ-2` under `related_issues`

#### Scenario: Issue has no links

- GIVEN an issue `PROJ-1` whose `issuelinks` field is empty
- WHEN `jira_get_related_issues` is called
- THEN the handler returns `{"related_issues": []}`

#### Scenario: Issue not found

- GIVEN a request for `UNKNOWN-1`
- AND Jira responds with HTTP 404
- WHEN `jira_get_related_issues` is called
- THEN the handler returns a tool error containing "resource not found"

---

## Feature 2: Issue History

### Requirement: Get issue history

The system MUST expose `jira_get_issue_history`, which returns the changelog of an issue.

| Parameter | Required | Type | Notes |
|-----------|----------|------|-------|
| `issue_key` | yes | string | Jira issue key |

The handler MUST call `GetIssueHistory(ctx, key)`. The client MUST perform `GET /rest/api/3/issue/{key}/changelog` and decode the response into a `Changelog` model. The handler MUST return the changelog as JSON.

#### Scenario: Issue has history entries

- GIVEN an issue `PROJ-1` with a changelog entry for a status change
- WHEN `jira_get_issue_history` is called
- THEN the client requests `GET /rest/api/3/issue/PROJ-1/changelog`
- AND the handler returns a JSON object containing `histories` with the status change item

#### Scenario: Missing issue_key

- GIVEN a request without `issue_key`
- WHEN `jira_get_issue_history` is called
- THEN the handler returns an error stating `missing required parameter: issue_key`
- AND the client is not invoked

#### Scenario: Jira returns a permission error

- GIVEN an issue `PROJ-1`
- AND Jira responds with HTTP 403
- WHEN `jira_get_issue_history` is called
- THEN the handler returns a tool error containing "permission denied"

---

## Feature 3: Project Versions

### Requirement: List project versions

The system MUST expose `jira_list_project_versions`, which returns the versions of a project.

| Parameter | Required | Type | Notes |
|-----------|----------|------|-------|
| `project_key` | yes | string | Project key passed verbatim |

The handler MUST call `ListProjectVersions(ctx, projectKey)`. The client MUST perform `GET /rest/api/3/project/{projectKey}/versions` and decode the response into `[]Version`. The handler MUST return `{"versions": <versions>}`.

#### Scenario: Project has versions

- GIVEN a project `PROJ` with versions `v1.0.0` and `v2.0.0`
- WHEN `jira_list_project_versions` is called
- THEN the client requests `GET /rest/api/3/project/PROJ/versions`
- AND the handler returns a JSON object containing both versions under `versions`

#### Scenario: Missing project_key

- GIVEN a request without `project_key`
- WHEN `jira_list_project_versions` is called
- THEN the handler returns an error stating `missing required parameter: project_key`
- AND the client is not invoked

### Requirement: Get version

The system MUST expose `jira_get_version`, which returns a single version by its ID.

| Parameter | Required | Type | Notes |
|-----------|----------|------|-------|
| `version_id` | yes | string | Version identifier passed verbatim |

The handler MUST call `GetVersion(ctx, id)`. The client MUST perform `GET /rest/api/3/version/{id}` and decode the response into a `Version`. The handler MUST return the version as JSON.

#### Scenario: Existing version

- GIVEN a version id `10000` for version `v1.0.0`
- WHEN `jira_get_version` is called
- THEN the client requests `GET /rest/api/3/version/10000`
- AND the handler returns a JSON object containing the version name

#### Scenario: Version not found

- GIVEN a version id `99999`
- AND Jira responds with HTTP 404
- WHEN `jira_get_version` is called
- THEN the handler returns a tool error containing "resource not found"

---

## Feature 4: Development Information

### Requirement: Get development information

The system MUST expose `jira_get_development_info`, which returns linked pull requests, branches, and commits for an issue.

| Parameter | Required | Type | Notes |
|-----------|----------|------|-------|
| `issue_key` | yes | string | Jira issue key |

The handler MUST call `GetDevelopmentInfo(ctx, key)`. The client MUST perform `GET /rest/api/3/issue/{key}?fields=development` and parse `fields.development` into a `DevelopmentInformation` model. If the field is absent the client MUST return an empty `DevelopmentInformation{}`. The handler MUST return the model as JSON.

#### Scenario: Issue has linked PRs and branches

- GIVEN an issue `PROJ-1` with a repository `acme/repo` linked to one PR and one branch
- WHEN `jira_get_development_info` is called
- THEN the client requests `GET /rest/api/3/issue/PROJ-1?fields=development`
- AND the handler returns a JSON object containing the PR and branch under `detail.repositories`

#### Scenario: No development information

- GIVEN an issue `PROJ-1` whose `development` field is absent
- WHEN `jira_get_development_info` is called
- THEN the handler returns `{"detail": null}` (or equivalent empty payload)

#### Scenario: Invalid issue key

- GIVEN a request for `UNKNOWN-1`
- AND Jira responds with HTTP 404
- WHEN `jira_get_development_info` is called
- THEN the handler returns a tool error containing "resource not found"

---

## Feature 5: Child Issues

### Requirement: Create child issue

The system MUST expose `jira_create_child_issue`, which creates a sub-task or other child issue under a parent issue.

| Parameter | Required | Type | Notes |
|-----------|----------|------|-------|
| `parent_key` | yes | string | Parent issue key |
| `project_key` | yes | string | Project key for the new issue |
| `issue_type` | yes | string | Issue type name (usually `Sub-task`) |
| `summary` | yes | string | Summary of the new issue |
| `description` | no | string | Optional plain-text description |
| `fields` | no | object | Optional additional fields as a JSON object |

The handler MUST call `CreateChildIssue(ctx, parentKey, *CreateChildIssueRequest)`. The client MUST POST to `/rest/api/3/issue` a JSON body shaped like:

```json
{
  "fields": {
    "project": {"key": "PROJ"},
    "issuetype": {"name": "Sub-task"},
    "summary": "Sub-task summary",
    "parent": {"key": "PROJ-1"},
    "description": "Optional description"
  }
}
```

The `description` key MUST be omitted when the parameter is empty. The `fields` object MUST be merged into the top-level `fields` map, allowing extra custom fields to be sent. On success the handler MUST return the created issue as JSON.

#### Scenario: Child issue created successfully

- GIVEN valid `parent_key="PROJ-1"`, `project_key="PROJ"`, `issue_type="Sub-task"`, `summary="Sub-task summary"`
- WHEN `jira_create_child_issue` is called
- THEN the client POSTs `/rest/api/3/issue` with `fields.parent.key = PROJ-1`
- AND the handler returns the created issue JSON

#### Scenario: Missing parent_key

- GIVEN a request without `parent_key`
- WHEN `jira_create_child_issue` is called
- THEN the handler returns an error stating `missing required parameter: parent_key`
- AND the client is not invoked

#### Scenario: Jira validation fails

- GIVEN valid required parameters
- AND Jira responds with HTTP 400 and `{"errorMessages":["Specify an issue type"]}`
- WHEN `jira_create_child_issue` is called
- THEN the handler returns a tool error containing the Jira message

---

## Feature 6: List Statuses

### Requirement: List project statuses

The system MUST expose `jira_list_statuses`, which returns the statuses available for a project, grouped by issue type.

| Parameter | Required | Type | Notes |
|-----------|----------|------|-------|
| `project_key` | yes | string | Project key passed verbatim |

The handler MUST call `ListStatuses(ctx, projectKey)`. The client MUST perform `GET /rest/api/3/project/{projectKey}/statuses` and decode the response into `[]ProjectStatus`. The handler MUST return `{"statuses": <statuses>}`. The response structure MUST preserve the grouping by issue type (i.e. `subjects` array with `statuses` inside).

#### Scenario: Project has grouped statuses

- GIVEN a project `PROJ` with a `Task` issue type containing statuses `To Do`, `In Progress`, `Done`
- WHEN `jira_list_statuses` is called
- THEN the client requests `GET /rest/api/3/project/PROJ/statuses`
- AND the handler returns a JSON object containing those statuses under `statuses`

#### Scenario: Missing project_key

- GIVEN a request without `project_key`
- WHEN `jira_list_statuses` is called
- THEN the handler returns an error stating `missing required parameter: project_key`
- AND the client is not invoked

---

## Feature 7: Read-only Mode (`ENABLED_TOOLS`)

### Requirement: Parse `ENABLED_TOOLS` from environment

The system MUST read the optional environment variable `ENABLED_TOOLS` during `config.Load()`. If the variable is absent or empty, `Config.EnabledTools` MUST be empty. If present, the value MUST be split on commas, each entry trimmed of leading and trailing whitespace, and empty entries MUST be discarded. The resulting list MUST be stored in `Config.EnabledTools`.

| Env var | Format | Example |
|---------|--------|---------|
| `ENABLED_TOOLS` | comma-separated exact tool names | `jira_search,jira_get_issue,jira_list_projects` |

#### Scenario: Env var unset

- GIVEN `ENABLED_TOOLS` is not set
- WHEN `config.Load()` runs
- THEN `cfg.EnabledTools` is empty
- AND all tools are exposed

#### Scenario: Subset of tools enabled

- GIVEN `ENABLED_TOOLS="jira_search,jira_get_issue"`
- WHEN `config.Load()` runs
- THEN `cfg.EnabledTools` equals `["jira_search", "jira_get_issue"]`
- AND only those two tools are registered

#### Scenario: Spaces and empty entries ignored

- GIVEN `ENABLED_TOOLS=" jira_search , , jira_get_issue "`
- WHEN `config.Load()` runs
- THEN `cfg.EnabledTools` equals `["jira_search", "jira_get_issue"]`
- AND the empty entry is discarded

### Requirement: Filter tools at registration time

`mcp.NewServer(client, enabledTools)` MUST accept the enabled-tools list. When the list is non-empty, the server MUST register only tools whose exact name appears in the list. Unknown names MUST be ignored silently; an unknown name MUST NOT cause startup failure. When the list is empty, the server MUST register all existing and new tools.

#### Scenario: Unknown tool names in list

- GIVEN `ENABLED_TOOLS="jira_search,jira_fantasy_tool"`
- WHEN the server starts
- THEN `jira_search` is registered
- AND `jira_fantasy_tool` is ignored
- AND the server starts without error

---

## Feature 8: Standalone CLI

### Requirement: CLI mode detection

`main.go` MUST detect CLI mode when either:

1. The first argument after the program name is a recognized subcommand (e.g. `jira-mcp list-projects`).
2. The `--cli` flag is present among the arguments (e.g. `jira-mcp --cli list-projects`).

In CLI mode the program MUST load config, build the Jira client, and invoke `cli.Run(args, client)`. The existing `--transport`, `--port`, and `--version` flags, and the default stdio MCP server behavior, MUST remain unchanged when no CLI subcommand is present.

| Global flag | Behavior |
|-------------|----------|
| `--pretty` | Pretty-print JSON output |
| `--help` | Show help and exit |

#### Scenario: Subcommand triggers CLI mode

- GIVEN the invocation `jira-mcp list-projects`
- WHEN `main` runs
- THEN it loads config and calls `cli.Run(["list-projects"], client)`
- AND the MCP stdio server is not started

#### Scenario: --cli flag triggers CLI mode

- GIVEN the invocation `jira-mcp --cli list-projects`
- WHEN `main` runs
- THEN the `--cli` flag is removed from the argument slice
- AND `cli.Run(["list-projects"], client)` is called

#### Scenario: No subcommand keeps MCP server mode

- GIVEN the invocation `jira-mcp --transport stdio`
- WHEN `main` runs
- THEN the MCP server starts over stdio
- AND `cli.Run` is not called

### Requirement: Supported CLI commands

The CLI package MUST implement the following read-only commands, each calling the corresponding `JiraClient` method and writing JSON to stdout:

| Command | Arguments | Client method |
|---------|-----------|---------------|
| `list-projects` | none | `ListProjects` |
| `get-issue` | `<issue-key>` | `GetIssue` |
| `search` | `<jql>` | `Search` |
| `list-boards` | `[<project-key>]` | `ListBoards` |
| `list-sprints` | `<board-id>` | `ListSprints` |
| `get-active-sprint` | `<board-id>` | `GetActiveSprint` |
| `list-project-versions` | `<project-key>` | `ListProjectVersions` |
| `get-version` | `<version-id>` | `GetVersion` |
| `list-statuses` | `<project-key>` | `ListStatuses` |
| `get-issue-history` | `<issue-key>` | `GetIssueHistory` |
| `get-related-issues` | `<issue-key>` | `GetRelatedIssues` |
| `get-development-info` | `<issue-key>` | `GetDevelopmentInfo` |

The CLI interface MUST mirror the client interface so that the real `*jira.Client` satisfies it. The `search` command SHOULD accept `--max-results=N` as an optional flag. The `list-boards` command SHOULD treat a positional argument as a project key filter. The `list-sprints` and `get-active-sprint` commands SHOULD accept `--max-results=N` as an optional flag.

#### Scenario: list-projects outputs JSON

- GIVEN `ENABLED_TOOLS` and config are valid
- AND the client returns `[{"key":"PROJ","name":"Project"}]`
- WHEN `jira-mcp list-projects` runs
- THEN stdout contains a JSON array with the project

#### Scenario: search with max-results

- GIVEN the invocation `jira-mcp search "project = PROJ" --max-results=5`
- WHEN the CLI parses arguments
- THEN it calls `Search(ctx, "project = PROJ", WithMaxResults(5))`
- AND stdout contains the JSON search result

#### Scenario: Unknown command

- GIVEN the invocation `jira-mcp frobnicate`
- WHEN the CLI parses arguments
- THEN it prints an error message to stderr
- AND exits with a non-zero status code

#### Scenario: Missing command argument

- GIVEN the invocation `jira-mcp get-issue`
- WHEN the CLI parses arguments
- THEN it prints an error message indicating the missing `<issue-key>`
- AND exits with a non-zero status code

#### Scenario: Config error in CLI mode

- GIVEN `JIRA_API_TOKEN` is missing
- WHEN `jira-mcp list-projects` runs
- THEN it prints `config error: missing required environment variable: JIRA_API_TOKEN` to stderr
- AND exits with status code 1

---

## Interface Contracts

### `Config` extension

```go
type Config struct {
    URL          string
    Username     string
    APIToken     string
    EnabledTools []string
}
```

`Load()` MUST parse `ENABLED_TOOLS` using the comma-split-and-trim rules above.

### `JiraClient` extension

```go
type JiraClient interface {
    // ... existing methods ...

    CreateIssueLink(ctx context.Context, req *jira.IssueLinkRequest) error
    GetRelatedIssues(ctx context.Context, key string) ([]jira.LinkedIssue, error)
    GetIssueHistory(ctx context.Context, key string) (*jira.Changelog, error)
    ListProjectVersions(ctx context.Context, projectKey string) ([]jira.Version, error)
    GetVersion(ctx context.Context, id string) (*jira.Version, error)
    GetDevelopmentInfo(ctx context.Context, key string) (*jira.DevelopmentInformation, error)
    CreateChildIssue(ctx context.Context, parentKey string, req *jira.CreateChildIssueRequest) (*jira.Issue, error)
    ListStatuses(ctx context.Context, projectKey string) ([]jira.ProjectStatus, error)
}
```

### `Server` construction

```go
func NewServer(client JiraClient, enabledTools []string) *Server
```

`main.go` MUST call `mcp.NewServer(client, cfg.EnabledTools)`.

---

## README and Documentation

### Requirement: Update README tool table

The README MUST list all new tools and the `ENABLED_TOOLS` environment variable. The new tools are:

- `jira_create_issue_link`
- `jira_get_related_issues`
- `jira_get_issue_history`
- `jira_list_project_versions`
- `jira_get_version`
- `jira_get_development_info`
- `jira_create_child_issue`
- `jira_list_statuses`

### Requirement: Document `ENABLED_TOOLS`

The README MUST include a table row for `ENABLED_TOOLS` explaining that it is optional, comma-separated, and restricts the exposed tool list.

### Requirement: Document standalone CLI

The README MUST mention the CLI mode with a short example, e.g. `jira-mcp list-projects` and `jira-mcp get-issue PROJ-1`.

---

## Non-Functional Requirements

### Requirement: Preserve existing behavior

The default MCP server mode, all existing tools, and the existing transport flags MUST continue to work exactly as before when `ENABLED_TOOLS` is unset and no CLI subcommand is used.

### Requirement: Release artifacts unchanged

GoReleaser, Docker, Homebrew, and install scripts MUST NOT be modified. The CLI is delivered by the same binary without changing release artifact names.

---

## Acceptance Criteria

- [ ] All 8 feature gaps are exposed as MCP tools or CLI commands.
- [ ] `go test ./...` passes with no coverage drop.
- [ ] Every new tool has handler tests using the `fakeJiraClient` pattern.
- [ ] Every new client method has `httptest` table-driven coverage.
- [ ] `ENABLED_TOOLS` is parsed, tested, and applied at tool registration.
- [ ] The standalone CLI has dispatch and flag-parsing tests.
- [ ] README documents the new tools, `ENABLED_TOOLS`, and CLI usage.
- [ ] Release artifacts (Homebrew, Docker, install scripts) remain unchanged.
