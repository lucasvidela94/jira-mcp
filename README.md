# jira-mcp

A Model Context Protocol (MCP) server that exposes safe read/write access to Jira Cloud through stdio.

## Requirements

- Jira Cloud instance
- Jira API token ([create one here](https://id.atlassian.com/manage-profile/security/api-tokens))

## Configuration

The server reads all configuration from environment variables:

| Variable | Description | Example |
|---|---|---|
| `JIRA_URL` | Your Jira Cloud base URL | `https://yourcompany.atlassian.net` |
| `JIRA_USERNAME` | Email of the Jira user | `you@example.com` |
| `JIRA_API_TOKEN` | Jira API token | `your-api-token` |

## Installation

### macOS / Linux

```bash
# Homebrew (recommended)
brew tap lucasvidela94/tap
brew install jira-mcp

# Or install script
curl -fsSL https://raw.githubusercontent.com/lucasvidela94/jira-mcp/main/scripts/install.sh | bash
```

### Windows

Download the latest binary from the [GitHub releases](https://github.com/lucasvidela94/jira-mcp/releases) page, or use `go install`.

### Go install (any platform)

```bash
go install github.com/lucasvidela94/jira-mcp@latest
```

## Client Configuration

### Claude Desktop

`~/.config/claude/claude_desktop_config.json`:

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

### Cursor

`~/.cursor/mcp.json`:

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

### OpenCode

`~/.config/opencode/opencode.json`:

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

### Windsurf

`~/.config/windsurf/mcp_config.json`:

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

## Available Tools

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
| `jira_add_comment` | Add a comment |
| `jira_add_worklog` | Add a worklog |

## Development

```bash
# Run tests
make test

# Run tests with race detector
make test-race

# Build binary
make build

# Format and vet
make fmt
make vet
```

## Release

Releases are built with [GoReleaser](https://goreleaser.com/) and published to GitHub releases. The Homebrew formula is updated automatically.

```bash
git tag -a v0.1.0 -m "Initial release"
git push origin v0.1.0
```

## Security

Credentials are never logged, returned in tool results, or included in error messages. The server only sends the API token in the HTTP `Authorization` header.

## License

MIT
