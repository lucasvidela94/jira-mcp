# Delta Spec: fix-oauth-login

## Overview

This change makes the Atlassian OAuth 2.0 login flow usable with credentials the operator controls, and makes its failure modes diagnosable. It modifies `internal/config` (two new environment variables), `internal/auth` (a timeout sentinel and clearer output), `main.go` (credential precedence, diagnostics, exit code), and `README.md` (documentation that currently describes behavior the code does not have).

RFC 2119 keywords (MUST, MUST NOT, SHOULD, MAY) are used deliberately below.

---

## Feature 1: Runtime OAuth credentials

### Requirement: OAuth credentials are resolved from the environment first

The system MUST read the OAuth 2.0 application credentials from the environment variables `JIRA_OAUTH_CLIENT_ID` and `JIRA_OAUTH_CLIENT_SECRET`, exposed as `Config.OAuthClientID` and `Config.OAuthClientSecret`.

When both environment variables are non-empty, they MUST take precedence over any build-time embedded value.

#### Scenario: Environment credentials are used

- GIVEN `JIRA_OAUTH_CLIENT_ID=env-id` and `JIRA_OAUTH_CLIENT_SECRET=env-secret` are set
- AND the binary was built with embedded credentials
- WHEN the OAuth configuration is resolved
- THEN the resolved `ClientID` MUST be `env-id`
- AND the resolved `ClientSecret` MUST be `env-secret`
- AND the reported source MUST be `environment`

#### Scenario: Only one environment variable is set

- GIVEN `JIRA_OAUTH_CLIENT_ID` is set and `JIRA_OAUTH_CLIENT_SECRET` is empty
- WHEN the OAuth configuration is resolved
- THEN the environment values MUST NOT be used as a pair
- AND resolution MUST fall back to the embedded pair
- AND the reported source MUST be `embedded`

#### Scenario: No environment credentials and no embedded credentials

- GIVEN neither environment variable is set
- AND the binary has no embedded credentials
- WHEN the OAuth configuration is resolved
- THEN resolution MUST report that OAuth is unconfigured
- AND no `oauth2.Config` MUST be produced

### Requirement: Build-time credentials remain supported

The existing ldflags variables `main.ClientID` and `main.ClientSecret` MUST remain valid inputs so that already-published release artifacts and the GoReleaser pipeline keep working unchanged.

#### Scenario: Embedded credentials still work

- GIVEN no OAuth environment variables are set
- AND the binary was built with `-X main.ClientID=... -X main.ClientSecret=...`
- WHEN the OAuth configuration is resolved
- THEN the embedded credentials MUST be used
- AND the reported source MUST be `embedded`

---

## Feature 2: `jira-mcp auth` diagnostics

### Requirement: The authorization URL is always shown

The `jira-mcp auth` flow MUST print the authorization URL before the user is expected to interact with it, so that a failed browser launch or a headless session is recoverable by copy/paste.

#### Scenario: Browser opens successfully

- GIVEN a configured OAuth client
- WHEN `jira-mcp auth` runs
- THEN the authorization URL MUST be printed
- AND a message MUST indicate that the CLI is waiting for the browser to complete authorization

### Requirement: Callback timeouts are identifiable and actionable

`CallbackServer.WaitForCode` MUST return an error that wraps the exported sentinel `auth.ErrCallbackTimeout` when the authorization callback never arrives.

`main.go` MUST detect that sentinel and return an error message that:

1. states that no authorization was received and the most likely cause is that the OAuth app is not authorized for the site, and
2. names the two ways forward: supplying your own app via `JIRA_OAUTH_CLIENT_ID`/`JIRA_OAUTH_CLIENT_SECRET`, or using Basic Auth with `JIRA_API_TOKEN`.

#### Scenario: Callback never arrives

- GIVEN a callback server started with a short timeout
- WHEN `WaitForCode` is called and nothing calls back
- THEN the returned error MUST satisfy `errors.Is(err, ErrCallbackTimeout)`

#### Scenario: CLI reports actionable guidance

- GIVEN the auth flow fails with an error wrapping `ErrCallbackTimeout`
- WHEN `runAuth` returns
- THEN the error text MUST mention `JIRA_OAUTH_CLIENT_ID`
- AND MUST mention `JIRA_API_TOKEN`

#### Scenario: Non-timeout failures stay unchanged

- GIVEN the auth flow fails with an error that does not wrap `ErrCallbackTimeout` (for example a denied consent)
- WHEN `runAuth` returns
- THEN the error MUST be reported as `login failed: <cause>`
- AND the timeout guidance MUST NOT be used

### Requirement: `jira-mcp auth` does not require any environment variable

The `auth` subcommand MUST NOT require `JIRA_URL` (or any other environment variable) to run the login flow, because the Atlassian Cloud ID is resolved from `GET https://api.atlassian.com/oauth/token/accessible-resources` and the site URL is not needed to authorize.

The site URL MUST be treated as an optional hint, resolved in this order:

1. `--url <site>` flag
2. `JIRA_URL` environment variable
3. no hint at all

Absence of a site URL MUST NOT be an error. The command MUST NOT prompt
interactively for it, because an optional value is not worth risking a hang in a
pipe or in CI.

#### Scenario: Login with no site URL anywhere

- GIVEN `JIRA_URL` is unset
- AND no `--url` flag is passed
- WHEN `jira-mcp auth` runs with resolvable OAuth credentials
- THEN the login flow MUST start
- AND the command MUST NOT fail because a site URL is missing
- AND the command MUST NOT block waiting for input

#### Scenario: Site URL provided as a flag

- GIVEN `JIRA_URL` is unset
- WHEN `jira-mcp auth --url https://magmalabs.atlassian.net` runs
- THEN the flow MUST use `https://magmalabs.atlassian.net` as the site hint
- AND MUST NOT require the environment variable

#### Scenario: Flag wins over the environment

- GIVEN `JIRA_URL=https://from-env.atlassian.net`
- WHEN `jira-mcp auth --url https://from-flag.atlassian.net` runs
- THEN the site hint MUST be `https://from-flag.atlassian.net`

#### Scenario: Malformed site URL

- GIVEN `--url magmalabs.atlassian.net` (no scheme)
- WHEN `jira-mcp auth` runs
- THEN the command MUST fail with a non-zero exit code
- AND the error MUST state that the URL must start with `http://` or `https://`

### Requirement: `auth` flags are parsed strictly

The `auth` subcommand MUST reject unknown flags with a non-zero exit code instead of silently ignoring them. `--url` MUST consume the following argument.

#### Scenario: Unknown flag

- GIVEN `jira-mcp auth --logout-everything`
- WHEN the command runs
- THEN it MUST fail with a non-zero exit code
- AND the error MUST name the offending flag

#### Scenario: Missing value for --url

- GIVEN `jira-mcp auth --url`
- WHEN the command runs
- THEN it MUST fail with a non-zero exit code
- AND the error MUST state that `--url` requires a value

### Requirement: The site hint and credential source are printed

When a site hint is resolved, the flow MUST print it. The flow MUST also print whether the OAuth app credentials came from the environment or were embedded. Secrets MUST NOT be printed.

#### Scenario: Reporting

- GIVEN a site hint and environment-provided credentials
- WHEN `jira-mcp auth` starts
- THEN the output MUST contain the site hint
- AND MUST contain the word `environment`
- AND MUST NOT contain the client secret value

### Requirement: Unconfigured OAuth is a failure

When `jira-mcp auth` is invoked without a request for `--status` or `--logout`, and no OAuth credentials can be resolved, the command MUST fail with a non-zero exit code and MUST explain that Basic Auth (`JIRA_API_TOKEN`) is an alternative.

#### Scenario: No credentials resolvable

- GIVEN no OAuth environment variables, no embedded credentials, and no `--status`/`--logout` flag
- WHEN `jira-mcp auth` runs
- THEN the command MUST return an error
- AND the error text MUST mention `JIRA_API_TOKEN`
- AND the exit code MUST be non-zero

---

## Feature 3: Documentation matches the code

### Requirement: README OAuth claims are accurate

`README.md` MUST state that OAuth requires an Atlassian OAuth 2.0 (3LO) application **owned by the operator**, whose redirect URI is `http://127.0.0.1:<port>/callback`, and MUST document `JIRA_OAUTH_CLIENT_ID` / `JIRA_OAUTH_CLIENT_SECRET`.

`README.md` MUST NOT present OAuth as the unconditionally recommended method for third-party users, and MUST NOT document installer flags that `scripts/install.sh` does not accept.

#### Scenario: Installer documentation is truthful

- GIVEN `scripts/install.sh --help`
- WHEN its documented flags are compared with the README installer section
- THEN every flag shown in the README MUST exist in the script
- AND the README MUST NOT advertise an `--auth-method` flag

#### Scenario: Docker OAuth is described correctly

- GIVEN the Dockerfile builds without embedded credentials
- WHEN a reader follows the README Docker section
- THEN the README MUST tell them to provide `JIRA_OAUTH_CLIENT_ID` and `JIRA_OAUTH_CLIENT_SECRET` at runtime for OAuth
- AND MUST NOT imply that mounting `~/.config/jira-mcp` alone is sufficient whenever the build lacks embedded credentials

---

## Feature 4: No credential leakage

### Requirement: Credentials are never printed

No log line, error message, or tool result introduced by this change may contain the value of `JIRA_API_TOKEN`, `JIRA_OAUTH_CLIENT_SECRET`, or an access/refresh token. The credential **source** (`environment` or `embedded`) and the client ID MAY be reported; the secret MUST NOT.

#### Scenario: Source is reported without the secret

- GIVEN `JIRA_OAUTH_CLIENT_SECRET=super-secret-value` is set
- WHEN the auth flow prints which credential source is used
- THEN the output MUST NOT contain `super-secret-value`
