# Design: workflow-friction

## Architecture Decision

We keep the change additive and behind the existing seams. No new packages are introduced. The new `GetCurrentUser` capability follows the exact same vertical slice as the other 24 read tools: model → client → interface → tool → handler → server registration → tests.

The `jira_search` enhancement is kept minimal by reusing the existing `jira.Option` mechanism for `fields` and by applying the `open_only` JQL mutation in the handler, where tool-level semantics live.

## Data Flow

### `jira_get_current_user`

```
MCP client
  └─> mcp.Server.handleGetCurrentUser
        └─> JiraClient.GetCurrentUser
              └─> jira.Client.doWithRetry(GET /rest/api/3/myself)
                    └─> Jira Cloud
              └─> decode CurrentUser
        └─> resultJSON(CurrentUser)
```

### `jira_search` with `open_only` and `fields`

```
MCP client
  └─> mcp.Server.handleSearch
        ├─> read jql (required)
        ├─> read open_only (optional)
        │     └─> if true: jql += " AND resolution = Unresolved"
        ├─> read fields (optional)
        │     └─> if present: opts = append(opts, jira.WithFields(fields))
        ├─> read max_results (optional)
        │     └─> if present: opts = append(opts, jira.WithMaxResults(n))
        └─> s.client.Search(ctx, jql, opts...)
              └─> jira.Client.Search
                    ├─> build SearchRequest
                    ├─> extract fields from url.Values (set by WithFields)
                    ├─> extract maxResults from url.Values
                    └─> POST /rest/api/3/search/jql
        └─> resultJSON(SearchResult)
```

## Model Design

```go
type CurrentUser struct {
    AccountID    string `json:"accountId"`
    DisplayName  string `json:"displayName"`
    EmailAddress string `json:"emailAddress,omitempty"`
    Active       bool   `json:"active"`
}
```

## Option Design

`jira.Option` is `func(*url.Values)`. We introduce `WithFields` as an option that stores the requested field names under the synthetic key `"fields"` in `url.Values`. The `Search` method extracts that key before building the request body and removes it from query values. This keeps the `JiraClient.Search` signature unchanged and avoids adding a second option type.

```go
func WithFields(fields []string) Option {
    return func(v *url.Values) {
        for _, f := range fields {
            v.Add("fields", f)
        }
    }
}
```

Inside `Search`:

```go
query := url.Values{}
for _, opt := range opts { opt(&query) }

req := SearchRequest{JQL: jql, Fields: defaultFields}
if fieldList, ok := query["fields"]; ok && len(fieldList) > 0 {
    req.Fields = fieldList
    delete(query, "fields")
}
if v := query.Get("maxResults"); v != "" { ... }
```

## Handler Helpers

Two new helpers are added to `internal/mcp/helpers.go`:

- `optionalBool(args, name) (bool, bool, error)` — returns (value, present, error)
- `optionalStringSlice(args, name) ([]string, bool, error)` — returns (values, present, error)

These follow the same pattern as `optionalInt` and `optionalString`.

## Tool Definitions

```go
func getCurrentUserTool() mcp.Tool {
    return mcp.NewTool("jira_get_current_user",
        mcp.WithDescription("Get the currently authenticated Jira user."),
    )
}
```

```go
func searchTool() mcp.Tool {
    return mcp.NewTool("jira_search",
        mcp.WithDescription("Search Jira issues using JQL."),
        mcp.WithString("jql", mcp.Description("JQL query string"), mcp.Required()),
        mcp.WithNumber("max_results", mcp.Description("Maximum number of issues to return")),
        mcp.WithBoolean("open_only", mcp.Description("When true, restrict to unresolved issues")),
        mcp.WithArray("fields", mcp.Description("Fields to return, e.g. [\"key\", \"summary\", \"status\"]")),
    )
}
```

## Server Registration

Add `jira_get_current_user` to `toolDefinitions` in `internal/mcp/server.go`. The `TestServer_ToolList` expectation is updated from 24 to 25 tools.

## Testing Strategy

- `internal/jira/client_test.go`: `TestClient_GetCurrentUser` verifies `GET /rest/api/3/myself` and decoding. `TestClient_Search_WithFields` verifies the custom fields reach the request body.
- `internal/mcp/handlers_test.go`: `TestHandleGetCurrentUser` verifies the handler returns JSON. `TestHandleSearch_OpenOnly` verifies JQL mutation. `TestHandleSearch_WithFields` verifies the option is passed.
- `internal/mcp/server_test.go`: update expected tool list to 25.
- All existing tests remain green.

## Rollback

Single revert commit. No migrations or environment changes.
