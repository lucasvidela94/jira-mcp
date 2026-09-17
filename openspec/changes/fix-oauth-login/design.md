# Design: fix-oauth-login

## Technical Approach

Three coordinated changes, each behind an existing package seam:

1. **`internal/config`** parses `JIRA_OAUTH_CLIENT_ID` / `JIRA_OAUTH_CLIENT_SECRET` into new `Config` fields. All environment handling continues to live in one place; `main.go` never reads `os.Getenv` for new variables.
2. **`main.go`** resolves the credential pair with an explicit precedence (environment ⊃ ldflags) in a pure, testable helper, and uses it wherever an `*oauth2.Config` is built. This is the fix that makes OAuth viable for non-owners and for Docker.
3. **`internal/auth`** exposes `ErrCallbackTimeout`; `main.go` translates that sentinel into user-facing guidance. The auth package stays free of CLI copy.

The ldflags path is preserved verbatim so that published artifacts, Homebrew, and the GoReleaser pipeline do not change.

## Architecture Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|--------------|-----------|
| Where environment variables are read | `internal/config.Config` | `os.Getenv` directly in `main.go` | The repo already centralizes every environment variable in `internal/config`; the design rules require preserving that seam. Also makes precedence unit-testable without process-global state where possible. |
| Credential precedence | environment wins over embedded | embedded wins; merge per-field | An operator who exports a variable expects it to take effect. Merging per-field would silently pair an environment ID with an embedded secret and produce a confusing `invalid_client` error, so the pair is taken atomically. |
| Behavior when only one variable is set | ignore the environment pair entirely, fall back to embedded | error out; use the lone value with the embedded counterpart | Using a lone value guarantees a broken pair. Falling back keeps existing users working and is the least surprising. An explicit error is a possible follow-up if this proves confusing. |
| Timeout signalling | exported sentinel `ErrCallbackTimeout` wrapped by `WaitForCode` | custom error type with fields; string matching | `errors.Is` is idiomatic, keeps `main.go` free of brittle string matching, and does not change `WaitForCode`'s signature. |
| Where user-facing guidance lives | `main.go` (`runAuth`) | `internal/auth` | The auth package is a library used by both the MCP server and the CLI; CLI copy does not belong in it. |
| How the authorization URL is surfaced | printed by `Login` (which owns the URL) | `Login` returns the URL for `main.go` to print | Avoids widening the `Login` contract and keeps the print next to the browser launch. Only the URL is printed — never the code, tokens, or secret. |
| Site URL requirement | optional hint (`--url` > `JIRA_URL` > absent) | keep `JIRA_URL` mandatory | The OAuth flow resolves the Cloud ID from `accessible-resources` and `auth.Detect` ignores the URL in OAuth mode, so demanding an exported variable is unnecessary friction. |
| `auth` flag parsing | strict `--flag value` loop returning an error on unknowns | keep the existing "scan for exact strings" loop | Silently ignoring `--urll` or `--url=...` makes failures invisible; strict parsing surfaces typos. |
| Interactive site-URL prompt | not offered | prompt when stdin is a terminal | An optional value does not justify a prompt that can hang in a pipe or CI. Detecting a terminal also needs an extra dependency (`golang.org/x/term`) or an unreliable `ModeCharDevice` check, which reports `/dev/null` as a terminal. |
| Installer OAuth wizard | Do not add one; fix the README | Implement the wizard in `install.sh`/`install.ps1` | A wizard that writes OAuth credentials cannot finish the flow (the user must still run `jira-mcp auth` in a browser) and duplicates logic across two shells. Correcting the documentation is the honest, smaller change. |

## Sequence: successful authorization (with operator-owned app)

```text
user            jira-mcp auth          browser            Atlassian
 |                    |                   |                   |
 |  export JIRA_OAUTH_CLIENT_ID/SECRET    |                   |
 |-------------------->|                   |                   |
 |                    | resolveOAuthCredentials()              |
 |                    |  -> env (source=environment)           |
 |                    |                   |                   |
 |                    | Start callback on 127.0.0.1:0          |
 |                    | print auth URL     |                   |
 |                    |------------------>|  GET /authorize?...|
 |                    |                   |------------------>|
 |                    |                   |  (consent screen)  |
 |                    |                   |<------------------|
 |                    |<------------------|  GET /callback?code|
 |                    | Exchange code      |                   |
 |                    |--------------------------------------->|
 |                    |<---------------------------------------|
 |                    | GET accessible-resources               |
 |                    |--------------------------------------->|
 |                    |<---------------------------------------|
 |                    | save token.json (0600)                 |
 |<-------------------|                   |                   |
 |  exit 0            |                   |                   |
```

## Sequence: rejected app (the bug being fixed)

```text
user            jira-mcp auth          browser            Atlassian
 |                    |                   |                   |
 |  (no own app; binary embeds author's)  |                   |
 |-------------------->|                   |                   |
 |                    | print auth URL     |                   |
 |                    |------------------>|  GET /authorize?...|
 |                    |                   |------------------>|
 |                    |                   |  "We couldn't identify the app"
 |                    |                   |  (no redirect back)|
 |                    |  ... waits 5 min ...                   |
 |                    | timeout -> error wrapping              |
 |                    |            ErrCallbackTimeout          |
 |<-------------------|                   |                   |
 |  exit 1 + guidance: bring your own app (JIRA_OAUTH_CLIENT_ID / JIRA_OAUTH_CLIENT_SECRET)
 |            or Basic Auth (JIRA_API_TOKEN)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/config/config.go` | Modify | Add `OAuthClientID`, `OAuthClientSecret` fields; read `JIRA_OAUTH_CLIENT_ID`, `JIRA_OAUTH_CLIENT_SECRET`. |
| `internal/config/config_test.go` | Modify | Add `TestLoad_OAuthClientCredentials` and a test that the fields are optional. |
| `internal/auth/callback.go` | Modify | Add `timeout` field, exported `ErrCallbackTimeout`, wrap it in `WaitForCode`. |
| `internal/auth/oauth.go` | Modify | Print the authorization URL and a waiting hint in `Login`. |
| `internal/auth/auth_test.go` | Modify | Assert `TestCallbackServer_Timeout` satisfies `errors.Is(err, ErrCallbackTimeout)`. |
| `main.go` | Modify | Add `resolveOAuthCredentials`, `parseAuthArgs`, `resolveSiteURL`; change `buildOAuthConfig` to accept `config.Config`; rewrite the `runAuth` default branch. |
| `main_test.go` | Modify | Tests for precedence, partial env, none configured, flag parsing, site-URL resolution, and timeout guidance. |
| `README.md` | Modify | Correct Authentication, Installation, and Docker sections; add bring-your-own-app. |
| `Dockerfile` | Modify | Comment documenting runtime OAuth credentials. |

## Interfaces / Contracts

### Config

```go
type Config struct {
    URL               string   // JIRA_URL
    Username          string   // JIRA_USERNAME
    APIToken          string   // JIRA_API_TOKEN
    AuthMode          string   // "basic" when JIRA_API_TOKEN is set, "oauth" otherwise
    EnabledTools      []string // ENABLED_TOOLS (comma-separated allowlist)
    ConfirmWrite      bool     // JIRA_MCP_CONFIRM
    OAuthClientID     string   // JIRA_OAUTH_CLIENT_ID
    OAuthClientSecret string   // JIRA_OAUTH_CLIENT_SECRET
}

func Load() (Config, error)
```

`Load` MUST NOT require either OAuth variable; both are optional in every mode.

### Credential resolution (`main.go`)

```go
// oauthCredentials is the resolved OAuth app credential pair plus its provenance.
type oauthCredentials struct {
    clientID     string
    clientSecret string
    source       string // "environment", "embedded", or "" when unconfigured
}

// resolveOAuthCredentials applies the precedence: environment over embedded.
// The environment pair is taken atomically, so a partial pair falls back to
// the embedded pair instead of being sent to Atlassian as a broken combination.
// envID/envSecret come from Config (already read from the environment by
// internal/config); embeddedID/embeddedSecret are the ldflags variables.
func resolveOAuthCredentials(envID, envSecret, embeddedID, embeddedSecret string) oauthCredentials

// buildOAuthConfig returns nil when no credential pair can be resolved.
func buildOAuthConfig(cfg config.Config) *oauth2.Config

// oauthConfigFrom builds the Atlassian endpoint/scopes for a resolved pair.
func oauthConfigFrom(creds oauthCredentials) *oauth2.Config
```

Credential values are passed in explicitly rather than read from globals so the precedence is unit-testable without mutating the process environment, and the single env-reading seam stays in `internal/config`.

### Auth sentinel (`internal/auth`)

```go
// ErrCallbackTimeout indicates the OAuth callback never arrived before the deadline.
var ErrCallbackTimeout = errors.New("timed out waiting for the OAuth callback")

func (s *CallbackServer) WaitForCode() (string, error) // wraps ErrCallbackTimeout on deadline
```

### Auth argument parsing and site-URL resolution (`main.go`)

```go
// authOptions is the parsed form of `jira-mcp auth`.
type authOptions struct {
    status  bool
    logout  bool
    siteURL string // from --url; empty when not given
}

// parseAuthArgs parses the auth subcommand flags strictly.
// Unknown flags and a missing --url value are errors.
func parseAuthArgs(args []string) (authOptions, error)

// resolveSiteURL resolves the optional site hint, in precedence order:
// the --url flag, then JIRA_URL, then "". Never prompts.
// A non-empty result that lacks an http(s) scheme is an error.
func resolveSiteURL(flagValue string, getenv func(string) string) (string, error)
```

`siteURL` is a **hint only**: `Login` derives the Cloud ID from `accessible-resources`, and `auth.Detect` ignores the URL in OAuth mode. A `""` result is therefore valid and MUST NOT block login.

### CLI guidance (`main.go`)

`runAuth` MUST map `errors.Is(err, auth.ErrCallbackTimeout)` to a message containing:

- the fact that no authorization was received,
- the most likely cause (the OAuth app is not authorized for the site),
- `JIRA_OAUTH_CLIENT_ID` / `JIRA_OAUTH_CLIENT_SECRET` (own app), and
- `JIRA_API_TOKEN` (Basic Auth).

Any other error MUST keep the existing `login failed: <cause>` shape.

## Test Strategy

| Layer | What to Test | Approach |
|-------|--------------|----------|
| Unit | Config parsing of the two new variables, including absence | `internal/config/config_test.go` with `t.Setenv` |
| Unit | Credential precedence: env wins, partial env ignored, none configured, embedded only | `main_test.go` table test over `resolveOAuthCredentials` with explicit arguments |
| Unit | `buildOAuthConfig` returns `nil` only when unconfigured | `main_test.go` |
| Unit | Timeout wraps `ErrCallbackTimeout` | `internal/auth/auth_test.go` |
| Unit | `parseAuthArgs`: `--status`, `--logout`, `--url <v>`, unknown flag, missing `--url` value | `main_test.go` |
| Unit | `resolveSiteURL`: flag wins, env fallback, `""` when unset, scheme validation, `nil` getenv tolerated | `main_test.go` with a fake `getenv` |
| Unit | Timeout guidance mentions both fallbacks and is not used for other errors | `main_test.go` |
| Integration | Build and full suite | `go build ./...`, `go test ./...` |
| Manual | Real Atlassian app | Not automated; documented in README |

Strict TDD: tests are written before the implementation and must fail first. The timeout guidance is tested through a small extracted helper (e.g. `authTimeoutHelp(siteURL string) string`) so no browser or network is involved.

## Migration / Rollout

No migration and no artifact change.

- Existing users with embedded credentials: unchanged behavior (`source=embedded`).
- Users who previously exported `JIRA_URL` only to satisfy `auth`: the variable becomes optional, and `--url` replaces it. No breaking change.
- Existing Basic Auth users: untouched; `JIRA_API_TOKEN` still short-circuits OAuth entirely in `auth.Detect`.
- Docker users: can now enable OAuth by exporting the two variables into the container.
- The `README.md` installer section loses examples that never worked; users following the previous OAuth text were already failing.

## Open Questions

1. Should a partial environment pair (only `JIRA_OAUTH_CLIENT_ID` set) be a hard error instead of a silent fallback? Deferred; silent fallback preserves current behavior.
2. Should `jira-mcp auth` fail fast when the resolved client ID equals the well-known embedded maintainer app and the target `JIRA_URL` is not the maintainer's site? Attractive, but it would require shipping a list of "known-good" sites; deferred.
3. Should `scripts/install.sh` gain an `--auth-method` flag that only accepts `basic` (and errors on `oauth` with a pointer to the docs)? Possible follow-up, out of scope here.
