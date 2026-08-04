# Proposal: workflow-friction

## Intent

Remove the friction an LLM experiences when trying to find issues assigned to the authenticated user. The current workflow requires multiple round-trips to discover the user's display name and then craft JQL. We fix this by exposing the authenticated identity directly and by giving `jira_search` an ergonomic "open issues" filter.

## Scope

### In Scope
- New read-only tool: `jira_get_current_user`
- Extend `jira_search` with optional `open_only` boolean parameter
- Extend `jira_search` with optional `fields` string array parameter
- Update README tool table and search examples
- Tests and build pass

### Out of Scope
- New write or admin tools
- CLI changes
- Landing page deployment
- Paginating large search results beyond existing `max_results`

## Capabilities

### New Capabilities
- `current-user`: expose the authenticated Jira user's `accountId`, `displayName`, and `emailAddress`.

### Modified Capabilities
- `jira_search`: optionally restrict results to unresolved issues (`open_only`) and optionally request a custom field set (`fields`).

## Approach

Add a `GetCurrentUser` method to the existing `JiraClient` seam, following the established pattern: model in `internal/jira/models.go`, client method in `internal/jira/client.go`, tool in `internal/mcp/tools.go`, handler in `internal/mcp/handlers_read.go`, and tests with `httptest` and `fakeJiraClient`. Extend `jira_search` by adding two optional parameters and the minimal client-side option `WithFields`. The `open_only` filter is applied by the handler appending `AND resolution = Unresolved` to the JQL string. The `fields` parameter is passed through to the Jira search body so callers can request lighter payloads. README is updated to document the new tool and the improved search.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/jira/models.go` | Modified | Add `CurrentUser` model |
| `internal/jira/client.go` | Modified | Add `GetCurrentUser` and `WithFields` option |
| `internal/mcp/interfaces.go` | Modified | Add `GetCurrentUser` to `JiraClient` |
| `internal/mcp/tools.go` | Modified | Add `getCurrentUserTool` and update `searchTool` |
| `internal/mcp/handlers_read.go` | Modified | Add `handleGetCurrentUser` and update `handleSearch` |
| `internal/mcp/helpers.go` | Modified | Add `optionalBool` and `optionalStringSlice` helpers |
| `internal/mcp/server.go` | Modified | Register `jira_get_current_user`; tool count becomes 25 |
| `internal/mcp/server_test.go` | Modified | Expect 25 tools |
| `internal/mcp/handlers_test.go` | Modified | Fake client + handler tests |
| `internal/jira/client_test.go` | Modified | `TestClient_GetCurrentUser` and `TestClient_Search_WithFields` |
| `README.md` | Modified | Tool table + search examples |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `fields` array not typed correctly by MCP clients | Low | Use `mcp.WithArray` with `mcp.Items` of type `string` |
| Appending `AND resolution = Unresolved` to user JQL produces invalid JQL | Low | Only append when `open_only=true`; trim trailing whitespace from JQL first |
| `myself` endpoint returns fewer fields on some Jira Cloud plans | Low | Mark `emailAddress` as optional in the model |

## Rollback Plan

Revert the single commit that adds these changes. The work is read-only and additive; no existing tool contracts change.

## Dependencies

- Jira Cloud REST API v3 `/rest/api/3/myself` endpoint
- Existing `JiraClient` interface seam
- `mark3labs/mcp-go` v0.44.0

## Success Criteria

- [ ] `jira_get_current_user` returns accountId, displayName, and emailAddress
- [ ] `jira_search` with `open_only=true` searches `resolution = Unresolved`
- [ ] `jira_search` with `fields` requests only those fields in the search body
- [ ] `go test ./...` passes
- [ ] README documents the new tool and search parameters
- [ ] Tool count is 25

## Size Estimate & Chained PRs

Estimated ~250 changed lines across models, client, handlers, helpers, tests, and README.

**Chained PRs recommended: No.** This is a single small slice that can merge directly to `master` under `size:exception`.

**Decision needed before apply: No.**
