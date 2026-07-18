# Design: Compete with top Jira MCP servers

## Technical Approach

Extend the existing seam at `internal/mcp.JiraClient` so the real `*jira.Client` and test fakes stay interchangeable. Add Agile sprint reads to `internal/jira`, add four sprint tools/handlers to `internal/mcp`, wire an HTTP/SSE transport path in `main.go`, and ship a minimal Docker image with CI publishing. Each slice is independent and rollbackable.

## Architecture Decisions

| Decision | Options | Tradeoffs | Choice |
|---|---|---|---|
| Jira Agile API endpoint family | Data Center Agile vs Cloud Agile | Cloud target is explicit in the README; Cloud Agile endpoints are `/rest/agile/1.0/*` | Use **Jira Cloud Agile REST API v1.0** |
| Sprint scope | Require `board_id` vs global search | Jira Agile has no global sprint search; requiring `board_id` keeps the API honest and the client thin | `board_id` is required for list, active, and search-by-name tools |
| Search by name implementation | Client-side filter vs third endpoint | No native search; filtering a board's sprints is deterministic and avoids extra calls | List sprints for the board and return matching subset |
| HTTP transport | `SSEServer` vs `StreamableHTTPServer` | Proposal explicitly says HTTP/SSE; `SSEServer` is the stable MCP SSE transport | Use `mcpserver.NewSSEServer` |
| Transport selection | `--transport` flag vs env var | Flag is explicit, default stdio preserves backward compatibility | `--transport` flag (stdio default, http optional) plus `--port` for HTTP |
| HTTP server lifecycle | `Start(addr)` vs custom `http.Server` | `Start` blocks; wrapping in `http.Server` allows graceful shutdown | Add `ServeSSE(addr string) error` that uses `SSEServer.Start` and catches shutdown |
| Docker base image | `scratch` vs `alpine` | `scratch` is smaller; `alpine` eases debugging | `scratch` with copied CA certificates from builder |
| Container registry | `ghcr.io` vs Docker Hub | `ghcr.io` needs no extra secret beyond `GITHUB_TOKEN` | **GitHub Container Registry** (`ghcr.io/<owner>/jira-mcp`) |

## Data Flow

```
MCP client
    │
    ├─stdio──→ main.go ──→ mcp.Server.ServeStdio()
    │
    └─HTTP/SSE──→ main.go ──→ mcp.Server.ServeSSE(":8080")
                              │
                              └── mcp.Server.mcp (MCPServer)
                                    │
                                    └── JiraClient interface
                                          │
                                          ├── *jira.Client (real)
                                          └── fakeJiraClient (tests)
```

Sprint data flows from Jira Cloud Agile endpoints through the same client and error mapping as issue data.

## File Changes

| File | Action | Description |
|---|---|---|
| `internal/jira/models.go` | Modify | Add `Sprint`, `SprintList`, `SprintListResponse` models |
| `internal/jira/client.go` | Modify | Add `ListSprints`, `GetSprint`, `GetActiveSprint`, `SearchSprintByName` methods |
| `internal/mcp/interfaces.go` | Modify | Add four sprint methods to `JiraClient` |
| `internal/mcp/tools.go` | Modify | Add `listSprintsTool`, `getSprintTool`, `getActiveSprintTool`, `searchSprintByNameTool` |
| `internal/mcp/handlers_sprint.go` | Create | Handler implementations for the four sprint tools |
| `internal/mcp/server.go` | Modify | Register sprint tools; add `ServeSSE(addr string) error` |
| `main.go` | Modify | Add `--transport` and `--port` flags; dispatch stdio or SSE |
| `Dockerfile` | Create | Multi-stage scratch image with CA certs |
| `.github/workflows/release.yaml` | Modify | Add Docker build/push job; add `packages: write` permission |
| `README.md` | Modify | Add Docker and HTTP/SSE setup sections |
| `internal/jira/client_test.go` | Modify | Add Agile endpoint tests |
| `internal/mcp/handlers_test.go` | Modify | Extend fake client and add sprint handler tests |

## Interfaces / Contracts

```go
// JiraClient (additions only)
type JiraClient interface {
    // ... existing 11 methods ...
    ListSprints(ctx context.Context, boardID string, opts ...jira.Option) ([]jira.Sprint, error)
    GetSprint(ctx context.Context, sprintID string) (*jira.Sprint, error)
    GetActiveSprint(ctx context.Context, boardID string, opts ...jira.Option) (*jira.Sprint, error)
    SearchSprintByName(ctx context.Context, boardID string, name string, opts ...jira.Option) ([]jira.Sprint, error)
}
```

New tool schemas:

| Tool | Required | Optional | Notes |
|---|---|---|---|
| `jira_list_sprints` | `board_id` | `max_results` | Reuses `jira.Option` |
| `jira_get_sprint` | `sprint_id` | — | |
| `jira_get_active_sprint` | `board_id` | `max_results` | Filters `state=active` |
| `jira_search_sprint_by_name` | `board_id`, `name` | `max_results` | Client-side substring filter |

Agile endpoints used:
- `GET /rest/agile/1.0/board/{boardId}/sprint?maxResults={n}`
- `GET /rest/agile/1.0/board/{boardId}/sprint?state=active&maxResults={n}`
- `GET /rest/agile/1.0/sprint/{sprintId}`

HTTP/SSE contract:
- `main.go --transport http --port 8080` starts `mcpserver.NewSSEServer(mcp)` via `mcp.Server.ServeSSE`.
- `GET /sse` is the SSE endpoint; `POST /message` is the client message endpoint.
- No built-in auth beyond Jira credentials; README must warn: run only on trusted networks.

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit | `internal/jira` Agile reads | `httptest` server returns Agile payloads; assert paths, query params, model decode |
| Unit | `internal/mcp` sprint handlers | Extend `fakeJiraClient`; table-driven tests for success, missing params, and error mapping |
| Unit | Transport dispatch | `main.go` flag parsing tested via small helper or direct unit if extractable |
| Integration | HTTP/SSE server | `mcpserver.NewTestServer` or a lightweight HTTP test that hits `/sse` and verifies the server responds |
| Integration | Docker build | CI runs `docker build` as a smoke test; no runtime container tests |
| E2E | Real Jira Cloud | Not automated; manual validation with a real board and sprint |

All tests run under `go test ./...` per strict TDD.

## Migration / Rollout

No migration required. Existing stdio behavior remains default. The new Docker image and HTTP transport are additive. Rollback: revert the PR or publish a patch release without the image tag.

## Open Questions

- [ ] Should `jira_search_sprint_by_name` be case-insensitive or exact? (recommend case-insensitive substring)
- [ ] Should the HTTP server expose a health endpoint? (recommend no, keep minimal)
- [ ] Which image tag strategy? (recommend `latest` and semver tag from Git ref)
