# GitHub Repository Setup

After creating the repository on GitHub, follow these steps to enable releases.

## 1. Create the repository

Create `https://github.com/lucasvidela94/jira-mcp` as a public repository and push this code.

## 2. Create the Homebrew tap repository

Create `https://github.com/lucasvidela94/homebrew-tap` as a public repository. GoReleaser will commit the `jira-mcp` formula to `Formula/jira-mcp.rb` on every release.

## 3. Enable workflows

Go to **Settings > Actions > General** and ensure "Allow all actions and reusable workflows" is selected.

## 4. Create a release

Tag a commit and push the tag. The release workflow will run GoReleaser automatically:

```bash
git tag -a v0.1.0 -m "Initial release"
git push origin v0.1.0
```

The workflow will:

- Run `go test -race ./...`
- Build binaries for darwin/linux/windows on amd64 and arm64
- Publish a GitHub release with archives and a checksum file
- Update the Homebrew tap formula

## 5. Install from release

### Homebrew

```bash
brew tap lucasvidela94/tap
brew install jira-mcp
```

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/lucasvidela94/jira-mcp/main/scripts/install.sh | bash
```

### Go install

```bash
go install github.com/lucasvidela94/jira-mcp@latest
```

### Manual download

Download binaries from the [GitHub releases](https://github.com/lucasvidela94/jira-mcp/releases) page.
