# Delta Spec: compete-jira-mcp

## Sprint Management

### Requirement: List sprints for a board

The system MUST expose `jira_list_sprints`, which returns the paginated sprints of a Jira Agile board.

| Parameter | Required | Type | Notes |
|-----------|----------|------|-------|
| `board_id` | yes | string | Board identifier passed verbatim to Jira |
| `max_results` | no | number | Upper bound returned; forwarded to Jira |

#### Scenario: List sprints with default pagination

- GIVEN a valid `board_id`
- WHEN `jira_list_sprints` is called without `max_results`
- THEN the handler calls Jira Cloud Agile `GET /rest/agile/1.0/board/{boardId}/sprint`
- AND returns a JSON array of sprint objects

#### Scenario: List sprints with explicit limit

- GIVEN a valid `board_id` and `max_results = 10`
- WHEN `jira_list_sprints` is called
- THEN the request includes `maxResults=10`
- AND the response contains at most 10 sprints

#### Scenario: Missing board_id

- GIVEN a request without `board_id`
- WHEN `jira_list_sprints` is called
- THEN the handler returns an error stating the parameter is required

### Requirement: Get a sprint by ID

The system MUST expose `jira_get_sprint`, which returns a single sprint by its Agile sprint ID.

| Parameter | Required | Type | Notes |
|-----------|----------|------|-------|
| `sprint_id` | yes | string | Sprint identifier passed verbatim |

#### Scenario: Existing sprint

- GIVEN a valid `sprint_id`
- WHEN `jira_get_sprint` is called
- THEN the handler calls `GET /rest/agile/1.0/sprint/{sprintId}`
- AND returns the sprint object as JSON

#### Scenario: Missing sprint_id

- GIVEN a request without `sprint_id`
- WHEN `jira_get_sprint` is called
- THEN the handler returns a required-parameter error

### Requirement: Get active sprint for a board

The system MUST expose `jira_get_active_sprint`, which returns the active sprint of a board, if any.

| Parameter | Required | Type | Notes |
|-----------|----------|------|-------|
| `board_id` | yes | string | Board identifier |
| `max_results` | no | number | Forwarded to Jira |

#### Scenario: Board has an active sprint

- GIVEN a board with one active sprint
- WHEN `jira_get_active_sprint` is called
- THEN the handler requests `state=active`
- AND returns the active sprint object

#### Scenario: No active sprint

- GIVEN a board with no active sprint
- WHEN `jira_get_active_sprint` is called
- THEN the handler returns an empty result indicating no active sprint

### Requirement: Search sprints by name

The system MUST expose `jira_search_sprint_by_name`, which performs a case-insensitive substring match against a board's sprints and returns all matches.

| Parameter | Required | Type | Notes |
|-----------|----------|------|-------|
| `board_id` | yes | string | Board identifier |
| `name` | yes | string | Substring filter, case-insensitive |
| `max_results` | no | number | Forwarded to Jira list endpoint |

#### Scenario: Matching substring

- GIVEN a board with sprints "Sprint 1", "sprint 2", "Backlog"
- WHEN `jira_search_sprint_by_name` is called with `name = "sprint"`
- THEN the handler returns the first two sprints

#### Scenario: No matches

- GIVEN a board with sprints "Sprint 1", "Sprint 2"
- WHEN `jira_search_sprint_by_name` is called with `name = "march"`
- THEN the handler returns an empty array

#### Scenario: Missing name

- GIVEN a request with `board_id` but no `name`
- WHEN `jira_search_sprint_by_name` is called
- THEN the handler returns a required-parameter error

### Requirement: Sprint error handling

Sprint handlers MUST reuse the existing client-error mapping so that HTTP 4xx/5xx responses produce sanitized, non-leaking tool errors.

#### Scenario: Board not found

- GIVEN a `board_id` that Jira responds to with HTTP 404
- WHEN any sprint list/read tool is called
- THEN the handler returns a tool error containing the Jira message
- AND no credentials are present in the error text

## Docker Packaging

### Requirement: Minimal Docker image

The system MUST ship a `Dockerfile` that builds a dependency-free image containing the `jira-mcp` binary.

#### Scenario: Build from source

- GIVEN a clean checkout
- WHEN `docker build -t jira-mcp .` runs
- THEN the build succeeds and the image contains a runnable binary

#### Scenario: Runtime with environment variables

- GIVEN a built image
- WHEN the container runs with `JIRA_URL`, `JIRA_USERNAME`, and `JIRA_API_TOKEN`
- THEN the binary starts and connects to the configured Jira instance

### Requirement: CI image publishing

The release workflow MUST publish the Docker image to GitHub Container Registry on every semver tag.

| Tag source | Image tags |
|------------|------------|
| Git ref `v1.2.3` | `ghcr.io/<owner>/jira-mcp:1.2.3`, `ghcr.io/<owner>/jira-mcp:latest` |

#### Scenario: Tag push publishes image

- GIVEN a pushed tag `v0.2.0`
- WHEN the release workflow runs
- THEN `ghcr.io/<owner>/jira-mcp:0.2.0` and `:latest` are pushed
- AND the workflow has `packages: write` permission

## HTTP/SSE Transport

### Requirement: Transport selection

The system MUST support stdio (default) and HTTP/SSE transports selected by a `--transport` flag.

| Flag | Behavior |
|------|----------|
| `--transport stdio` | Starts the existing stdio JSON-RPC server |
| `--transport http` | Starts an SSE MCP server |
| `--port <n>` | HTTP listen port; default `8080` |

#### Scenario: Default stdio start

- GIVEN no `--transport` flag
- WHEN the binary starts
- THEN it serves stdio as before

#### Scenario: HTTP/SSE start

- GIVEN `--transport http --port 8080`
- WHEN the binary starts
- THEN it exposes `GET /sse` for SSE and `POST /message` for client messages

### Requirement: No health endpoint

The HTTP/SSE server MUST NOT expose a dedicated health endpoint; clients rely on the SSE endpoint for liveness.

#### Scenario: Minimal surface

- GIVEN the server is running in HTTP mode
- WHEN `GET /health` is requested
- THEN it returns HTTP 404

### Requirement: Trusted-network documentation

The README MUST state that the HTTP/SSE transport carries no built-in auth and MUST run only on trusted networks.

#### Scenario: README covers HTTP security

- GIVEN the README is updated
- WHEN the HTTP/SSE section is read
- THEN it includes a warning about trusted-network-only deployment

## Acceptance Criteria

- `go test ./...` passes with no coverage drop.
- Sprint tools return correct Jira Agile data.
- `docker build` and `docker run` work with environment variables.
- `--transport http` starts an HTTP/SSE MCP server.
- README covers Docker and HTTP/SSE setup.
