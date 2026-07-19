# Tasks: best-jira-mcp

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~900 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | Slice 1 → Slice 2 → Slice 3 |
| Delivery strategy | single-pr |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | PR | Notes |
|------|------|-----|-------|
| 1 | Read-only tools | PR 1 | history, versions, dev-info, statuses |
| 2 | Relationship/child tools | PR 2 | issue linking, create child issue |
| 3 | Operational modes | PR 3 | ENABLED_TOOLS + CLI standalone |

## Phase 1: Slice 1 — Read-only Tools

- [ ] 1.1 Add read-only models in `internal/jira/models.go` (`Changelog`, `Version`, `DevelopmentInformation`, `ProjectStatus`, `Status`, etc.) and extend `internal/mcp/interfaces.go` `JiraClient` with `GetIssueHistory`, `ListProjectVersions`, `GetVersion`, `GetDevelopmentInfo`, `ListStatuses`.
- [ ] 1.2 Implement client methods in `internal/jira/client.go`: `GetIssueHistory`, `ListProjectVersions`, `GetVersion`, `GetDevelopmentInfo`, `ListStatuses` with correct endpoints; add `TestClient_GetIssueHistory`, `TestClient_ListProjectVersions`, `TestClient_GetVersion`, `TestClient_GetDevelopmentInfo`, `TestClient_ListStatuses` in `internal/jira/client_test.go`.
- [ ] 1.3 Add tool constructors in `internal/mcp/tools.go`: `getIssueHistoryTool`, `listProjectVersionsTool`, `getVersionTool`, `getDevelopmentInfoTool`, `listStatusesTool`.
- [ ] 1.4 Create handler files `internal/mcp/handlers_history.go`, `internal/mcp/handlers_version.go`, `internal/mcp/handlers_dev.go`, `internal/mcp/handlers_status.go` with handlers matching the spec; add tests in `internal/mcp/handlers_test.go`.
- [ ] 1.5 Register the 5 new tools in `internal/mcp/server.go` `toolDefinitions`; update `TestServer_ToolList` in `internal/mcp/server_test.go` to expect 21 tools.

## Phase 2: Slice 2 — Relationship/Child Tools

- [ ] 2.1 Add write models in `internal/jira/models.go` (`IssueLinkRequest`, `IssueLink`, `LinkedIssue`, `CreateChildIssueRequest` with custom `MarshalJSON`, `parseIssueLinks` helper) and extend `internal/mcp/interfaces.go` `JiraClient` with `CreateIssueLink`, `GetRelatedIssues`, `CreateChildIssue`.
- [ ] 2.2 Implement client methods in `internal/jira/client.go`: `CreateIssueLink` (`POST /rest/api/3/issueLink`), `GetRelatedIssues` (`GET /rest/api/3/issue/{key}?fields=issuelinks`), `CreateChildIssue` (`POST /rest/api/3/issue` with `fields.parent.key`); add tests in `internal/jira/client_test.go`.
- [ ] 2.3 Add tool constructors in `internal/mcp/tools.go`: `createIssueLinkTool`, `getRelatedIssuesTool`, `createChildIssueTool`.
- [ ] 2.4 Create `internal/mcp/handlers_link.go` with `handleCreateIssueLink` and `handleGetRelatedIssues`, and `internal/mcp/handlers_child.go` with `handleCreateChildIssue`; add tests in `internal/mcp/handlers_test.go`.
- [ ] 2.5 Register the 3 new tools in `internal/mcp/server.go` `toolDefinitions`; update `TestServer_ToolList` to expect 24 tools.

## Phase 3: Slice 3 — Operational Modes

- [ ] 3.1 Add `EnabledTools []string` to `internal/config/config.go` `Config`, parse `ENABLED_TOOLS` with `splitAndTrim`, and add `TestLoad_EnabledTools` in `internal/config/config_test.go`.
- [ ] 3.2 Change `internal/mcp/server.go` `NewServer(client JiraClient, enabledTools []string)` to filter `toolDefinitions` by exact name; add `TestServer_ToolList_Filtered`; update `NewServer` calls in `main.go` and `internal/mcp/server_test.go` to pass `cfg.EnabledTools`.
- [ ] 3.3 Create `internal/cli/cli.go` with `Run(args []string, client JiraClient)` dispatch, local `JiraClient` interface, `writeJSON`, and `--pretty` flag; create `internal/cli/commands.go` with the 12 read-only commands from the spec; add `internal/cli/cli_test.go` with fake-client tests.
- [ ] 3.4 Add `detectCLI` and CLI dispatch in `main.go` preserving the default MCP server mode; add tests in `main_test.go`.
- [ ] 3.5 Update `README.md` with the 8 new tools, the `ENABLED_TOOLS` row, and CLI examples; run `go test ./...` and `go build ./...` to verify.
