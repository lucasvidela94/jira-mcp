package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

// DefaultTokenPath returns the default path for the OAuth token file.
func DefaultTokenPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "token.json")
	}
	return filepath.Join(home, ".config", "jira-mcp", "token.json")
}

// Detect selects the appropriate AuthProvider based on configuration and cached state.
// Basic mode (JIRA_API_TOKEN set): returns BasicProvider.
// OAuth mode (no API token): loads from token store if valid, else errors.
func Detect(url, username, apiToken string, oauthConfig *oauth2.Config) (AuthProvider, error) {
	if apiToken != "" {
		return NewBasicProvider(url, username, apiToken), nil
	}

	// OAuth mode: check for cached token
	store := NewTokenStore(DefaultTokenPath())

	token, err := store.Load()
	if err != nil {
		return nil, fmt.Errorf("no credentials found: set JIRA_API_TOKEN or run 'jira-mcp auth'")
	}

	if token.CloudID == "" {
		return nil, fmt.Errorf("invalid cached token: missing cloud_id")
	}

	provider := NewOAuthProvider(token.CloudID, token.SiteURL, store, nil)

	// Try to set up a token source. If an oauthConfig is provided AND we have a refresh token,
	// set up a full OAuth token source with refresh support.
	if oauthConfig != nil && token.RefreshToken != "" {
		ts := oauthConfig.TokenSource(nil, &oauth2.Token{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			Expiry:       token.Expiry,
		})
		// Wrap to persist on refresh
		siteURL := token.SiteURL
		if siteURL == "" {
			siteURL = fmt.Sprintf("https://api.atlassian.com/ex/jira/%s", token.CloudID)
		}
		persistWrapper := &refreshPersistWrapper{
			wrapped: ts,
			storage: store,
			cloudID: token.CloudID,
			siteURL: siteURL,
		}
		provider.SetTokenSource(persistWrapper)
	} else if token.AccessToken != "" && !store.IsExpired() {
		// Static token source for valid unexpired tokens
		ts := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token.AccessToken,
			Expiry:      token.Expiry,
		})
		provider.SetTokenSource(ts)
	} else {
		return nil, fmt.Errorf("no valid credentials: cached token is expired and no refresh token available; run 'jira-mcp auth'")
	}

	return provider, nil
}
