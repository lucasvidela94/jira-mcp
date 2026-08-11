package auth

import (
	"encoding/base64"
	"net/http"
	"strings"
)

// BasicProvider implements AuthProvider using Atlassian API token (email + token).
// It produces a Basic auth header and returns the Jira site URL as the base URL.
type BasicProvider struct {
	siteURL string
	auth    string
}

// NewBasicProvider creates a BasicProvider from Jira site URL, email, and API token.
func NewBasicProvider(siteURL, username, apiToken string) *BasicProvider {
	raw := username + ":" + apiToken
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	return &BasicProvider{
		siteURL: strings.TrimRight(siteURL, "/"),
		auth:    "Basic " + encoded,
	}
}

func (p *BasicProvider) Authorize(req *http.Request) error {
	req.Header.Set("Authorization", p.auth)
	return nil
}

func (p *BasicProvider) BaseURL() string {
	return p.siteURL
}
