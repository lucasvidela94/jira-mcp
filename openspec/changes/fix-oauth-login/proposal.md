# Proposal: fix-oauth-login

## Intent

`jira-mcp auth` (the Atlassian OAuth 2.0 flow) is **broken for every user who is not the app owner**, and its documentation claims a level of support that the code and installers do not provide. Concretely:

1. OAuth client credentials (`ClientID`/`ClientSecret`) are only injectable at **build time** via ldflags (`.goreleaser.yaml`). Every released binary therefore embeds the maintainer's Atlassian app. An Atlassian 3LO app that is not distributed/authorized for a given site is rejected by Atlassian with *"We couldn't identify the app requesting access"*, so OAuth cannot work for a third-party site (e.g. `magmalabs.atlassian.net`).
2. The Dockerfile builds **without** the ldflags, so `ClientID == ""` and `jira-mcp auth` prints *"OAuth credentials not embedded"* — OAuth is impossible in Docker, yet the README documents a Docker OAuth workflow.
3. `jira-mcp auth` is a **silent black box**: it prints one line, opens a browser, and then hangs for 5 minutes. When Atlassian rejects the app on its own page, no callback ever arrives, the user gets a generic `callback timeout`, and there is no hint about the cause or the fallback.
4. `runAuth` returns `nil` (exit code 0) when OAuth credentials are missing, so scripted/CI usage cannot detect the failure.
5. `README.md` documents an installer wizard (`Authentication method [oauth/basic]`, `--auth-method oauth`) that **does not exist** in `scripts/install.sh` / `scripts/install.ps1`; both scripts only implement Basic Auth.
6. **UX anti-pattern**: `runAuth` hard-requires the `JIRA_URL` environment variable (`return fmt.Errorf("JIRA_URL environment variable is required for OAuth login")`), forcing the user to export a variable before they can log in — yet the OAuth flow **never needs it**: the Cloud ID is resolved from `https://api.atlassian.com/oauth/token/accessible-resources`, and `auth.Detect` ignores the site URL entirely in OAuth mode. Unknown flags are also silently ignored, and no `--url` flag exists.

The goal is to make OAuth **usable with credentials the user controls** (without recompiling), **self-sufficient** (no mandatory environment variables to log in), **diagnosable**, and to make the documentation **true**.

## Scope

### In Scope
- Runtime resolution of OAuth client credentials from `JIRA_OAUTH_CLIENT_ID` / `JIRA_OAUTH_CLIENT_SECRET`, falling back to build-time embedded values.
- `jira-mcp auth --url <site>` plus interactive fallback, so login no longer requires exporting `JIRA_URL`; the flow MUST work with no site URL at all.
- Strict flag parsing for the `auth` subcommand (unknown flags are an error, not silently ignored).
- Actionable diagnostics for the `auth` flow (authorization URL always printed, waiting hint, timeout mapped to guidance).
- Non-zero exit code when OAuth is requested but not configured.
- Correcting `README.md` OAuth/installer/Docker documentation, including a "bring your own Atlassian OAuth app" section.
- Unit tests for every behavior above.

### Out of Scope
- Registering an Atlassian OAuth app on the user's behalf, or shipping a distributed app.
- Implementing an OAuth wizard inside `scripts/install.sh` / `install.ps1` (they stay Basic Auth only; the README is corrected to match).
- Reading other applications' MCP configuration files to discover `JIRA_URL`.
- Changing `jira-mcp update`, the Homebrew formula, or the GoReleaser pipeline.
- Browser-side detection of Atlassian's error page (not observable from the CLI).

## Capabilities

### New Capabilities
- `oauth-runtime-credentials`: OAuth app credentials can be supplied at runtime via environment variables.
- `oauth-diagnostics`: the `auth` flow reports progress, the authorization URL, and an actionable message when authorization never completes.
- `oauth-self-sufficient-login`: `jira-mcp auth` works without any mandatory environment variable, accepting an optional site URL via `--url` or a prompt.

### Modified Capabilities
- `configuration`: `Config` gains `OAuthClientID` and `OAuthClientSecret`.
- `jira-mcp`: README OAuth, Docker, and installer sections corrected.

## Approach

Keep the existing package seams (`internal/config`, `internal/auth`, `main.go`).

1. `internal/config` parses the two new environment variables into `Config`, so all environment handling stays in one place.
2. `main.go` gains a pure `resolveOAuthCredentials(cfg)` helper with a documented precedence: **environment > build-time ldflags**. `buildOAuthConfig` is switched to use it, which makes OAuth work in Docker and for any user who registers their own app.
3. `internal/auth` gains an exported `ErrCallbackTimeout` sentinel; `CallbackServer.WaitForCode` wraps it. `main.go` maps that sentinel to human guidance, keeping user-facing text out of the auth package.
4. `runAuth` returns a real error (exit code 1) when OAuth is unconfigured, always prints the authorization URL plus a "waiting" hint, parses its flags strictly, and treats the site URL as **optional** (`--url` flag > `JIRA_URL` > interactive prompt > absent), because the flow resolves the Cloud ID itself.
5. Documentation is corrected to match the code that actually exists.

TDD is strict: each behavior gets its test first, then the minimal implementation.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/config/config.go` | Modified | Parse `JIRA_OAUTH_CLIENT_ID` / `JIRA_OAUTH_CLIENT_SECRET` |
| `internal/config/config_test.go` | Modified | Tests for the new fields |
| `internal/auth/callback.go` | Modified | `ErrCallbackTimeout` sentinel, timeout-aware error |
| `internal/auth/oauth.go` | Modified | Always print authorization URL and waiting hint |
| `internal/auth/auth_test.go` | Modified | Tests for the timeout sentinel |
| `main.go` | Modified | `resolveOAuthCredentials`, `buildOAuthConfig(cfg)`, `runAuth` flags/site-URL/diagnostics and exit code |
| `main_test.go` | Modified | Tests for credential precedence, flag parsing, site-URL resolution, and timeout guidance |
| `README.md` | Modified | OAuth, Docker, and installer sections corrected |
| `Dockerfile` | Modified | Document that OAuth credentials come from the runtime environment |

**Unchanged:** `.goreleaser.yaml`, `.github/workflows/release.yaml`, Homebrew formula, `scripts/install.sh`, `scripts/install.ps1`.

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| A user's own Atlassian app still needs the `http://127.0.0.1:<port>/callback` redirect registered | High | Document it explicitly in the README bring-your-own-app section |
| Env vars silently override embedded credentials and confuse debugging | Low | Print which source was used (`environment` / `embedded`) in the `auth` flow |
| Changing `buildOAuthConfig` signature breaks a caller | Low | Only called from `main.go`; verified by `go build ./...` |
| README rewrite drifts from the scripts again | Med | Document only flags present in `scripts/install.sh --help` |

## Rollback Plan

Revert the single merged PR (or the branch). The change is additive and touches no release artifact:

- Removing the two environment variables restores the previous build-time-only behavior.
- The `ErrCallbackTimeout` sentinel is internal; reverting restores the generic timeout message.
- No data migration, no config file format change, no token file change.

## Dependencies

- `golang.org/x/oauth2` (already in `go.mod`)
- An Atlassian OAuth 2.0 (3LO) app owned by the operator, for the bring-your-own-app path

## Success Criteria

- [ ] With `JIRA_OAUTH_CLIENT_ID`/`JIRA_OAUTH_CLIENT_SECRET` set, `jira-mcp auth` uses them and never the embedded values
- [ ] With the variables unset, behavior is identical to today (embedded values)
- [ ] With neither present, `jira-mcp auth` exits non-zero and explains both fallbacks
- [ ] `jira-mcp auth` succeeds with **no** `JIRA_URL` set and no `--url` flag
- [ ] `jira-mcp auth --url https://x.atlassian.net` is accepted; an unknown flag is rejected with a non-zero exit code
- [ ] A callback timeout produces a message naming the most likely cause and both ways forward
- [ ] `go test ./...` and `go build ./...` pass
- [ ] README contains no claim that installers or Docker support OAuth without credentials

## Size Estimate & Chained PRs

Estimated ~200 changed lines (mostly README and tests). This is a single focused fix; **chained PRs are not required**.

**Decision needed before apply: No.**
