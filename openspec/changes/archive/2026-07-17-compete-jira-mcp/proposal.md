# Proposal: Compete with top Jira MCP servers

## Intent

Close the feature gap with leading Jira MCP servers while exploiting our advantage: a single, dependency-free Go binary. Add sprint management, Docker packaging, and HTTP transport.

## Scope

### In Scope

- Sprint tools: `jira_list_sprints`, `jira_get_sprint`, `jira_get_active_sprint`, `jira_search_sprint_by_name`.
- Docker support: `Dockerfile`, CI publishing, docs.
- HTTP/SSE transport alongside existing stdio.

### Out of Scope

- Issue linking, issue history, and project versions: deferred to a follow-up change.

## Capabilities

### New Capabilities

- `sprint-management`: list, fetch, and search Agile sprints.
- `docker-support`: build a minimal container image.
- `http-transport`: expose the MCP server over HTTP/SSE.

### Modified Capabilities

- None.

## Approach

Extend `internal/jira` with sprint endpoints and `JiraClient` with methods. Add sprint tools and handlers in `internal/mcp`. Add a `--transport` flag in `main.go` selecting stdio (default) or HTTP/SSE. Provide a `Dockerfile`, update CI to publish the image, and document Docker and HTTP setup.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/jira/client.go` | Modified | Sprint methods. |
| `internal/jira/models.go` | Modified | Sprint models. |
| `internal/mcp/interfaces.go` | Modified | Sprint methods on `JiraClient`. |
| `internal/mcp/tools.go` | Modified | Register sprint tools. |
| `internal/mcp/handlers_sprint.go` | New | Sprint handlers. |
| `main.go` | Modified | `--transport` flag and HTTP wiring. |
| `Dockerfile` | New | Multi-stage image. |
| `.github/workflows/release.yaml` | Modified | Publish Docker image on tag. |
| `README.md` | Modified | Docker and HTTP sections. |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| HTTP transport leaks credentials if exposed | Med | Document trusted-network-only use; reuse existing auth handling. |
| Jira Agile API differs from Data Center | Low | Target Jira Cloud only; document the assumption. |
| Docker/CI image push breaks release | Low | Test Dockerfile locally; add dry-run CI step. |

## Rollback Plan

1. Revert merged PR(s) or ship a patch release without new artifacts.
2. Remove or deprecate the published image tag if needed.
3. Homebrew changes are additive, so no formula rollback is needed unless the binary breaks.

## Dependencies

- `github.com/mark3labs/mcp-go` already supports HTTP/SSE.
- Docker push requires GitHub Packages or Docker Hub access.

## Success Criteria

- [ ] `go test ./...` passes with no coverage drop.
- [ ] Sprint tools return correct Jira Agile data.
- [ ] `docker build` and `docker run` work with env vars.
- [ ] `--transport http` starts an HTTP/SSE MCP server.
- [ ] README covers Docker and HTTP setup.

## Size Estimate

Estimated 400–700 changed lines. **Chained PRs recommended: Yes** — deliver sprint tools, Docker packaging, and HTTP transport as independent, rollbackable slices.
