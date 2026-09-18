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
- **OAuth 2.0 login** — browser-based authentication with automatic refresh, using your own Atlassian app credentials
- **Basic Auth** — works with any Jira Cloud site and no extra setup
- **Safe by design** — credentials are never logged, returned, or leaked
- **Multi-platform** — macOS, Linux, Windows, ARM64
- **Docker-ready** — minimal `scratch` image
- **HTTP/SSE transport** — run as a remote MCP server on a trusted network

## Authentication

jira-mcp supports two authentication methods. **Basic Auth is the default and
the only method that works with no extra setup.** OAuth 2.0 requires an
Atlassian OAuth app that you own.

### Basic Auth (Recommended default)

Traditional email + API token authentication via environment variables. Works on
any Jira Cloud site, needs no Atlassian admin approval, and is the simplest path.

Create or verify your Jira API token at: https://id.atlassian.com/manage-profile/security/api-tokens

```bash
export JIRA_URL=https://yourcompany.atlassian.net
export JIRA_USERNAME=you@example.com
export JIRA_API_TOKEN=your-api-token
```

### OAuth 2.0 (bring your own app)

Browser-based login with automatic token refresh, so no manual token creation or
rotation. It routes API calls through `api.atlassian.com/ex/jira/{cloudId}` and
stores tokens in `~/.config/jira-mcp/token.json` with `0600` permissions.

> **Important:** OAuth only works with an Atlassian OAuth 2.0 (3LO) app that is
> authorized for your site. Release binaries embed the publisher's own app, and
> Atlassian rejects it for any other site with *"We couldn't identify the app
> requesting access"*. To use OAuth you must register **your own** app.

1. Create an OAuth 2.0 (3LO) app at
   https://developer.atlassian.com/console/myapps/.
2. Add the callback URL `http://127.0.0.1:<port>/callback`. `jira-mcp` binds a
   random local port for each login; any port on `127.0.0.1` is accepted.
3. Add the scopes `read:jira-work`, `write:jira-work`, and `read:jira-user`.
4. Export your app credentials and log in:

```bash
export JIRA_OAUTH_CLIENT_ID=your-client-id
export JIRA_OAUTH_CLIENT_SECRET=your-client-secret

# The site URL is optional — it is only a hint. Pass it with --url or via
# JIRA_URL. Login works without it; Atlassian resolves the site for you.
jira-mcp auth --url https://yourcompany.atlassian.net

# Check token status
jira-mcp auth --status

# Logout and clear cached tokens
jira-mcp auth --logout
```

Credentials are resolved in this order: `JIRA_OAUTH_CLIENT_ID` /
`JIRA_OAUTH_CLIENT_SECRET`, then the values embedded in the binary at build
time. Set the variables whenever you want to use your own app.

> **Note:** If `JIRA_API_TOKEN` is present, Basic Auth takes precedence and OAuth
> is not used at all.

If the browser shows *"We couldn't identify the app requesting access"*, the app
is not authorized for the site: check the app's distribution settings, or fall
back to Basic Auth.

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

The installer can optionally configure your MCP client automatically. When run in
a terminal it asks which client you want to set up, then collects the Jira site
URL, your email, and an API token (Basic Auth). OAuth is not configured by the
installer: an OAuth app has to be created by you in the Atlassian developer
console and the browser login has to be run once afterwards.

```text
Which MCP client do you want to configure? [opencode/claude/cursor/windsurf/none] (default: opencode):
Jira URL (e.g. https://yourcompany.atlassian.net): https://yourcompany.atlassian.net
Jira username/email: you@example.com
Jira API token:
Will write jira-mcp configuration for opencode at /home/you/.config/opencode/opencode.json
Jira URL: https://yourcompany.atlassian.net
Username: you@example.com
API token: <hidden>
Proceed? [Y/n] y
```

Create or verify your Jira API token at: https://id.atlassian.com/manage-profile/security/api-tokens

Skip the wizard, or pass the answers as flags:

```bash
curl -fsSL https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.sh | bash -s -- --no-configure
```

```bash
curl -fsSL https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.sh | bash -s -- \
  --client opencode \
  --jira-url https://yourcompany.atlassian.net \
  --jira-username you@example.com \
  --jira-api-token your-api-token \
  --yes
```

Run `bash install.sh --help` for the full flag list (`--method`, `--dir`,
`--version`, `--sudo`, `--insecure`, `--client`, `--jira-url`,
`--jira-username`, `--jira-api-token`, `--configure`, `--no-configure`,
`--yes`, `--inspect`).

To use OAuth instead, see [OAuth 2.0 (bring your own app)](#oauth-20-bring-your-own-app)
and set the two `JIRA_OAUTH_*` variables in the generated config instead of the
Basic Auth ones.

### Docker

Build the image locally:

```bash
docker build -t jira-mcp .
```

The image is built **without** embedded OAuth credentials, so OAuth needs your
own app credentials at runtime:

```bash
docker run \
  -e JIRA_OAUTH_CLIENT_ID=your-client-id \
  -e JIRA_OAUTH_CLIENT_SECRET=your-client-secret \
  -v ~/.config/jira-mcp:/root/.config/jira-mcp \
  jira-mcp
```

> **Note:** Run `jira-mcp auth` on your host machine first to generate the token
> file. The callback listens on `127.0.0.1` inside the container, so complete the
> browser login on the host before starting or restarting the container.

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

The PowerShell installer supports the same optional auto-configuration when run
interactively (Basic Auth only). Save the script first to pass the values as
parameters:

```powershell
irm https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.ps1 -OutFile install.ps1
.\install.ps1 -Configure -Client opencode -JiraUrl https://yourcompany.atlassian.net -JiraUsername you@example.com -JiraApiToken your-api-token -Yes
```

To use OAuth instead, see [OAuth 2.0 (bring your own app)](#oauth-20-bring-your-own-app)
and set the two `JIRA_OAUTH_*` variables in the generated config.

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

### OAuth Mode

Requires your own Atlassian OAuth 2.0 (3LO) app; see
[OAuth 2.0 (bring your own app)](#oauth-20-bring-your-own-app).

| Variable | Description | Example |
|---|---|---|
| `JIRA_OAUTH_CLIENT_ID` | Client ID of **your** Atlassian OAuth app | `abc123` |
| `JIRA_OAUTH_CLIENT_SECRET` | Client secret of your app | `secret` |
| `JIRA_URL` | Optional site URL hint for `jira-mcp auth` | `https://yourcompany.atlassian.net` |
| `ENABLED_TOOLS` | Optional comma-separated tool allowlist | `jira_search,jira_get_issue,jira_list_projects` |
| `JIRA_MCP_CONFIRM` | Confirmation guardrail for destructive tools (`on` by default) | `off` |

The two `JIRA_OAUTH_*` variables are optional: when unset, release binaries fall
back to the credentials embedded at build time, which only work for the
publisher's own site.

### Confirmation guardrail

Destructive write tools — `jira_create_issue`, `jira_create_child_issue`, and `jira_delete_issue` — require explicit user confirmation before they run.

- When the MCP client supports **elicitation**, the server prompts the user interactively and executes only on accept.
- When the client does **not** support elicitation, the tools accept a `confirm` parameter: pass `confirm=true` to authorize the action. Without it, the server refuses to run (fails closed) rather than executing without consent.

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

After installing, configure your MCP client. **Basic Auth users** need
`JIRA_URL`, `JIRA_USERNAME`, and `JIRA_API_TOKEN`. **OAuth users** need their own
app credentials (`JIRA_OAUTH_CLIENT_ID`, `JIRA_OAUTH_CLIENT_SECRET`) and must run
`jira-mcp auth` once; `JIRA_URL` is optional there.

<details>
<summary><b>Claude Desktop</b> (<code>~/.config/claude/claude_desktop_config.json</code>)</summary>

**OAuth:**
```json
{
  "mcpServers": {
    "jira": {
      "command": "jira-mcp",
      "env": {
        "JIRA_OAUTH_CLIENT_ID": "your-client-id",
        "JIRA_OAUTH_CLIENT_SECRET": "your-client-secret",
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
        "JIRA_OAUTH_CLIENT_ID": "your-client-id",
        "JIRA_OAUTH_CLIENT_SECRET": "your-client-secret",
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
        "JIRA_OAUTH_CLIENT_ID": "your-client-id",
        "JIRA_OAUTH_CLIENT_SECRET": "your-client-secret",
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
        "JIRA_OAUTH_CLIENT_ID": "your-client-id",
        "JIRA_OAUTH_CLIENT_SECRET": "your-client-secret",
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

## Command-line usage

The same binary doubles as a standalone Jira CLI for terminal use. Any bare subcommand switches to CLI mode (or force it with `--cli`). Output is JSON; add `--pretty` to indent it.

```bash
jira-mcp list-projects
jira-mcp get-issue MC-1
jira-mcp search "project = MC ORDER BY created DESC"
jira-mcp get-issue-history MC-1
jira-mcp get-related-issues MC-1
jira-mcp list-statuses MC
jira-mcp list-project-versions MC
jira-mcp list-sprints 265
jira-mcp --pretty list-users
```

Run `jira-mcp help` for the full command list.

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
