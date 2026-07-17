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
- **Safe by design** — credentials are never logged, returned, or leaked
- **Multi-platform** — macOS, Linux, Windows, ARM64

## Quick Start

```bash
# Homebrew (macOS / Linux)
brew tap lucasvidela94/tap
brew install jira-mcp

# Or one-liner (macOS / Linux)
curl -fsSL https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.sh | bash

# Or Windows (PowerShell)
irm https://raw.githubusercontent.com/lucasvidela94/jira-mcp/master/scripts/install.ps1 | iex

# Or Go install
go install github.com/lucasvidela94/jira-mcp@latest
```

## Configuration

Set these environment variables:

| Variable | Description | Example |
|---|---|---|
| `JIRA_URL` | Your Jira Cloud base URL | `https://yourcompany.atlassian.net` |
| `JIRA_USERNAME` | Email of the Jira user | `you@example.com` |
| `JIRA_API_TOKEN` | Jira API token | Create one [here](https://id.atlassian.com/manage-profile/security/api-tokens) |

## Client Setup

<details>
<summary><b>Claude Desktop</b> (<code>~/.config/claude/claude_desktop_config.json</code>)</summary>

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

Credentials are only sent in the HTTP `Authorization` header. They are never logged, included in error messages, or returned in tool results.

## License

MIT
