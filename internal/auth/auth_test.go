package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// --- BasicProvider Tests ---

func TestBasicProvider_Authorize(t *testing.T) {
	provider := NewBasicProvider("https://example.atlassian.net", "user@example.com", "secret-token")

	req, err := http.NewRequest("GET", "https://example.atlassian.net/rest/api/3/project", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if err := provider.Authorize(req); err != nil {
		t.Fatalf("Authorize() error: %v", err)
	}

	auth := req.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		t.Errorf("expected Basic auth prefix, got %q", auth)
	}
	// base64("user@example.com:secret-token") = dXNlckBleGFtcGxlLmNvbTpzZWNyZXQtdG9rZW4=
	want := "Basic dXNlckBleGFtcGxlLmNvbTpzZWNyZXQtdG9rZW4="
	if auth != want {
		t.Errorf("expected auth header %q, got %q", want, auth)
	}
}

func TestBasicProvider_BaseURL(t *testing.T) {
	provider := NewBasicProvider("https://example.atlassian.net", "user@example.com", "token")
	if got := provider.BaseURL(); got != "https://example.atlassian.net" {
		t.Errorf("expected base URL https://example.atlassian.net, got %q", got)
	}
}

func TestBasicProvider_BaseURL_TrimsTrailingSlash(t *testing.T) {
	provider := NewBasicProvider("https://example.atlassian.net/", "user@example.com", "token")
	if got := provider.BaseURL(); got != "https://example.atlassian.net" {
		t.Errorf("expected trimmed base URL, got %q", got)
	}
}

func TestBasicProvider_ImplementsAuthProvider(t *testing.T) {
	var _ AuthProvider = NewBasicProvider("https://x", "u", "t")
}

// --- TokenStore Tests ---

func TestTokenStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	token := TokenData{
		AccessToken:  "access-abc",
		RefreshToken: "refresh-xyz",
		Expiry:       time.Now().Add(1 * time.Hour),
		CloudID:      "cloud-123",
		SiteURL:      "https://example.atlassian.net",
	}

	store := NewTokenStore(path)
	if err := store.Save(token); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Errorf("expected 0600 permissions, got %o", info.Mode().Perm())
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.AccessToken != token.AccessToken {
		t.Errorf("expected access token %q, got %q", token.AccessToken, loaded.AccessToken)
	}
	if loaded.RefreshToken != token.RefreshToken {
		t.Errorf("expected refresh token %q, got %q", token.RefreshToken, loaded.RefreshToken)
	}
	if loaded.CloudID != token.CloudID {
		t.Errorf("expected cloud ID %q, got %q", token.CloudID, loaded.CloudID)
	}
	if loaded.SiteURL != token.SiteURL {
		t.Errorf("expected site URL %q, got %q", token.SiteURL, loaded.SiteURL)
	}
	if !loaded.Expiry.Equal(token.Expiry) {
		t.Errorf("expected expiry %v, got %v", token.Expiry, loaded.Expiry)
	}
}

func TestTokenStore_Load_NoFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")

	store := NewTokenStore(path)
	_, err := store.Load()
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestTokenStore_Load_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	store := NewTokenStore(path)
	_, err := store.Load()
	if err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
}

func TestTokenStore_IsExpired(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	tests := []struct {
		name    string
		expiry  time.Time
		expired bool
	}{
		{"future", time.Now().Add(1 * time.Hour), false},
		{"past", time.Now().Add(-1 * time.Hour), true},
		{"zero", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := TokenData{
				AccessToken:  "at",
				RefreshToken: "rt",
				Expiry:       tt.expiry,
			}

			store := NewTokenStore(path)
			if err := store.Save(token); err != nil {
				t.Fatalf("Save() error: %v", err)
			}
			if got := store.IsExpired(); got != tt.expired {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expired)
			}
		})
	}
}

func TestTokenStore_ExpiresAt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	exp := time.Now().Add(30 * time.Minute)
	token := TokenData{
		AccessToken:  "at",
		RefreshToken: "rt",
		Expiry:       exp,
	}

	store := NewTokenStore(path)
	if err := store.Save(token); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	got, err := store.ExpiresAt()
	if err != nil {
		t.Fatalf("ExpiresAt() error: %v", err)
	}
	if !got.Equal(exp) {
		t.Errorf("ExpiresAt() = %v, want %v", got, exp)
	}
}

func TestTokenStore_ExpiresAt_NoFile_ReturnsZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")

	store := NewTokenStore(path)
	got, err := store.ExpiresAt()
	if err != nil {
		t.Fatalf("ExpiresAt() error: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("expected zero time for no-file, got %v", got)
	}
}

func TestTokenStore_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	store := NewTokenStore(path)

	const goroutines = 10
	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			token := TokenData{
				AccessToken:  "access-" + string(rune('0'+id)),
				RefreshToken: "refresh",
				Expiry:       time.Now().Add(time.Duration(id) * time.Hour),
			}
			_ = store.Save(token)
			done <- true
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	// The file should still be valid JSON and loadable
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() after concurrent writes error: %v", err)
	}
	if loaded.AccessToken == "" {
		t.Error("expected non-empty access token after concurrent writes")
	}
	// The file content is serialized in valid JSON
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	var check TokenData
	if err := json.Unmarshal(raw, &check); err != nil {
		t.Errorf("file content is not valid JSON after concurrent writes: %s", raw)
	}
}

func TestTokenStore_HasRefreshToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	t.Run("with refresh token", func(t *testing.T) {
		token := TokenData{
			AccessToken:  "at",
			RefreshToken: "rt",
			Expiry:       time.Now().Add(1 * time.Hour),
		}
		store := NewTokenStore(path)
		if err := store.Save(token); err != nil {
			t.Fatalf("Save() error: %v", err)
		}
		if !store.HasRefreshToken() {
			t.Error("expected HasRefreshToken() = true when refresh token exists")
		}
	})

	t.Run("without refresh token", func(t *testing.T) {
		emptyPath := filepath.Join(dir, "empty.json")
		token := TokenData{
			AccessToken: "at",
			Expiry:      time.Now().Add(1 * time.Hour),
		}
		store := NewTokenStore(emptyPath)
		if err := store.Save(token); err != nil {
			t.Fatalf("Save() error: %v", err)
		}
		if store.HasRefreshToken() {
			t.Error("expected HasRefreshToken() = false when refresh token is empty")
		}
	})
}

func TestTokenStore_PersistsFileOnSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	store := NewTokenStore(path)
	token := TokenData{
		AccessToken:  "at-1",
		RefreshToken: "rt-1",
		Expiry:       time.Now().Add(1 * time.Hour),
	}
	if err := store.Save(token); err != nil {
		t.Fatalf("first Save() error: %v", err)
	}

	// Create a new store instance pointing to the same file
	store2 := NewTokenStore(path)
	loaded, err := store2.Load()
	if err != nil {
		t.Fatalf("second Load() error: %v", err)
	}
	if loaded.AccessToken != "at-1" {
		t.Errorf("expected access token %q from second store, got %q", "at-1", loaded.AccessToken)
	}
}

// --- OAuthProvider Tests ---

// fakeTokenSource implements oauth2.TokenSource for testing.
type fakeTokenSource struct {
	accessToken  string
	refreshToken string
	expiry       time.Time
	err          error
}

func (f *fakeTokenSource) Token() (*oauth2.Token, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &oauth2.Token{
		AccessToken:  f.accessToken,
		RefreshToken: f.refreshToken,
		Expiry:       f.expiry,
	}, nil
}

func TestOAuthProvider_Authorize(t *testing.T) {
	dir := t.TempDir()
	store := NewTokenStore(filepath.Join(dir, "token.json"))
	provider := NewOAuthProvider("cloud-xyz", "https://example.atlassian.net", store, nil)

	provider.SetTokenSource(&fakeTokenSource{
		accessToken: "bearer-token-abc",
		expiry:      time.Now().Add(1 * time.Hour),
	})

	req, err := http.NewRequest("GET", "https://api.atlassian.com/ex/jira/cloud-xyz/rest/api/3/project", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if err := provider.Authorize(req); err != nil {
		t.Fatalf("Authorize() error: %v", err)
	}

	auth := req.Header.Get("Authorization")
	if auth != "Bearer bearer-token-abc" {
		t.Errorf("expected Bearer auth header, got %q", auth)
	}
}

func TestOAuthProvider_Authorize_NoTokenSource(t *testing.T) {
	dir := t.TempDir()
	store := NewTokenStore(filepath.Join(dir, "token.json"))
	provider := NewOAuthProvider("cloud-xyz", "https://example.atlassian.net", store, nil)

	req, _ := http.NewRequest("GET", "https://api.atlassian.com/ex/jira/cloud-xyz/rest/api/3/project", nil)
	err := provider.Authorize(req)
	if err == nil {
		t.Fatal("expected error when token source is nil")
	}
}

func TestOAuthProvider_BaseURL(t *testing.T) {
	dir := t.TempDir()
	store := NewTokenStore(filepath.Join(dir, "token.json"))
	provider := NewOAuthProvider("cloud-xyz", "https://example.atlassian.net", store, nil)

	want := "https://api.atlassian.com/ex/jira/cloud-xyz"
	if got := provider.BaseURL(); got != want {
		t.Errorf("expected base URL %q, got %q", want, got)
	}
}

func TestOAuthProvider_ImplementsAuthProvider(t *testing.T) {
	dir := t.TempDir()
	store := NewTokenStore(filepath.Join(dir, "token.json"))
	var _ AuthProvider = NewOAuthProvider("cloud-xyz", "https://example.atlassian.net", store, nil)
}

// --- CallbackServer Tests ---

func TestCallbackServer_StartsOnRandomPort(t *testing.T) {
	srv := NewCallbackServer()
	// Create a context with timeout to prevent test hanging
	srv.Start(time.Second)
	defer srv.Stop()

	port := srv.Port()
	if port == 0 {
		t.Error("expected non-zero port after Start()")
	}
	if port < 0 || port > 65535 {
		t.Errorf("port %d out of valid range", port)
	}
}

func TestCallbackServer_ReceivesCode(t *testing.T) {
	srv := NewCallbackServer()
	srv.Start(5 * time.Second)
	defer srv.Stop()

	port := srv.Port()
	url := fmt.Sprintf("http://127.0.0.1:%d/callback?code=test-auth-code", port)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	code, err := srv.WaitForCode()
	if err != nil {
		t.Fatalf("WaitForCode() error: %v", err)
	}
	if code != "test-auth-code" {
		t.Errorf("expected auth code 'test-auth-code', got %q", code)
	}
}

func TestCallbackServer_ReceivesErrorFromCallback(t *testing.T) {
	srv := NewCallbackServer()
	srv.Start(5 * time.Second)
	defer srv.Stop()

	port := srv.Port()
	url := fmt.Sprintf("http://127.0.0.1:%d/callback?error=access_denied&error_description=User+denied", port)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	defer resp.Body.Close()

	_, err = srv.WaitForCode()
	if err == nil {
		t.Fatal("expected error for denied consent")
	}
	if !strings.Contains(err.Error(), "access_denied") {
		t.Errorf("expected error to mention access_denied, got %v", err)
	}
}

func TestCallbackServer_Timeout(t *testing.T) {
	srv := NewCallbackServer()
	srv.Start(50 * time.Millisecond)
	defer srv.Stop()

	_, err := srv.WaitForCode()
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout error, got %v", err)
	}
}

func TestCallbackServer_BrowserOpenInjected(t *testing.T) {
	called := false
	srv := NewCallbackServer()
	srv.BrowserOpen = func(url string) error {
		called = true
		if url != "https://example.com/auth" {
			t.Errorf("expected url https://example.com/auth, got %q", url)
		}
		return nil
	}

	err := srv.OpenBrowser("https://example.com/auth")
	if err != nil {
		t.Fatalf("OpenBrowser error: %v", err)
	}
	if !called {
		t.Error("expected BrowserOpen to be called")
	}
}

func TestOAuthProvider_RefreshPersistWrapper_SavesToken(t *testing.T) {
	dir := t.TempDir()
	store := NewTokenStore(filepath.Join(dir, "token.json"))

	// Create a token source that returns a token
	baseTS := &fakeTokenSource{
		accessToken:  "fresh-access",
		refreshToken: "fresh-refresh",
		expiry:       time.Now().Add(1 * time.Hour),
	}

	wrapper := &refreshPersistWrapper{
		wrapped: baseTS,
		storage: store,
		cloudID: "cloud-xyz",
		siteURL: "https://api.atlassian.com/ex/jira/cloud-xyz",
	}

	token, err := wrapper.Token()
	if err != nil {
		t.Fatalf("Token() error: %v", err)
	}
	if token.AccessToken != "fresh-access" {
		t.Errorf("expected fresh-access, got %s", token.AccessToken)
	}

	// Wait for async persist
	time.Sleep(50 * time.Millisecond)

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.AccessToken != "fresh-access" {
		t.Errorf("expected persisted access token fresh-access, got %q", loaded.AccessToken)
	}
	if loaded.CloudID != "cloud-xyz" {
		t.Errorf("expected persisted cloud_id cloud-xyz, got %q", loaded.CloudID)
	}
}
