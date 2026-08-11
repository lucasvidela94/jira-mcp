package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// OAuthProvider implements AuthProvider using Atlassian OAuth 2.0 (3LO).
// It uses an oauth2.TokenSource for Bearer tokens and routes API calls
// through api.atlassian.com/ex/jira/{cloudid}/rest/...
type OAuthProvider struct {
	cloudID     string
	baseURL     string
	tokenSource oauth2.TokenSource
	storage     *TokenStore
	mu          sync.Mutex
}

// NewOAuthProvider creates an OAuthProvider for a given Cloud instance.
func NewOAuthProvider(cloudID, siteURL string, storage *TokenStore, tokenSource oauth2.TokenSource) *OAuthProvider {
	_ = siteURL
	return &OAuthProvider{
		cloudID:     cloudID,
		baseURL:     fmt.Sprintf("https://api.atlassian.com/ex/jira/%s", cloudID),
		tokenSource: tokenSource,
		storage:     storage,
	}
}

// SetTokenSource replaces the current token source.
func (p *OAuthProvider) SetTokenSource(ts oauth2.TokenSource) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tokenSource = ts
}

// TokenSource returns the current token source.
func (p *OAuthProvider) TokenSource() oauth2.TokenSource {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.tokenSource
}

// Authorize sets the Authorization header to the current Bearer token.
func (p *OAuthProvider) Authorize(req *http.Request) error {
	p.mu.Lock()
	ts := p.tokenSource
	p.mu.Unlock()

	if ts == nil {
		return fmt.Errorf("oauth: no token source configured")
	}
	token, err := ts.Token()
	if err != nil {
		return fmt.Errorf("oauth: get token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	return nil
}

// BaseURL returns the Atlassian API gateway URL for this Cloud instance.
func (p *OAuthProvider) BaseURL() string {
	return p.baseURL
}

// CloudID returns the Atlassian Cloud instance ID.
func (p *OAuthProvider) CloudID() string {
	return p.cloudID
}

// refreshPersistWrapper wraps an oauth2.TokenSource to persist tokens to storage on refresh.
type refreshPersistWrapper struct {
	wrapped oauth2.TokenSource
	storage *TokenStore
	cloudID string
	siteURL string
	mu      sync.Mutex
}

func (w *refreshPersistWrapper) Token() (*oauth2.Token, error) {
	token, err := w.wrapped.Token()
	if err != nil {
		return nil, err
	}
	// Persist asynchronously to avoid blocking API calls
	go w.persist(token)
	return token, nil
}

func (w *refreshPersistWrapper) persist(token *oauth2.Token) {
	w.mu.Lock()
	defer w.mu.Unlock()

	td := TokenData{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		CloudID:      w.cloudID,
		SiteURL:      w.siteURL,
	}
	_ = w.storage.Save(td)
}

// Login runs the browser-based OAuth 2.0 authorization code flow.
// After exchanging the authorization code for tokens, it fetches the
// Atlassian Cloud ID from the accessible-resources endpoint and stores
// it alongside the token.
func (p *OAuthProvider) Login(ctx context.Context, config *oauth2.Config) error {
	srv := NewCallbackServer()
	srv.Start(5 * time.Minute)
	defer srv.Stop()

	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/callback", srv.Port())
	config.RedirectURL = redirectURL

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	if err := srv.BrowserOpen(authURL); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser: %v\nVisit this URL to authorize:\n%s\n", err, authURL)
	}

	code, err := srv.WaitForCode()
	if err != nil {
		return fmt.Errorf("oauth login: %w", err)
	}

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("oauth token exchange: %w", err)
	}

	// Fetch the Cloud ID from Atlassian's accessible-resources endpoint.
	cloudID, err := fetchCloudID(ctx, config, token)
	if err != nil {
		return fmt.Errorf("fetch cloud ID: %w", err)
	}

	// Update provider with the resolved Cloud ID.
	p.mu.Lock()
	p.cloudID = cloudID
	p.baseURL = fmt.Sprintf("https://api.atlassian.com/ex/jira/%s", cloudID)
	p.mu.Unlock()

	ts := config.TokenSource(ctx, token)

	// Wrap to persist on refresh
	persistWrapper := &refreshPersistWrapper{
		wrapped: ts,
		storage: p.storage,
		cloudID: cloudID,
		siteURL: p.baseURL,
	}
	p.SetTokenSource(persistWrapper)

	// Persist initial token with Cloud ID.
	td := TokenData{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		CloudID:      cloudID,
		SiteURL:      p.baseURL,
	}
	_ = p.storage.Save(td)

	return nil
}

// fetchCloudID retrieves the Atlassian Cloud ID using the OAuth access token.
func fetchCloudID(ctx context.Context, config *oauth2.Config, token *oauth2.Token) (string, error) {
	client := config.Client(ctx, token)
	resp, err := client.Get("https://api.atlassian.com/oauth/token/accessible-resources")
	if err != nil {
		return "", fmt.Errorf("request accessible resources: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("accessible resources returned status %d", resp.StatusCode)
	}

	var resources []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return "", fmt.Errorf("decode accessible resources: %w", err)
	}

	if len(resources) == 0 {
		return "", fmt.Errorf("no accessible resources found")
	}

	return resources[0].ID, nil
}
