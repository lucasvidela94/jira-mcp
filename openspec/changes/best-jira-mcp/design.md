# Design: best-jira-mcp

## Technical Approach

Add the 8 missing Jira capabilities behind the existing `JiraClient` interface seam. Each new capability follows the established repo pattern: a thin client method in `internal/jira/client.go`, a model in `internal/jira/models.go`, a tool definition in `internal/mcp/tools.go`, a handler in `internal/mcp/handlers_*.go`, and table-driven tests using `httptest` for the client and `fakeJiraClient` for the handler. The `ENABLED_TOOLS` filter is applied at tool registration time in `internal/mcp/server.go`. The standalone CLI lives in a new `internal/cli` package and is dispatched from `main.go` without breaking the default stdio MCP server mode.

The work is delivered in the 3 chained PR slices from the proposal:
1. Read-only tools: issue history, versions, dev-info, statuses.
2. Relationship/child tools: issue linking, create child issue.
3. Operational modes: `ENABLED_TOOLS` + CLI standalone.

## Architecture Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|--------------|-----------|
| Issue links retrieval | `GET /rest/api/3/issue/{key}?fields=issuelinks` | Dedicated endpoint (none exists) | Jira Cloud has no standalone “related issues” endpoint; the data lives inside `fields.issuelinks`. |
| Development info source | `GET /rest/api/3/issue/{key}?fields=development` | `GET /rest/dev-status/1.0/issue/detail` | Single request, no need to resolve key→ID; field payload is permissive enough to capture PR/branch/commit summaries. |
| Child issue creation | Reuse `CreateIssueRequest` + `ParentKey` | Separate `CreateChildIssueRequest` | Avoids duplicating the marshaler; parent is just another optional field. |
| Read-only tool filtering | Allowlist at registration time | Denylist or handler-level gating | Allowlist is explicit and safe; a missing entry means a tool is disabled rather than accidentally enabled. |
| CLI dispatch | Subcommands with auto-detect in `main.go` | Separate binary under `cmd/` | Keeps release artifacts unchanged; single binary can still be MCP server by default. |
| CLI output | JSON by default, `--pretty` flag | Human-formatted tables | JSON is composable with `jq` and matches the MCP tool result shape. |

## Data Flow

```text
MCP request        ┌─────────────┐     Jira Cloud
  ────────────────> │ mcp handler │ ─────────────────>
                    │   (args)    │ <────────────────
                    └──────┬──────┘     JSON
                           │
                    ┌──────▼──────┐
                    │ JiraClient  │
                    │ interface   │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │ internal/   │
                    │ jira.Client │ <── httptest (client tests)
                    └─────────────┘

CLI invocation
  ────────────────> main.go ────> internal/cli ────> JiraClient ────> Jira Cloud
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/jira/client.go` | Modify | Add 8 new client methods for links, history, versions, dev-info, child issues, statuses. |
| `internal/jira/models.go` | Modify | Add `IssueLink`, `IssueLinkType`, `LinkedIssue`, `Changelog`, `ChangelogEntry`, `ChangelogItem`, `Version`, `DevelopmentInformation`, `PullRequest`, `Branch`, `Commit`, `Status`, `StatusCategory`, `CreateChildIssueRequest`, and helper types. |
| `internal/jira/client_test.go` | Modify | Add `httptest` coverage for every new client method. |
| `internal/mcp/tools.go` | Modify | Add 8 new tool constructors. |
| `internal/mcp/handlers_link.go` | Create | Handlers for `jira_create_issue_link` and `jira_get_related_issues`. |
| `internal/mcp/handlers_history.go` | Create | Handler for `jira_get_issue_history`. |
| `internal/mcp/handlers_version.go` | Create | Handlers for `jira_list_project_versions` and `jira_get_version`. |
| `internal/mcp/handlers_dev.go` | Create | Handler for `jira_get_development_info`. |
| `internal/mcp/handlers_child.go` | Create | Handler for `jira_create_child_issue`. |
| `internal/mcp/handlers_status.go` | Create | Handler for `jira_list_statuses`. |
| `internal/mcp/server.go` | Modify | Accept `EnabledTools` and filter the tool list before registration. |
| `internal/mcp/interfaces.go` | Modify | Add new methods to `JiraClient`. |
| `internal/mcp/handlers_test.go` | Modify | Extend `fakeJiraClient` with new method stubs and add handler tests. |
| `internal/config/config.go` | Modify | Parse `ENABLED_TOOLS` into `[]string`. |
| `internal/config/config_test.go` | Modify | Add tests for `ENABLED_TOOLS` parsing. |
| `internal/cli/cli.go` | Create | Standalone CLI root and subcommand dispatch. |
| `internal/cli/commands.go` | Create | Individual CLI command implementations. |
| `internal/cli/cli_test.go` | Create | Unit tests for CLI dispatch and flag parsing. |
| `main.go` | Modify | Detect CLI invocation and dispatch to `internal/cli`. |
| `main_test.go` | Modify | Add tests for CLI dispatch logic. |
| `README.md` | Modify | Add new tools and `ENABLED_TOOLS` env var to tables. |

## Interfaces / Contracts

### Config

```go
type Config struct {
    URL          string
    Username     string
    APIToken     string
    EnabledTools []string // from ENABLED_TOOLS
}

func Load() (Config, error) {
    // ... existing required vars ...
    enabledTools := os.Getenv("ENABLED_TOOLS")
    if enabledTools != "" {
        cfg.EnabledTools = splitAndTrim(enabledTools, ",")
    }
    return cfg, nil
}
```

### JiraClient

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

### Server construction

```go
func NewServer(client JiraClient, enabledTools []string) *Server {
    s := &Server{client: client, mcp: mcpserver.NewMCPServer(...), enabledTools: enabledTools}
    s.registerTools()
    return s
}
```

`main.go` will call `mcp.NewServer(client, cfg.EnabledTools)`.

## Feature Designs

### 1. Issue Linking

#### Endpoints

- `POST /rest/api/3/issueLink` — create a link.
- `GET /rest/api/3/issue/{key}?fields=issuelinks` — read related issues.

#### Models (`internal/jira/models.go`)

```go
type IssueLinkRequest struct {
    LinkTypeName string `json:"-"`
    InwardKey    string `json:"-"`
    OutwardKey   string `json:"-"`
}

func (r IssueLinkRequest) MarshalJSON() ([]byte, error) {
    return json.Marshal(map[string]any{
        "type":         map[string]any{"name": r.LinkTypeName},
        "inwardIssue":  map[string]any{"key": r.InwardKey},
        "outwardIssue": map[string]any{"key": r.OutwardKey},
    })
}

type IssueLink struct {
    ID           string       `json:"id,omitempty"`
    Self         string       `json:"self,omitempty"`
    Type         IssueLinkType `json:"type"`
    InwardIssue  *LinkedIssue `json:"inwardIssue,omitempty"`
    OutwardIssue *LinkedIssue `json:"outwardIssue,omitempty"`
}

type IssueLinkType struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Inward      string `json:"inward"`
    Outward     string `json:"outward"`
    Self        string `json:"self,omitempty"`
}

type LinkedIssue struct {
    ID     string `json:"id"`
    Key    string `json:"key"`
    Self   string `json:"self"`
    Fields json.RawMessage `json:"fields,omitempty"`
}
```

#### Client (`internal/jira/client.go`)

```go
func (c *Client) CreateIssueLink(ctx context.Context, req *IssueLinkRequest) error {
    resp, err := c.do(ctx, http.MethodPost, "/rest/api/3/issueLink", nil, req)
    if err != nil {
        return err
    }
    return handleResponse(resp, nil)
}

func (c *Client) GetRelatedIssues(ctx context.Context, key string) ([]LinkedIssue, error) {
    query := url.Values{"fields": []string{"issuelinks"}}
    resp, err := c.doWithRetry(ctx, http.MethodGet, fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(key)), query, nil)
    if err != nil {
        return nil, err
    }
    var issue Issue
    if err := handleResponse(resp, &issue); err != nil {
        return nil, err
    }
    return parseIssueLinks(issue.Fields)
}
```

`parseIssueLinks` extracts `fields.issuelinks`, collects every `inwardIssue` and `outwardIssue`, and returns them as `[]LinkedIssue`.

#### Tools (`internal/mcp/tools.go`)

```go
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
```

#### Handlers (`internal/mcp/handlers_link.go`)

```go
func (s *Server) handleCreateIssueLink(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    args := request.GetArguments()
    linkType, err := requiredString(args, "link_type")
    if err != nil {
        return nil, err
    }
    inwardKey, err := requiredString(args, "inward_issue_key")
    if err != nil {
        return nil, err
    }
    outwardKey, err := requiredString(args, "outward_issue_key")
    if err != nil {
        return nil, err
    }

    req := &jira.IssueLinkRequest{
        LinkTypeName: linkType,
        InwardKey:    inwardKey,
        OutwardKey:   outwardKey,
    }
    if err := s.client.CreateIssueLink(ctx, req); err != nil {
        return handleClientError(err), nil
    }
    return resultText(fmt.Sprintf("Link created between %s and %s", inwardKey, outwardKey)), nil
}

func (s *Server) handleGetRelatedIssues(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    key, err := requiredString(request.GetArguments(), "issue_key")
    if err != nil {
        return nil, err
    }
    issues, err := s.client.GetRelatedIssues(ctx, key)
    if err != nil {
        return handleClientError(err), nil
    }
    return resultJSON(map[string]any{"related_issues": issues})
}
```

#### Test strategy

- Client: verify `POST /rest/api/3/issueLink` body and `GET /rest/api/3/issue/{key}?fields=issuelinks` parsing.
- Handler: verify required args, correct client method call, and JSON wrapping.

#### Example JSON payloads

Create link request:
```json
{
  "type": {"name": "Relates"},
  "inwardIssue": {"key": "PROJ-1"},
  "outwardIssue": {"key": "PROJ-2"}
}
```

Get issue response (issuelinks only):
```json
{
  "id": "10001",
  "key": "PROJ-1",
  "fields": {
    "issuelinks": [
      {
        "id": "12345",
        "type": {"name": "Relates", "inward": "relates to", "outward": "relates to"},
        "outwardIssue": {"id": "10002", "key": "PROJ-2", "self": "https://..."}
      }
    ]
  }
}
```

---

### 2. Issue History

#### Endpoint

- `GET /rest/api/3/issue/{key}/changelog`

#### Models

```go
type Changelog struct {
    StartAt    int              `json:"startAt,omitempty"`
    MaxResults int              `json:"maxResults,omitempty"`
    Total      int              `json:"total,omitempty"`
    Histories  []ChangelogEntry `json:"histories"`
}

type ChangelogEntry struct {
    ID      string          `json:"id"`
    Author  *User           `json:"author,omitempty"`
    Created string          `json:"created"`
    Items   []ChangelogItem `json:"items"`
}

type ChangelogItem struct {
    Field      string `json:"field"`
    FieldType  string `json:"fieldtype"`
    From       string `json:"from,omitempty"`
    FromString string `json:"fromString,omitempty"`
    To         string `json:"to,omitempty"`
    ToString   string `json:"toString,omitempty"`
}

type User struct {
    AccountID   string `json:"accountId"`
    DisplayName string `json:"displayName"`
    EmailAddress string `json:"emailAddress,omitempty"`
}
```

#### Client

```go
func (c *Client) GetIssueHistory(ctx context.Context, key string) (*Changelog, error) {
    resp, err := c.doWithRetry(ctx, http.MethodGet, fmt.Sprintf("/rest/api/3/issue/%s/changelog", url.PathEscape(key)), nil, nil)
    if err != nil {
        return nil, err
    }
    var changelog Changelog
    if err := handleResponse(resp, &changelog); err != nil {
        return nil, err
    }
    return &changelog, nil
}
```

#### Tool

```go
func getIssueHistoryTool() mcp.Tool {
    return mcp.NewTool("jira_get_issue_history",
        mcp.WithDescription("Get the change history (changelog) of a Jira issue."),
        mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
    )
}
```

#### Handler (`internal/mcp/handlers_history.go`)

```go
func (s *Server) handleGetIssueHistory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    key, err := requiredString(request.GetArguments(), "issue_key")
    if err != nil {
        return nil, err
    }
    history, err := s.client.GetIssueHistory(ctx, key)
    if err != nil {
        return handleClientError(err), nil
    }
    return resultJSON(history)
}
```

#### Test strategy

- Client: assert `GET /rest/api/3/issue/PROJ-1/changelog` path and `Changelog` decoding.
- Handler: assert required param and JSON output.

#### Example JSON payload

```json
{
  "startAt": 0,
  "maxResults": 100,
  "total": 1,
  "histories": [
    {
      "id": "123",
      "created": "2024-01-01T10:00:00.000+0000",
      "author": {"accountId": "abc", "displayName": "Alice"},
      "items": [
        {"field": "status", "fromString": "To Do", "toString": "In Progress"}
      ]
    }
  ]
}
```

---

### 3. Project Versions

#### Endpoints

- `GET /rest/api/3/project/{key}/versions`
- `GET /rest/api/3/version/{id}`

#### Models

```go
type Version struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
    Archived    bool   `json:"archived"`
    Released    bool   `json:"released"`
    ReleaseDate string `json:"releaseDate,omitempty"`
    StartDate   string `json:"startDate,omitempty"`
    Overdue     bool   `json:"overdue,omitempty"`
    ProjectID   int    `json:"projectId,omitempty"`
    Self        string `json:"self"`
}
```

#### Client

```go
func (c *Client) ListProjectVersions(ctx context.Context, projectKey string) ([]Version, error) {
    resp, err := c.doWithRetry(ctx, http.MethodGet, fmt.Sprintf("/rest/api/3/project/%s/versions", url.PathEscape(projectKey)), nil, nil)
    if err != nil {
        return nil, err
    }
    var versions []Version
    if err := handleResponse(resp, &versions); err != nil {
        return nil, err
    }
    return versions, nil
}

func (c *Client) GetVersion(ctx context.Context, id string) (*Version, error) {
    resp, err := c.doWithRetry(ctx, http.MethodGet, fmt.Sprintf("/rest/api/3/version/%s", url.PathEscape(id)), nil, nil)
    if err != nil {
        return nil, err
    }
    var version Version
    if err := handleResponse(resp, &version); err != nil {
        return nil, err
    }
    return &version, nil
}
```

#### Tools

```go
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
```

#### Handlers (`internal/mcp/handlers_version.go`)

```go
func (s *Server) handleListProjectVersions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    key, err := requiredString(request.GetArguments(), "project_key")
    if err != nil {
        return nil, err
    }
    versions, err := s.client.ListProjectVersions(ctx, key)
    if err != nil {
        return handleClientError(err), nil
    }
    return resultJSON(map[string]any{"versions": versions})
}

func (s *Server) handleGetVersion(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    id, err := requiredString(request.GetArguments(), "version_id")
    if err != nil {
        return nil, err
    }
    version, err := s.client.GetVersion(ctx, id)
    if err != nil {
        return handleClientError(err), nil
    }
    return resultJSON(version)
}
```

#### Test strategy

- Client: assert paths and empty/non-empty version lists.
- Handler: assert required params and JSON wrapping.

#### Example JSON payload

```json
[
  {
    "id": "10000",
    "name": "v1.0.0",
    "released": true,
    "releaseDate": "2024-06-01",
    "self": "https://example.atlassian.net/rest/api/3/version/10000"
  }
]
```

---

### 4. Development Information

#### Endpoint

- `GET /rest/api/3/issue/{key}?fields=development`

#### Models

```go
type DevelopmentInformation struct {
    Detail *DevelopmentDetail `json:"detail,omitempty"`
}

type DevelopmentDetail struct {
    Repositories []Repository `json:"repositories,omitempty"`
}

type Repository struct {
    Name         string         `json:"name,omitempty"`
    URL          string         `json:"url,omitempty"`
    PullRequests []PullRequest  `json:"pullRequests,omitempty"`
    Branches     []Branch       `json:"branches,omitempty"`
    Commits      []Commit       `json:"commits,omitempty"`
}

type PullRequest struct {
    ID          string `json:"id,omitempty"`
    Name        string `json:"name,omitempty"`
    URL         string `json:"url,omitempty"`
    Status      string `json:"status,omitempty"`
    Open        bool   `json:"open,omitempty"`
    Closed      bool   `json:"closed,omitempty"`
    Merged      bool   `json:"merged,omitempty"`
}

type Branch struct {
    Name         string `json:"name"`
    URL          string `json:"url,omitempty"`
    CreatePullRequestUrl string `json:"createPullRequestUrl,omitempty"`
}

type Commit struct {
    ID              string `json:"id"`
    Message         string `json:"message,omitempty"`
    URL             string `json:"url,omitempty"`
    AuthorTimestamp string `json:"authorTimestamp,omitempty"`
}
```

#### Client

```go
func (c *Client) GetDevelopmentInfo(ctx context.Context, key string) (*DevelopmentInformation, error) {
    query := url.Values{"fields": []string{"development"}}
    resp, err := c.doWithRetry(ctx, http.MethodGet, fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(key)), query, nil)
    if err != nil {
        return nil, err
    }
    var issue Issue
    if err := handleResponse(resp, &issue); err != nil {
        return nil, err
    }
    return parseDevelopmentInfo(issue.Fields)
}
```

`parseDevelopmentInfo` reads `fields.development` into `DevelopmentInformation`. If the field is absent, return an empty struct.

#### Tool

```go
func getDevelopmentInfoTool() mcp.Tool {
    return mcp.NewTool("jira_get_development_info",
        mcp.WithDescription("Get linked pull requests, branches, and commits for a Jira issue."),
        mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
    )
}
```

#### Handler (`internal/mcp/handlers_dev.go`)

```go
func (s *Server) handleGetDevelopmentInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    key, err := requiredString(request.GetArguments(), "issue_key")
    if err != nil {
        return nil, err
    }
    info, err := s.client.GetDevelopmentInfo(ctx, key)
    if err != nil {
        return handleClientError(err), nil
    }
    return resultJSON(info)
}
```

#### Test strategy

- Client: assert `fields=development` query and parsing of empty/present payloads.
- Handler: assert required param and JSON output.

#### Example JSON payload

```json
{
  "development": {
    "detail": {
      "repositories": [
        {
          "name": "lucasvidela94/jira-mcp",
          "pullRequests": [
            {"id": "42", "name": "feat/awesome", "status": "OPEN", "url": "https://github.com/..."}
          ],
          "branches": [
            {"name": "feat/awesome", "url": "https://github.com/..."}
          ],
          "commits": [
            {"id": "abc123", "message": "initial commit", "url": "https://github.com/..."}
          ]
        }
      ]
    }
  }
}
```

---

### 5. Child Issues / Sub-tasks

#### Endpoint

- `POST /rest/api/3/issue` with `fields.parent.key`

#### Models

```go
type CreateChildIssueRequest struct {
    ParentKey   string
    ProjectKey  string
    IssueType   string
    Summary     string
    Description string
    Fields      map[string]any
}

func (r CreateChildIssueRequest) MarshalJSON() ([]byte, error) {
    fields := map[string]any{
        "project":   map[string]any{"key": r.ProjectKey},
        "issuetype": map[string]any{"name": r.IssueType},
        "summary":   r.Summary,
        "parent":    map[string]any{"key": r.ParentKey},
    }
    if r.Description != "" {
        fields["description"] = r.Description
    }
    for k, v := range r.Fields {
        fields[k] = v
    }
    return json.Marshal(map[string]any{"fields": fields})
}
```

#### Client

```go
func (c *Client) CreateChildIssue(ctx context.Context, parentKey string, req *CreateChildIssueRequest) (*Issue, error) {
    resp, err := c.do(ctx, http.MethodPost, "/rest/api/3/issue", nil, req)
    if err != nil {
        return nil, err
    }
    var issue Issue
    if err := handleResponse(resp, &issue); err != nil {
        return nil, err
    }
    return &issue, nil
}
```

#### Tool

```go
func createChildIssueTool() mcp.Tool {
    return mcp.NewTool("jira_create_child_issue",
        mcp.WithDescription("Create a sub-task or child issue under a parent issue."),
        mcp.WithString("parent_key", mcp.Description("Parent issue key"), mcp.Required()),
        mcp.WithString("project_key", mcp.Description("Project key"), mcp.Required()),
        mcp.WithString("issue_type", mcp.Description("Issue type name (usually Sub-task)"), mcp.Required()),
        mcp.WithString("summary", mcp.Description("Issue summary"), mcp.Required()),
        mcp.WithString("description", mcp.Description("Optional plain-text description")),
        mcp.WithObject("fields", mcp.Description("Optional additional fields as a JSON object")),
    )
}
```

#### Handler (`internal/mcp/handlers_child.go`)

```go
func (s *Server) handleCreateChildIssue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    args := request.GetArguments()
    parentKey, err := requiredString(args, "parent_key")
    if err != nil {
        return nil, err
    }
    projectKey, err := requiredString(args, "project_key")
    if err != nil {
        return nil, err
    }
    issueType, err := requiredString(args, "issue_type")
    if err != nil {
        return nil, err
    }
    summary, err := requiredString(args, "summary")
    if err != nil {
        return nil, err
    }
    fields, err := optionalObject(args, "fields")
    if err != nil {
        return nil, err
    }

    req := &jira.CreateChildIssueRequest{
        ParentKey:   parentKey,
        ProjectKey:  projectKey,
        IssueType:   issueType,
        Summary:     summary,
        Description: optionalString(args, "description"),
        Fields:      fields,
    }
    issue, err := s.client.CreateChildIssue(ctx, parentKey, req)
    if err != nil {
        return handleClientError(err), nil
    }
    return resultJSON(issue)
}
```

#### Test strategy

- Client: assert `POST /rest/api/3/issue` body contains `fields.parent.key`.
- Handler: assert required params and that the request is passed to the client.

#### Example JSON payload

```json
{
  "fields": {
    "project": {"key": "PROJ"},
    "issuetype": {"name": "Sub-task"},
    "summary": "Sub-task summary",
    "parent": {"key": "PROJ-1"}
  }
}
```

---

### 6. List Statuses

#### Endpoint

- `GET /rest/api/3/project/{key}/statuses`

#### Models

```go
type StatusCategory struct {
    Self     string  `json:"self"`
    ID       int     `json:"id"`
    Key      string  `json:"key"`
    ColorName string `json:"colorName"`
    Name     string  `json:"name"`
}

type Status struct {
    ID           string         `json:"id"`
    Name         string         `json:"name"`
    Description  string         `json:"description,omitempty"`
    StatusCategory StatusCategory `json:"statusCategory"`
}
```

The endpoint actually returns a list of issue types with statuses, but the proposal models are `StatusCategory` and `Status`. The client will return a flattened `[]Status` with their categories, or we can return the raw structure. To match the proposal, define an intermediate type:

```go
type ProjectStatus struct {
    Self     string  `json:"self"`
    ID       string  `json:"id"`
    Name     string  `json:"name"`
    Subjects []struct {
        Name    string   `json:"name"`
        Statuses []Status `json:"statuses"`
    } `json:"subjects,omitempty"`
    Statuses []Status `json:"statuses,omitempty"`
}
```

However, the Jira Cloud endpoint `GET /rest/api/3/project/{key}/statuses` returns a list of objects, each with `self`, `id`, `name`, `subjects` (issue types with statuses), or `statuses` directly. The simpler approach is to return the raw list as `[]ProjectStatus` and let the handler return it. But the proposal says models: StatusCategory, Status. So I'll define `Status` and `StatusCategory`, and the client will flatten the response into `[]Status`. Actually, the endpoint returns statuses grouped by issue type. Flattening loses grouping. Let me return the grouped structure.

Final decision: define `ProjectStatus` as the response type, and embed `Status` and `StatusCategory` inside it. The handler returns `{"statuses": projectStatuses}`.

```go
type ProjectStatus struct {
    Self     string `json:"self"`
    ID       string `json:"id"`
    Name     string `json:"name"`
    Subjects []IssueTypeStatuses `json:"subjects,omitempty"`
    Statuses []Status            `json:"statuses,omitempty"`
}

type IssueTypeStatuses struct {
    ID       string   `json:"id"`
    Name     string   `json:"name"`
    Self     string   `json:"self,omitempty"`
    Statuses []Status `json:"statuses"`
}
```

#### Client

```go
func (c *Client) ListStatuses(ctx context.Context, projectKey string) ([]ProjectStatus, error) {
    resp, err := c.doWithRetry(ctx, http.MethodGet, fmt.Sprintf("/rest/api/3/project/%s/statuses", url.PathEscape(projectKey)), nil, nil)
    if err != nil {
        return nil, err
    }
    var statuses []ProjectStatus
    if err := handleResponse(resp, &statuses); err != nil {
        return nil, err
    }
    return statuses, nil
}
```

#### Tool

```go
func listStatusesTool() mcp.Tool {
    return mcp.NewTool("jira_list_statuses",
        mcp.WithDescription("List statuses available for a Jira project."),
        mcp.WithString("project_key", mcp.Description("Project key"), mcp.Required()),
    )
}
```

#### Handler (`internal/mcp/handlers_status.go`)

```go
func (s *Server) handleListStatuses(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    key, err := requiredString(request.GetArguments(), "project_key")
    if err != nil {
        return nil, err
    }
    statuses, err := s.client.ListStatuses(ctx, key)
    if err != nil {
        return handleClientError(err), nil
    }
    return resultJSON(map[string]any{"statuses": statuses})
}
```

#### Test strategy

- Client: assert `GET /rest/api/3/project/PROJ/statuses` path and decoding of grouped statuses.
- Handler: assert required param and JSON wrapping.

#### Example JSON payload

```json
[
  {
    "self": "https://example.atlassian.net/rest/api/3/project/10000/statuses",
    "id": "10000",
    "name": "Task",
    "statuses": [
      {"id": "1", "name": "To Do", "statusCategory": {"id": 2, "key": "new", "colorName": "blue-gray", "name": "To Do"}}
    ]
  }
]
```

---

### 7. Read-only Mode (`ENABLED_TOOLS`)

#### Config

Add to `Config`:

```go
EnabledTools []string
```

`Load` parses `ENABLED_TOOLS` as comma-separated, trimmed values:

```go
func splitAndTrim(s, sep string) []string {
    parts := strings.Split(s, sep)
    var out []string
    for _, p := range parts {
        p = strings.TrimSpace(p)
        if p != "" {
            out = append(out, p)
        }
    }
    return out
}
```

#### Server

`Server` gains `enabledTools []string`. `NewServer` signature changes to `NewServer(client JiraClient, enabledTools []string)`. `toolDefinitions` filters before returning:

```go
func (s *Server) toolDefinitions() []serverTool {
    all := []serverTool{
        // ... all 24 tools ...
    }
    if len(s.enabledTools) == 0 {
        return all
    }
    allowed := make(map[string]struct{}, len(s.enabledTools))
    for _, name := range s.enabledTools {
        allowed[name] = struct{}{}
    }
    var filtered []serverTool
    for _, st := range all {
        if _, ok := allowed[st.tool.Name]; ok {
            filtered = append(filtered, st)
        }
    }
    return filtered
}
```

`registerTools` iterates over the filtered list.

#### Test strategy

- Config: verify comma-separated parsing, spaces, and empty values.
- Server: verify `TestServer_ToolList` checks both full list and filtered list.
- Update `main.go` and `main_test.go` calls to `NewServer` to pass `cfg.EnabledTools`.

#### Example env var

```bash
ENABLED_TOOLS="jira_search,jira_get_issue,jira_list_projects"
```

---

### 8. Standalone CLI

#### Dispatch in `main.go`

Current `main()` flow:

1. If `len(os.Args) > 1 && os.Args[1] == "update"` → run update.
2. Scan `os.Args[1:]` for `--cli`. If found, set `cliMode = true` and remove it from the arg slice.
3. If not `cliMode` and `len(args) > 0 && !strings.HasPrefix(args[0], "-")` → `cliMode = true` (bare subcommand).
4. If `cliMode` → load config, build client, call `cli.Run(args, client)`.
5. Otherwise → parse server flags (`--transport`, `--port`, `--version`) and start MCP server.

```go
func main() {
    if len(os.Args) > 1 && os.Args[1] == "update" {
        if err := runUpdate(); err != nil {
            log.Fatalf("update failed: %s", err)
        }
        return
    }

    cliMode, args := detectCLI(os.Args[1:])
    if cliMode {
        cfg, err := config.Load()
        if err != nil {
            fmt.Fprintf(os.Stderr, "config error: %s\n", err)
            os.Exit(1)
        }
        client := jira.New(cfg, nil)
        if err := cli.Run(args, client); err != nil {
            fmt.Fprintf(os.Stderr, "error: %s\n", err)
            os.Exit(1)
        }
        return
    }

    // existing MCP server flag parsing and dispatch
}

func detectCLI(args []string) (bool, []string) {
    var filtered []string
    for _, a := range args {
        if a == "--cli" {
            continue
        }
        filtered = append(filtered, a)
    }
    if len(filtered) > 0 && !strings.HasPrefix(filtered[0], "-") {
        return true, filtered
    }
    return len(args) != len(filtered), filtered
}
```

#### CLI package (`internal/cli`)

Files:
- `internal/cli/cli.go` — root parser, subcommand dispatch, shared output formatting.
- `internal/cli/commands.go` — command implementations.
- `internal/cli/cli_test.go` — tests.

Subcommands (read-only first):

```text
jira-mcp list-projects
jira-mcp get-issue <issue-key>
jira-mcp search <jql> [--max-results=N]
jira-mcp list-boards [<project-key>]
jira-mcp list-sprints <board-id> [--max-results=N]
jira-mcp get-active-sprint <board-id>
jira-mcp list-project-versions <project-key>
jira-mcp get-version <version-id>
jira-mcp list-statuses <project-key>
jira-mcp get-issue-history <issue-key>
jira-mcp get-related-issues <issue-key>
jira-mcp get-development-info <issue-key>
```

Global flags:
- `--pretty` — pretty-print JSON output.
- `--help` — show help.

#### CLI interface

```go
package cli

import "github.com/lucasvidela94/jira-mcp/internal/jira"

type JiraClient interface {
    Search(ctx context.Context, jql string, opts ...jira.Option) (*jira.SearchResult, error)
    GetIssue(ctx context.Context, key string) (*jira.Issue, error)
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
}

func Run(args []string, client JiraClient) error
```

#### Command implementation pattern

Each command function:

```go
func runListProjects(ctx context.Context, client JiraClient, out io.Writer, pretty bool) error {
    projects, err := client.ListProjects(ctx)
    if err != nil {
        return err
    }
    return writeJSON(out, projects, pretty)
}
```

`writeJSON` encodes to JSON, pretty-printing if `pretty` is true.

#### Test strategy

- `cli_test.go`: test dispatch for each subcommand using a fake client.
- `main_test.go`: test `detectCLI` and `main` dispatch behavior.
- Integration: skip in `-short`; run `go build` only.

---

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Client methods for each new endpoint | `httptest` table-driven tests in `internal/jira/client_test.go`. |
| Unit | Handler argument parsing and error mapping | `fakeJiraClient` in `internal/mcp/handlers_test.go`. |
| Unit | `ENABLED_TOOLS` parsing and server filtering | `config_test.go` and `server_test.go`. |
| Unit | CLI dispatch and flag parsing | `internal/cli/cli_test.go` and `main_test.go`. |
| Integration | Build and CLI invocation | `go build ./...` and `go test ./...`. |
| E2E | Real Jira Cloud | Skipped by default; manual only. |

Strict TDD is active: each new behavior gets one test first, then the minimal implementation. Golden files are not used; JSON output is asserted by string containment or decoded values.

## Migration / Rollout

No data migration. Operational changes:
- `NewServer` signature gains a second parameter. Update all call sites (`main.go`, tests).
- `ENABLED_TOOLS` is optional; default behavior is all tools enabled.
- CLI subcommands are additive; default behavior remains stdio MCP server.
- README and tool table updated in the final PR slice.

## Open Questions

1. Should the standalone CLI eventually support write commands (create issue, transition, etc.)? Out of scope for this design; start read-only.
2. Should `jira_get_development_info` fall back to `dev-status` endpoint if `fields=development` is empty? Defer until real Jira data shows the need.
3. Should `ENABLED_TOOLS` support glob patterns or only exact names? Use exact names for predictability.
