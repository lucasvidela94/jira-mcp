# Delta Spec: workflow-friction

## Overview

This change makes the most common Jira lookup workflow — "find open issues assigned to me" — require fewer tool calls. It adds `jira_get_current_user` and extends `jira_search` with `open_only` and `fields` optional parameters.

---

## Feature 1: Current User

### Requirement: Get current user

The system MUST expose `jira_get_current_user`, which returns the identity of the authenticated Jira user.

| Parameter | Required | Type | Notes |
|-----------|----------|------|-------|
| (none) | — | — | Tool takes no arguments |

The handler MUST call the client method `GetCurrentUser(ctx)`. The client MUST perform `GET /rest/api/3/myself` and decode the response into a `CurrentUser` model. The handler MUST return the model as JSON with keys `accountId`, `displayName`, `emailAddress`, and `active`.

#### Scenario: Current user fetched successfully

- GIVEN an authenticated Jira session
- WHEN `jira_get_current_user` is called
- THEN the client requests `GET /rest/api/3/myself`
- AND the handler returns a JSON object containing `accountId`, `displayName`, and `emailAddress`

#### Scenario: Jira returns an authentication error

- GIVEN invalid credentials
- AND Jira responds with HTTP 401
- WHEN `jira_get_current_user` is called
- THEN the handler returns a tool error containing "authentication failed"

---

## Feature 2: Search Improvements

### Requirement: Search open issues

The system MUST extend `jira_search` with an optional boolean parameter `open_only`.

| Parameter | Required | Type | Notes |
|-----------|----------|------|-------|
| `jql` | yes | string | Base JQL query |
| `max_results` | no | number | Maximum number of issues |
| `open_only` | no | boolean | When true, append `AND resolution = Unresolved` to the JQL |
| `fields` | no | string[] | Custom fields to request in the search body |

When `open_only` is true, the handler MUST append ` AND resolution = Unresolved` to the provided JQL string after trimming trailing whitespace. The resulting JQL string MUST be passed to `client.Search`. When `open_only` is false or absent, the handler MUST use the JQL unchanged.

#### Scenario: Search open issues

- GIVEN `jql="assignee = currentUser()"` and `open_only=true`
- WHEN `jira_search` is called
- THEN the client receives `jql="assignee = currentUser() AND resolution = Unresolved"`
- AND the search is executed

#### Scenario: Search without open_only keeps JQL unchanged

- GIVEN `jql="project = PROJ"` and no `open_only`
- WHEN `jira_search` is called
- THEN the client receives `jql="project = PROJ"`

### Requirement: Search with custom fields

When `fields` is provided and non-empty, the client MUST send those field names in the search request body instead of the default field set. When `fields` is empty or absent, the client MUST continue to request the default fields (`id`, `key`, `summary`, `status`, `assignee`, `issuetype`).

The handler MUST pass `fields` to the client using a new `jira.WithFields` option.

#### Scenario: Request lighter fields

- GIVEN `jql="project = PROJ"` and `fields=["key", "summary", "status"]`
- WHEN `jira_search` is called
- THEN the POST body to `/rest/api/3/search/jql` contains `fields: ["key", "summary", "status"]`

#### Scenario: Default fields remain unchanged

- GIVEN `jql="project = PROJ"` with no `fields`
- WHEN `jira_search` is called
- THEN the POST body contains the default six fields

---

## Feature 3: Documentation

### Requirement: README reflects new capabilities

The README MUST list `jira_get_current_user` in the tool table. The README MUST document the new `open_only` and `fields` parameters for `jira_search` with at least one example.

---

## Non-Functional Requirements

- The system MUST continue to pass `go test ./...`.
- The system MUST continue to build with `go build ./...`.
- No existing tool contracts or environment variables may change.
- The total number of registered tools MUST be 25.
