# GitHub Repository Setup

After creating the repository on GitHub, follow these steps to enable releases.

## 1. Create the repository

Create `https://github.com/lucasvidela/jira-mcp` as a public repository and push this code:

```bash
git remote add origin https://github.com/lucasvidela/jira-mcp.git
git branch -M main
git push -u origin main
```

## 2. Enable workflows

Go to **Settings > Actions > General** and ensure "Allow all actions and reusable workflows" is selected.

## 3. Create a release

Tag a commit and push the tag. The release workflow will run GoReleaser automatically:

```bash
git tag -a v0.1.0 -m "Initial release"
git push origin v0.1.0
```

The workflow will:

- Run `go test -race ./...`
- Build binaries for darwin/linux/windows on amd64 and arm64
- Publish a GitHub release with archives and a checksum file

## 4. Install from release

Users can download binaries from the release page or install with:

```bash
go install github.com/lucasvidela/jira-mcp@latest
```
