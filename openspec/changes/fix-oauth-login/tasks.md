# Tasks: fix-oauth-login

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~320 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Delivery strategy | single-pr |
| Chain strategy | n/a |

Decision needed before apply: No
Chained PRs recommended: No
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | PR | Notes |
|------|------|-----|-------|
| 1 | Runtime credentials + auth UX + diagnostics | PR 1 | Code + tests, one reviewable change |
| 2 | Documentation truth-up | PR 1 | Same PR; docs only |

## Phase 1: Config — runtime OAuth credentials

- [x] 1.1 Write `TestLoad_OAuthClientCredentials` in `internal/config/config_test.go`: with `JIRA_OAUTH_CLIENT_ID`/`JIRA_OAUTH_CLIENT_SECRET` set via `t.Setenv`, `Load()` returns them in `Config.OAuthClientID`/`OAuthClientSecret`; assert the fields are empty when unset, that `Load()` still succeeds in OAuth mode without them, and that they are not required in Basic mode.
- [x] 1.2 Add `OAuthClientID` and `OAuthClientSecret` to `Config` in `internal/config/config.go` and populate them in `Load()`. Run `go test ./internal/config/...`.

## Phase 2: Auth — timeout sentinel and visible authorization URL

- [x] 2.1 Update `TestCallbackServer_Timeout` in `internal/auth/auth_test.go` to require `errors.Is(err, ErrCallbackTimeout)` while keeping the existing "timeout" message assertion.
- [x] 2.2 Add `ErrCallbackTimeout` to `internal/auth/callback.go`, store the configured timeout on `CallbackServer`, and wrap the sentinel in `WaitForCode`'s deadline branch. Run `go test ./internal/auth/...`.
- [x] 2.3 In `internal/auth/oauth.go` `Login`, always print the authorization URL and a waiting hint to stderr, replacing the current print-only-on-browser-failure behavior. Ensure neither the client secret nor any token is printed.

## Phase 3: Auth UX — no mandatory environment variables

- [x] 3.1 Write `main_test.go` tests for `parseAuthArgs`: `--status` and `--logout` set their flags; `--url <value>` is captured; an unknown flag returns an error naming the flag; a trailing `--url` without a value returns an error stating the flag requires a value.
- [x] 3.2 Write `main_test.go` tests for `resolveSiteURL`: a flag value wins over the environment; the environment is used when no flag is given; the result is empty and error-free when nothing is available; a `nil` getenv is tolerated; a value without an `http(s)` scheme returns an error.
- [x] 3.3 Implement `authOptions`, `parseAuthArgs`, and `resolveSiteURL` in `main.go`. Replace the current string-scanning loop in `runAuth` and delete the `JIRA_URL environment variable is required` failure path. Do NOT add an interactive prompt.

## Phase 4: main.go — precedence, diagnostics, exit code

- [x] 4.1 Write table tests in `main_test.go` for `resolveOAuthCredentials(embeddedID, embeddedSecret, getenv)`: env pair wins (source `environment`), partial env falls back to embedded (source `embedded`), no env + embedded (source `embedded`), nothing (source `""`).
- [x] 4.2 Write tests in `main_test.go` for `buildOAuthConfig(cfg)`: returns a config with env credentials when present, embedded credentials otherwise, and `nil` when unconfigured; assert the Atlassian endpoints and scopes are preserved.
- [x] 4.3 Write tests in `main_test.go` for the timeout guidance helper: the message mentions `JIRA_OAUTH_CLIENT_ID` and `JIRA_API_TOKEN`; assert the helper is not applied to non-timeout errors (cover the `errors.Is` branch used by `runAuth`).
- [x] 4.4 Implement `oauthCredentials` + `resolveOAuthCredentials` and change `buildOAuthConfig` to take `config.Config`; update both `main.go` call sites (`main` and the CLI dispatch) to pass the loaded config.
- [x] 4.5 Rewrite the `runAuth` default branch: load config, print the site hint (when any) and the credential source, fail with an error (exit code 1) when unconfigured mentioning `JIRA_API_TOKEN`, and map `auth.ErrCallbackTimeout` to the guidance helper while leaving other errors as `login failed: <cause>`. Run `go build ./...` and `go test ./...`.

## Phase 5: Documentation

- [x] 5.1 Rewrite the README `Authentication` section: OAuth requires an operator-owned Atlassian OAuth 2.0 (3LO) app; document `JIRA_OAUTH_CLIENT_ID` / `JIRA_OAUTH_CLIENT_SECRET`; describe Basic Auth as the default path; document `jira-mcp auth --url` and state that `JIRA_URL` is optional; add the `http://127.0.0.1:<port>/callback` redirect requirement.
- [x] 5.2 Fix the README installer sections: remove the `Authentication method [oauth/basic]`, `--auth-method`, and `--jira-api-token`-with-oauth examples; document only the flags present in `scripts/install.sh --help`; remove the OAuth installer flows for both macOS/Linux and Windows.
- [x] 5.3 Fix the README Docker section: state that the image builds without embedded credentials, so OAuth requires passing `JIRA_OAUTH_CLIENT_ID` and `JIRA_OAUTH_CLIENT_SECRET` at runtime, and that Basic Auth requires the three Basic variables.
- [x] 5.4 Add the new variables to the README environment-variable tables (OAuth mode and Basic mode).
- [x] 5.5 Add a short comment in `Dockerfile` next to `ENTRYPOINT` noting that OAuth credentials are supplied at runtime via environment variables.

## Phase 6: Verification

- [x] 6.1 Run `gofmt -l .` (must be empty), `go vet ./...`, `go build ./...`, and `go test ./...`.
- [x] 6.2 Manually verify credential precedence and secret hygiene: run `jira-mcp auth --status` and confirm no secret value appears in output.
- [x] 6.3 Confirm no release artifact changed: `.goreleaser.yaml`, `.github/workflows/release.yaml`, and `scripts/install.sh`/`install.ps1` are untouched (`git diff --stat`).
