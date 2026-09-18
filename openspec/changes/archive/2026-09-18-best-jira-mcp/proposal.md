# Proposal: best-jira-mcp

## Intent

Make `jira-mcp` the most feature-complete Jira MCP server written in Go, closing the gap with `nguyenvanduocit/jira-mcp` and differentiating from `sooperset/mcp-atlassian` (Python/Confluence). We win by being the best Go-native Jira MCP server; we do not compete on Confluence.

## Scope

### In Scope
- Issue linking: link issues, get related issues
- Issue history: get change history of an issue
- Project versions: list versions, get version
- Development information: get linked PRs/branches/commits
- Sub-tasks / child issues: create child issue
- List statuses: list available statuses for a project
- Read-only mode: `ENABLED_TOOLS` env var to expose only read tools
- CLI standalone: general-purpose `jira` CLI for terminal use

### Out of Scope
- Confluence support (deliberately not competing with `sooperset`)
- New transport protocols beyond stdio and HTTP/SSE
- Web UI or dashboards

## Capabilities

### New Capabilities
- `issue-linking`: create, list, and delete issue links
- `issue-history`: retrieve change history of an issue
- `project-versions`: list and get project versions
- `development-info`: retrieve linked PRs, branches, and commits
- `child-issues`: create sub-tasks and child issues
- `project-statuses`: list available statuses for a project
- `read-only-mode`: `ENABLED_TOOLS` env var filters exposed tools
- `standalone-cli`: `jira` CLI subcommands for direct terminal use

### Modified Capabilities
- `jira-mcp`: update README tool table and environment variables

## Approach

Add new Jira Cloud REST API endpoints behind the existing `JiraClient` interface seam. Each feature follows the established pattern: a client method in `internal/jira`, models in `internal/jira/models.go`, a tool definition in `internal/mcp/tools.go`, a handler in `internal/mcp/handlers_*.go`, and tests using the `fakeJiraClient` and `httptest` patterns. `ENABLED_TOOLS` filters the registered tool list at server startup. CLI standalone introduces a new command dispatch layer in `main.go` (or a separate `cmd/jira` entry) without breaking the existing MCP server mode.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/jira/client.go` | Modified | New REST API methods |
| `internal/jira/models.go` | Modified | New request/response models |
| `internal/mcp/tools.go` | Modified | New tool definitions |
| `internal/mcp/handlers_*.go` | Modified | New tool handlers |
| `internal/mcp/interfaces.go` | Modified | New `JiraClient` methods |
| `internal/mcp/server.go` | Modified | `ENABLED_TOOLS` filtering |
| `internal/config/config.go` | Modified | Parse `ENABLED_TOOLS` |
| `main.go` | Modified | CLI dispatch for standalone mode |
| `README.md` | Modified | Tool table and env vars |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Jira Cloud dev-info payload varies by connected app | Med | Use permissive JSON models; verify against real Jira |
| CLI standalone scope creep | Med | Start with read-only commands; defer complex writes |
| `ENABLED_TOOLS` breaks existing workflows | Low | Default to all tools; only filter when env var is set |

## Rollback Plan

Revert the merged PR branch. Because the work is delivered via chained PRs, each slice can be reverted independently without affecting the others.

## Dependencies

- Jira Cloud REST API v3 and Agile v1 access
- `mark3labs/mcp-go` v0.44.0

## Success Criteria

- [ ] All 8 feature gaps are exposed as MCP tools or CLI commands
- [ ] `go test ./...` passes with no coverage drop
- [ ] README documents new tools and `ENABLED_TOOLS`
- [ ] Release artifacts (Homebrew, Docker, install scripts) remain unchanged

## Size Estimate & Chained PRs

Estimated ~900 changed lines across the client, handlers, tests, and CLI.

**Chained PRs recommended: Yes.** Split into 3 reviewable slices:
1. Read-only tools: issue history, versions, dev-info, statuses
2. Relationship/child tools: issue linking, create child issue
3. Operational modes: `ENABLED_TOOLS` + CLI standalone

**Decision needed before apply: Yes.** Confirm the chained PR strategy before implementation.
