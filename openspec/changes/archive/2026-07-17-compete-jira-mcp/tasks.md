# Tasks: Compete with top Jira MCP servers

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 500–750 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1: Sprint tools → PR 2: HTTP/SSE transport → PR 3: Docker packaging + docs |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Sprint management tools (client + handlers + tests) | PR 1 | Base `main`; no transport or Docker changes |
| 2 | HTTP/SSE transport (`--transport`, `--port`, tests) | PR 2 | Base PR 1 branch |
| 3 | Docker image + CI publish + README | PR 3 | Base PR 2 branch |

## Phase 1: Sprint Foundation

- [x] 1.1 Add `Sprint`, `SprintList`, and `SprintListResponse` models to `internal/jira/models.go`.
- [x] 1.2 Add `ListSprints`, `GetSprint`, `GetActiveSprint`, and `SearchSprintByName` to `internal/jira/client.go` using Agile endpoints.
- [x] 1.3 Add the four sprint methods to `internal/mcp/interfaces.go` and verify `*jira.Client` satisfies `JiraClient`.
- [x] 1.4 Add `WithState` option in `internal/jira/client.go` for active-sprint queries.

## Phase 2: Sprint Tools and Tests

- [x] 2.1 Add `listSprintsTool`, `getSprintTool`, `getActiveSprintTool`, and `searchSprintByNameTool` to `internal/mcp/tools.go`.
- [x] 2.2 Create `internal/mcp/handlers_sprint.go` with handlers for the four sprint tools.
- [x] 2.3 Register the four sprint tools in `internal/mcp/server.go`.
- [x] 2.4 Write `internal/jira/client_test.go` tests for Agile endpoints, paths, query params, and error mapping.
- [x] 2.5 Update `internal/mcp/handlers_test.go` fake client and add table-driven sprint handler tests.

## Phase 3: HTTP/SSE Transport

- [x] 3.1 Add `ServeSSE(addr string) error` to `internal/mcp/server.go` using `mcpserver.NewSSEServer`.
- [x] 3.2 Add `--transport` and `--port` flags to `main.go` and dispatch `stdio` (default) or `http`.
- [x] 3.3 Add unit tests for transport flag parsing/dispatch in `main.go` or a small extracted helper.
- [x] 3.4 Add integration test in `internal/mcp` that starts the SSE server and asserts `/sse` responds and `/health` returns 404.

## Phase 4: Docker Packaging

- [x] 4.1 Create a multi-stage `Dockerfile` using `scratch` and copied CA certificates.
- [x] 4.2 Add a Docker build/push job to `.github/workflows/release.yaml` with `packages: write`.
- [x] 4.3 Verify the image publishes `ghcr.io/<owner>/jira-mcp:<tag>` and `:latest` on semver tag pushes.

## Phase 5: Documentation

- [x] 5.1 Add Docker build/run instructions to `README.md`.
- [x] 5.2 Add HTTP/SSE setup section with a trusted-network-only warning to `README.md`.
- [x] 5.3 Update the tools table in `README.md` with the four new sprint tools.

## Phase 6: Verification

- [x] 6.1 Run `go test ./...` and ensure all new and existing tests pass.
- [x] 6.2 Run `go vet ./...` and `gofmt -w .`.
- [x] 6.3 Run `docker build -t jira-mcp .` locally.
- [x] 6.4 Smoke-test `--transport http --port 8080` on a local port.
