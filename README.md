# jira-mcp

Model Context Protocol (MCP) server for Jira Cloud. Lets you search, create, update, transition, assign, delete, comment, and log work on Jira issues — directly from any MCP-compatible AI client.

[![Go Version](https://img.shields.io/github/go-mod/go-version/lucasvidela94/jira-mcp)](https://github.com/lucasvidela94/jira-mcp)
[![License](https://img.shields.io/github/license/lucasvidela94/jira-mcp)](LICENSE)
[![Release](https://img.shields.io/github/v/release/lucasvidela94/jira-mcp)](https://github.com/lucasvidela94/jira-mcp/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/lucasvidela94/jira-mcp)](https://goreportcard.com/report/github.com/lucasvidela94/jira-mcp)

---

## Features

- **Full CRUD** on issues — create, read, update, delete
- **Transition** issues through workflows
- **Assign** issues to users
- **Add comments** and **log work**
- **JQL search** with configurable result limits
- **List projects** and **available transitions**
- **Sprint management** — list, get, and search Jira Agile sprints
- **OAuth 2.0 login** — browser-based authentication, no manual token creation
- **Safe by design** — credentials are never logged, returned, or leaked
- **Multi-platform** — macOS, Linux, Windows, ARM64
- **Docker-ready** — minimal `scratch` image
- **HTTP/SSE transport** — run as a remote MCP server on a trusted network

## Authentication

jira-mcp supports two authentication methods:

### OAuth 2.0 (Recommended)

Browser-based login with automatic token refresh. No manual token creation required.

```bash
# Set your Jira URL
export JIRA_URL=https://yourcompany.atlassian.net

# Login via browser
jira-mcp auth

# Check token status
jira-mcp auth --status

# Logout and clear cached tokens
jira-mcp auth --logout
```

Tokens are stored in `~/.config/jira-mcp/token.json` with `0600` permissions and automatically refreshed.

### Basic Auth (Fallback)

Traditional email + API token authentication via environment variables.

Create or verify your Jira API token at: https://id.atlassian.com/manage-profile/security/api-tokens

```bash
export JIRA_URL=https://yourcompany.atlassian.net
export JIRA_USERNAME=you@example.com
export JIRA_API_TOKEN=your-api-token
```

> **Note:** If both OAuth tokens and Basic Auth environment variables are present, Basic Auth takes precedence.

## Installation

### Homebrew (macOS / Linux)

```bash
brew tap lucasvidela94/tap
brew install jira-mcp
```

### One-liner installer (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.sh | bash
```

The installer can optionally configure your MCP client automatically. When run in a terminal, it asks which client you want to set up, prompts for your Jira URL, and then asks whether you want to use OAuth (recommended) or Basic Auth.

**OAuth flow:**
```text
Which MCP client do you want to configure? [opencode/claude/cursor/windsurf/none] (default: opencode):
Jira URL (e.g. https://yourcompany.atlassian.net): https://yourcompany.atlassian.net
Authentication method [oauth/basic] (default: oauth): oauth
Proceed? [Y/n] y
```

After installation, run `jira-mcp auth` to complete the OAuth login via browser.

**Basic Auth flow:**
```text
Which MCP client do you want to configure? [opencode/claude/cursor/windsurf/none] (default: opencode):
Jira URL (e.g. https://yourcompany.atlassian.net): https://yourcompany.atlassian.net
Authentication method [oauth/basic] (default: oauth): basic
Jira username/email: you@example.com
Jira API token:
Proceed? [Y/n] y
```

Create or verify your Jira API token at: https://id.atlassian.com/manage-profile/security/api-tokens

Skip the wizard or pass the answers as flags:

```bash
curl -fsSL https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.sh | bash -s -- --no-configure
```

```bash
curl -fsSL https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.sh | bash -s -- \
  --client opencode \
  --jira-url https://yourcompany.atlassian.net \
  --auth-method oauth \
  --yes
```

Or with Basic Auth:

```bash
curl -fsSL https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.sh | bash -s -- \
  --client opencode \
  --jira-url https://yourcompany.atlassian.net \
  --auth-method basic \
  --jira-username you@example.com \
  --jira-api-token your-api-token \
  --yes
```

### Docker

Build the image locally:

```bash
docker build -t jira-mcp .
```

**OAuth mode:** Mount your token file and set `JIRA_URL`:

```bash
docker run -v ~/.config/jira-mcp:/root/.config/jira-mcp \
  -e JIRA_URL=https://yourcompany.atlassian.net \
  jira-mcp
```

> **Note:** Run `jira-mcp auth` on your host machine first to generate the token file.

**Basic Auth mode:** Pass all credentials as environment variables:

```bash
docker run -e JIRA_URL=https://yourcompany.atlassian.net \
  -e JIRA_USERNAME=you@example.com \
  -e JIRA_API_TOKEN=your-api-token \
  jira-mcp
```

To use the HTTP/SSE transport inside a container:

```bash
docker run -p 8080:8080 \
  -e JIRA_URL=https://yourcompany.atlassian.net \
  -e JIRA_USERNAME=you@example.com \
  -e JIRA_API_TOKEN=your-api-token \
  jira-mcp --transport http --port 8080
```

The published image is available from GitHub Container Registry:

```bash
docker pull ghcr.io/lucasvidela94/jira-mcp:latest
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.ps1 | iex
```

The PowerShell installer supports the same optional auto-configuration when run interactively. To force or skip the wizard, or to pass the values as parameters, save the script first:

**OAuth:**
```powershell
irm https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.ps1 -OutFile install.ps1
.\install.ps1 -Configure -Client opencode -JiraUrl https://yourcompany.atlassian.net -AuthMethod oauth -Yes
```

**Basic Auth:**
```powershell
irm https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.ps1 -OutFile install.ps1
.\install.ps1 -Configure -Client opencode -JiraUrl https://yourcompany.atlassian.net -AuthMethod basic -JiraUsername you@example.com -JiraApiToken your-api-token -Yes
```

### Go install

```bash
go install github.com/lucasvidela94/jira-mcp@latest
```

This installs the binary only; you will still need to configure your MCP client manually.

### Updating

Because `jira-mcp` is a background MCP server, your MCP client does not update it automatically. Use the same channel you used to install it:

| Install method | Update command |
|---|---|
| Homebrew | `brew upgrade jira-mcp` |
| One-liner installer | Re-run the install script |
| Docker | `docker pull ghcr.io/lucasvidela94/jira-mcp:latest` |
| Go install | `go install github.com/lucasvidela94/jira-mcp@latest` |

After updating, **restart your MCP client** so it reloads the new binary. Verify the version with:

```bash
jira-mcp --version
```

You can also update the binary directly from the CLI:

```bash
jira-mcp update
```

This checks GitHub releases, downloads the latest asset for your platform, verifies the checksum, and replaces the running binary. After it finishes, restart your MCP client.

### Review the installer before running it

If you prefer to inspect the script instead of piping it directly to your shell:

```bash
curl -fsSL https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.sh -o install.sh
# Review the file, then run it:
bash install.sh
```

Or use the built-in inspect flag:

```bash
curl -fsSL https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.sh | bash -s -- --inspect
```

## Configuration

### OAuth Mode (Recommended)

Only `JIRA_URL` is required. Run `jira-mcp auth` once to complete the login flow.

| Variable | Description | Example |
|---|---|---|
| `JIRA_URL` | Your Jira Cloud base URL | `https://yourcompany.atlassian.net` |
| `ENABLED_TOOLS` | Optional comma-separated tool allowlist | `jira_search,jira_get_issue,jira_list_projects` |
| `JIRA_MCP_CONFIRM` | Confirmation guardrail for destructive tools (`on` by default) | `off` |

### Confirmation guardrail

Destructive write tools — `jira_create_issue`, `jira_create_child_issue`, and `jira_delete_issue` — require explicit user confirmation before they run. The server prompts the MCP client for confirmation and, if the client does not support elicitation, the tool fails closed (refuses to run) rather than executing without consent.

- Default is **on**. Set `JIRA_MCP_CONFIRM=off` (or `false`, `0`, `no`) to disable the guardrail.

### Basic Auth Mode

All three variables are required.

| Variable | Description | Example |
|---|---|---|
| `JIRA_URL` | Your Jira Cloud base URL | `https://yourcompany.atlassian.net` |
| `JIRA_USERNAME` | Email of the Jira user | `you@example.com` |
| `JIRA_API_TOKEN` | Jira API token | Create one [here](https://id.atlassian.com/manage-profile/security/api-tokens) |
| `ENABLED_TOOLS` | Optional comma-separated tool allowlist | `jira_search,jira_get_issue,jira_list_projects` |

## Client Setup

After installing, configure your MCP client. **OAuth users** only need `JIRA_URL` (run `jira-mcp auth` first). **Basic Auth users** need all three variables.

<details>
<summary><b>Claude Desktop</b> (<code>~/.config/claude/claude_desktop_config.json</code>)</summary>

**OAuth:**
```json
{
  "mcpServers": {
    "jira": {
      "command": "jira-mcp",
      "env": {
        "JIRA_URL": "https://yourcompany.atlassian.net"
      }
    }
  }
}
```

**Basic Auth:**
```json
{
  "mcpServers": {
    "jira": {
      "command": "jira-mcp",
      "env": {
        "JIRA_URL": "https://yourcompany.atlassian.net",
        "JIRA_USERNAME": "you@example.com",
        "JIRA_API_TOKEN": "your-api-token"
      }
    }
  }
}
```
</details>

<details>
<summary><b>Cursor</b> (<code>~/.cursor/mcp.json</code>)</summary>

**OAuth:**
```json
{
  "mcpServers": {
    "jira": {
      "command": "jira-mcp",
      "env": {
        "JIRA_URL": "https://yourcompany.atlassian.net"
      }
    }
  }
}
```

**Basic Auth:**
```json
{
  "mcpServers": {
    "jira": {
      "command": "jira-mcp",
      "env": {
        "JIRA_URL": "https://yourcompany.atlassian.net",
        "JIRA_USERNAME": "you@example.com",
        "JIRA_API_TOKEN": "your-api-token"
      }
    }
  }
}
```
</details>

<details>
<summary><b>OpenCode</b> (<code>~/.config/opencode/opencode.json</code>)</summary>

**OAuth:**
```json
{
  "mcp": {
    "jira": {
      "type": "local",
      "command": ["jira-mcp"],
      "environment": {
        "JIRA_URL": "https://yourcompany.atlassian.net"
      }
    }
  }
}
```

**Basic Auth:**
```json
{
  "mcp": {
    "jira": {
      "type": "local",
      "command": ["jira-mcp"],
      "environment": {
        "JIRA_URL": "https://yourcompany.atlassian.net",
        "JIRA_USERNAME": "you@example.com",
        "JIRA_API_TOKEN": "your-api-token"
      }
    }
  }
}
```
</details>

<details>
<summary><b>Windsurf</b> (<code>~/.config/windsurf/mcp_config.json</code>)</summary>

**OAuth:**
```json
{
  "mcpServers": {
    "jira": {
      "command": "jira-mcp",
      "env": {
        "JIRA_URL": "https://yourcompany.atlassian.net"
      }
    }
  }
}
```

**Basic Auth:**
```json
{
  "mcpServers": {
    "jira": {
      "command": "jira-mcp",
      "env": {
        "JIRA_URL": "https://yourcompany.atlassian.net",
        "JIRA_USERNAME": "you@example.com",
        "JIRA_API_TOKEN": "your-api-token"
      }
    }
  }
}
```
</details>

## Tools

| Tool | Description |
|---|---|
| `jira_search` | Search issues using JQL |
| `jira_get_issue` | Get an issue by key |
| `jira_create_issue` | Create a new issue |
| `jira_update_issue` | Update an existing issue |
| `jira_transition_issue` | Transition an issue |
| `jira_assign_issue` | Assign an issue to a user |
| `jira_delete_issue` | Delete an issue |
| `jira_list_projects` | List accessible projects |
| `jira_get_transitions` | List available transitions |
| `jira_add_comment` | Add a comment to an issue |
| `jira_add_worklog` | Log time against an issue |
| `jira_list_boards` | List Jira Agile boards, optionally filtered by project |
| `jira_list_sprints` | List sprints for a Jira Agile board |
| `jira_get_sprint` | Get a sprint by its Agile ID |
| `jira_get_active_sprint` | Get the active sprint for a board |
| `jira_search_sprint_by_name` | Search sprints by name on a board |
| `jira_get_issue_history` | Get the change history of an issue |
| `jira_list_project_versions` | List versions in a project |
| `jira_get_version` | Get a version by its ID |
| `jira_get_development_info` | Get linked PRs, branches, and commits |
| `jira_list_statuses` | List statuses for a project |
| `jira_create_issue_link` | Link two issues |
| `jira_get_related_issues` | Get issues linked to an issue |
| `jira_create_child_issue` | Create a sub-task or child issue |

### Description formatting and custom fields

The `jira_create_issue`, `jira_update_issue`, and `jira_create_child_issue` tools accept a `description` that is either a plain string or a pre-built ADF document object. The `jira_add_comment` tool accepts a `body` with the same contract.

A plain string is converted to Atlassian Document Format (ADF) with the following rules:

- Each line becomes a paragraph; blank lines are ignored.
- `#`, `##`, and `###` at the start of a line produce level-1/2/3 headings.
- `- item` or `* item` produce a bullet list (consecutive items are grouped).
- `- [ ] item` or `- [x] item` produce a checkbox list (`[x]` marks the item done).
- `**text**` renders bold inline text.

A pre-built ADF object (e.g. `{"type":"doc","version":1,"content":[...]}`) is embedded verbatim.

The optional `fields` object accepts custom Jira fields, for example:

- `parent`: `{"key":"PROJ-1"}`
- `labels`: `["tag"]`

Set the assignee with the separate `jira_assign_issue` tool, or via `fields` as `{"assignee":{"accountId":"..."}}`.

> **Note:** `jira_create_issue` returns only `{id, key, self}` (fields are `null`). Use `jira_get_issue` to verify the resulting `parent`, `labels`, and `assignee`.

## HTTP/SSE Transport

By default the binary speaks MCP over `stdio`. You can also run it as an HTTP/SSE server:

```bash
jira-mcp --transport http --port 8080
```

- `GET /sse` is the Server-Sent Events endpoint.
- `POST /message` is the client message endpoint.

> **Warning:** The HTTP/SSE transport carries no built-in authentication. Run it only on trusted networks and protect the endpoint with your own reverse proxy or VPN.

## Development

```bash
make test        # Run tests
make test-race   # Run tests with race detector
make test-cover  # Run tests with coverage
make build       # Build all packages
make fmt         # Format code
make vet         # Vet code
make clean       # Remove binary
```

## Releases

Built with [GoReleaser](https://goreleaser.com/). Tag and push to publish:

```bash
git tag -a v0.1.0 -m "Initial release"
git push origin v0.1.0
```

The Homebrew formula at `lucasvidela94/homebrew-tap` updates automatically.

## Security

- **In transit**: OAuth tokens and API tokens are sent only to your Jira Cloud instance in the HTTP `Authorization` header.
- **In logs**: Credentials are never logged, included in error messages, or returned in tool results.
- **On disk**: OAuth tokens are stored in `~/.config/jira-mcp/token.json` with `chmod 600` permissions. The MCP client stores Basic Auth tokens in its local config file with the same restricted permissions.
- **Token refresh**: OAuth tokens are automatically refreshed using the refresh token. If the refresh token expires (after 90 days of inactivity), run `jira-mcp auth` again.
- **Review**: The installer is open source. You can inspect it before running with `--inspect` or by downloading it to a file first.
- **Token creation**: Create or verify your Jira API token at https://id.atlassian.com/manage-profile/security/api-tokens.

## License

MIT
