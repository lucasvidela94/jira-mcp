# Tasks: workflow-friction

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~250 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single commit/PR |
| Delivery strategy | single-pr |
| Chain strategy | N/A |

Decision needed before apply: No

## Task 1 — Add `CurrentUser` model and client method

Files: `internal/jira/models.go`, `internal/jira/client.go`, `internal/jira/client_test.go`

- [ ] 1.1 In `internal/jira/models.go`, add the `CurrentUser` struct:
  ```go
  type CurrentUser struct {
      AccountID    string `json:"accountId"`
      DisplayName  string `json:"displayName"`
      EmailAddress string `json:"emailAddress,omitempty"`
      Active       bool   `json:"active"`
  }
  ```
- [ ] 1.2 In `internal/jira/client.go`, add `GetCurrentUser`:
  ```go
  func (c *Client) GetCurrentUser(ctx context.Context) (*CurrentUser, error) {
      resp, err := c.doWithRetry(ctx, http.MethodGet, "/rest/api/3/myself", nil, nil)
      if err != nil {
          return nil, err
      }
      var user CurrentUser
      if err := handleResponse(resp, &user); err != nil {
          return nil, err
      }
      return &user, nil
  }
  ```
- [ ] 1.3 In `internal/jira/client.go`, add `WithFields` option:
  ```go
  func WithFields(fields []string) Option {
      return func(v *url.Values) {
          for _, f := range fields {
              v.Add("fields", f)
          }
      }
  }
  ```
- [ ] 1.4 In `internal/jira/client.go`, modify `Search` to extract custom fields from `url.Values`:
  ```go
  defaultFields := []string{"id", "key", "summary", "status", "assignee", "issuetype"}
  req := SearchRequest{JQL: jql, Fields: defaultFields}
  query := url.Values{}
  for _, opt := range opts { opt(&query) }
  if fieldList, ok := query["fields"]; ok && len(fieldList) > 0 {
      req.Fields = fieldList
      delete(query, "fields")
  }
  if v := query.Get("maxResults"); v != "" {
      if n, err := strconv.Atoi(v); err == nil {
          req.MaxResults = n
      }
  }
  ```
- [ ] 1.5 In `internal/jira/client_test.go`, add `TestClient_GetCurrentUser`:
  - Verify `GET /rest/api/3/myself`.
  - Return JSON with `accountId`, `displayName`, `emailAddress`, `active`.
  - Assert the returned `CurrentUser` fields match.
- [ ] 1.6 In `internal/jira/client_test.go`, add `TestClient_Search_WithFields`:
  - Verify `POST /rest/api/3/search/jql` body contains `fields: ["key", "summary"]`.
  - Call `client.Search(ctx, "project = PROJ", WithFields([]string{"key", "summary"}))`.
- [ ] 1.7 Run `go test ./internal/jira/...` and ensure it passes.

## Task 2 — Update `JiraClient` interface

Files: `internal/mcp/interfaces.go`, `internal/mcp/handlers_test.go`

- [ ] 2.1 In `internal/mcp/interfaces.go`, add `GetCurrentUser(ctx context.Context) (*jira.CurrentUser, error)` to the `JiraClient` interface.
- [ ] 2.2 In `internal/mcp/handlers_test.go`, add the matching method to `fakeJiraClient`:
  ```go
  getCurrentUserFn func(ctx context.Context) (*jira.CurrentUser, error)
  ```
  and
  ```go
  func (f *fakeJiraClient) GetCurrentUser(ctx context.Context) (*jira.CurrentUser, error) {
      return f.getCurrentUserFn(ctx)
  }
  ```
- [ ] 2.3 Run `go test ./internal/mcp/...` and ensure it compiles.

## Task 3 — Add handler helpers

Files: `internal/mcp/helpers.go`

- [ ] 3.1 Add `optionalBool`:
  ```go
  func optionalBool(args map[string]any, name string) (bool, bool, error) {
      v, ok := args[name]
      if !ok {
          return false, false, nil
      }
      b, ok := v.(bool)
      if !ok {
          return false, false, fmt.Errorf("parameter %s must be a boolean", name)
      }
      return b, true, nil
  }
  ```
- [ ] 3.2 Add `optionalStringSlice`:
  ```go
  func optionalStringSlice(args map[string]any, name string) ([]string, bool, error) {
      v, ok := args[name]
      if !ok {
          return nil, false, nil
      }
      raw, ok := v.([]any)
      if !ok {
          return nil, false, fmt.Errorf("parameter %s must be an array", name)
      }
      out := make([]string, 0, len(raw))
      for _, item := range raw {
          s, ok := item.(string)
          if !ok {
              return nil, false, fmt.Errorf("parameter %s must contain only strings", name)
          }
          out = append(out, s)
      }
      return out, true, nil
  }
  ```
- [ ] 3.3 Run `go test ./internal/mcp/...`.

## Task 4 — Add `jira_get_current_user` tool and handler

Files: `internal/mcp/tools.go`, `internal/mcp/handlers_read.go`, `internal/mcp/handlers_test.go`

- [ ] 4.1 In `internal/mcp/tools.go`, add `getCurrentUserTool`:
  ```go
  func getCurrentUserTool() mcp.Tool {
      return mcp.NewTool("jira_get_current_user",
          mcp.WithDescription("Get the currently authenticated Jira user."),
      )
  }
  ```
- [ ] 4.2 In `internal/mcp/handlers_read.go`, add `handleGetCurrentUser`:
  ```go
  func (s *Server) handleGetCurrentUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      user, err := s.client.GetCurrentUser(ctx)
      if err != nil {
          return handleClientError(err), nil
      }
      return resultJSON(user), nil
  }
  ```
- [ ] 4.3 In `internal/mcp/handlers_test.go`, add `TestHandleGetCurrentUser`:
  - Set `fakeJiraClient.getCurrentUserFn` to return a `CurrentUser`.
  - Call `srv.handleGetCurrentUser`.
  - Assert the JSON result contains `displayName`.
- [ ] 4.4 Run `go test ./internal/mcp/...`.

## Task 5 — Extend `jira_search` with `open_only` and `fields`

Files: `internal/mcp/tools.go`, `internal/mcp/handlers_read.go`, `internal/mcp/handlers_test.go`

- [ ] 5.1 In `internal/mcp/tools.go`, update `searchTool`:
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
- [ ] 5.2 In `internal/mcp/handlers_read.go`, update `handleSearch` to read the new parameters and mutate the JQL when `open_only` is true:
  ```go
  func (s *Server) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      args := request.GetArguments()
      jql, err := requiredString(args, "jql")
      if err != nil {
          return nil, err
      }

      if openOnly, ok, err := optionalBool(args, "open_only"); err != nil {
          return nil, err
      } else if ok && openOnly {
          jql = strings.TrimSpace(jql) + " AND resolution = Unresolved"
      }

      opts := []jira.Option{}
      if maxResults, ok, err := optionalInt(args, "max_results"); err != nil {
          return nil, err
      } else if ok {
          opts = append(opts, jira.WithMaxResults(maxResults))
      }
      if fields, ok, err := optionalStringSlice(args, "fields"); err != nil {
          return nil, err
      } else if ok {
          opts = append(opts, jira.WithFields(fields))
      }

      result, err := s.client.Search(ctx, jql, opts...)
      if err != nil {
          return handleClientError(err), nil
      }
      return resultJSON(result)
  }
  ```
  Add `strings` import if not already present.
- [ ] 5.3 In `internal/mcp/handlers_test.go`, add `TestHandleSearch_OpenOnly`:
  - `fakeJiraClient.searchFn` asserts `jql == "assignee = currentUser() AND resolution = Unresolved"`.
  - Call `srv.handleSearch` with `jql: "assignee = currentUser()"`, `open_only: true`.
  - Assert no error.
- [ ] 5.4 In `internal/mcp/handlers_test.go`, add `TestHandleSearch_WithFields`:
  - `fakeJiraClient.searchFn` captures the options, verify `jira.WithFields` is present by checking the request body in an integration test, or simply assert the handler does not error.
  - For a more robust test, add an option-inspection helper in the fake that records the last `opts` slice.
- [ ] 5.5 Run `go test ./internal/mcp/...`.

## Task 6 — Register the new tool and update server tests

Files: `internal/mcp/server.go`, `internal/mcp/server_test.go`

- [ ] 6.1 In `internal/mcp/server.go`, add `jira_get_current_user` to `toolDefinitions`:
  ```go
  {tool: getCurrentUserTool(), handler: s.handleGetCurrentUser},
  ```
- [ ] 6.2 In `internal/mcp/server_test.go`, update `TestServer_ToolList` to expect 25 tools and add `"jira_get_current_user"` to the `expected` slice.
- [ ] 6.3 Run `go test ./internal/mcp/...`.

## Task 7 — Update README

Files: `README.md`

- [ ] 7.1 Add `jira_get_current_user` to the tool table.
- [ ] 7.2 Update the `jira_search` row to mention `open_only` and `fields`.
- [ ] 7.3 Add an example in the README showing:
  ```json
  {"jql": "assignee = currentUser()", "open_only": true}
  ```
  or
  ```json
  {"jql": "project = PROJ", "fields": ["key", "summary", "status"]}
  ```
- [ ] 7.4 Run `go test ./...` and `go build ./...` after README changes (no code impact).

## Task 8 — Final verification

- [ ] 8.1 Run `go test ./...` and ensure all tests pass.
- [ ] 8.2 Run `go build ./...` and ensure it compiles.
- [ ] 8.3 Run `go vet ./...` and ensure no issues.
- [ ] 8.4 Run `gofmt -w .` on all modified files.
- [ ] 8.5 Commit the change with a conventional commit message such as:
  ```
  feat: add jira_get_current_user and improve jira_search with open_only and fields
  ```
- [ ] 8.6 Push to `master` (solo workflow, single PR).

## Task 9 — Optional: smoke test against real Jira

- [ ] 9.1 Build the binary: `go build -o jira-mcp ./`.
- [ ] 9.2 Run the MCP server with environment variables set.
- [ ] 9.3 Call `jira_get_current_user` and confirm it returns your identity.
- [ ] 9.4 Call `jira_search` with `jql: "assignee = currentUser()"`, `open_only: true`, and verify the returned issues are unresolved.
- [ ] 9.5 Call `jira_search` with `fields: ["key", "summary"]` and confirm the response is lighter.

## Notes for the Executor

- Do not change the `Search` method signature in the `JiraClient` interface.
- Do not introduce new dependencies.
- Keep the change strictly additive; no existing tool behavior should change unless the caller explicitly passes `open_only` or `fields`.
- The `fields` option is passed through `url.Values` under the synthetic key `"fields"` and extracted in `jira.Client.Search`. This is intentional to keep `jira.Option` uniform.
- Remember to update the `fakeJiraClient` in tests; otherwise the code will not compile.
- If `mcp.WithArray` requires `mcp.Items` configuration, use the simplest schema that accepts an array of strings.
