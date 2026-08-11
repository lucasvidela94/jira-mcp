package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lucasvidela94/jira-mcp/internal/auth"
)

func TestAuthStatus_HasValidToken(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.json")
	store := auth.NewTokenStore(tokenPath)

	token := auth.TokenData{
		AccessToken:  "at",
		RefreshToken: "rt",
		Expiry:       time.Now().Add(1 * time.Hour),
		CloudID:      "cloud-xyz",
		SiteURL:      "https://example.atlassian.net",
	}
	if err := store.Save(token); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	var buf bytes.Buffer
	err := AuthStatus(&buf, store)
	if err != nil {
		t.Fatalf("AuthStatus() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "cloud-xyz") {
		t.Errorf("expected output to contain cloud_id cloud-xyz, got: %s", out)
	}
}

func TestAuthStatus_NoToken(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.json")
	store := auth.NewTokenStore(tokenPath)

	var buf bytes.Buffer
	err := AuthStatus(&buf, store)
	if err == nil {
		t.Fatal("expected error for no token")
	}
}

func TestAuthStatus_ExpiredToken(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.json")
	store := auth.NewTokenStore(tokenPath)

	token := auth.TokenData{
		AccessToken: "at",
		Expiry:      time.Now().Add(-1 * time.Hour),
	}
	if err := store.Save(token); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	var buf bytes.Buffer
	err := AuthStatus(&buf, store)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestAuthLogout_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.json")
	store := auth.NewTokenStore(tokenPath)

	token := auth.TokenData{
		AccessToken:  "at",
		RefreshToken: "rt",
		Expiry:       time.Now().Add(1 * time.Hour),
	}
	if err := store.Save(token); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	var buf bytes.Buffer
	err := AuthLogout(&buf, tokenPath)
	if err != nil {
		t.Fatalf("AuthLogout() error: %v", err)
	}

	// Verify token file is removed
	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Error("expected token file to be removed after logout")
	}
}

func TestAuthLogout_NoFile(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "nonexistent.json")

	var buf bytes.Buffer
	err := AuthLogout(&buf, tokenPath)
	if err != nil {
		t.Fatalf("AuthLogout() with no file should not error: %v", err)
	}
}
