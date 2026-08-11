// Package auth provides authentication providers for Jira Cloud.
// It supports Basic Auth (email + API token) and OAuth 2.0 (Atlassian 3LO).
package auth

import "net/http"

// AuthProvider supplies an Authorization header and the correct base URL
// for the Jira API. Basic Auth uses the site URL directly; OAuth routes
// through api.atlassian.com/ex/jira/{cloudid}/rest/...
type AuthProvider interface {
	// Authorize sets the Authorization header on the outgoing request.
	Authorize(req *http.Request) error

	// BaseURL returns the API base URL for the current auth mode.
	BaseURL() string
}
