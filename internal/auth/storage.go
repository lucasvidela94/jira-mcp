package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// TokenData represents a persisted OAuth token.
type TokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry"`
	CloudID      string    `json:"cloud_id,omitempty"`
	SiteURL      string    `json:"site_url,omitempty"`
}

// TokenStore persists token data to a JSON file with mutex-protected writes.
type TokenStore struct {
	path string
	mu   sync.Mutex
}

// NewTokenStore creates a TokenStore that reads/writes the given file path.
func NewTokenStore(path string) *TokenStore {
	return &TokenStore{path: path}
}

// Save writes token data to the file with 0600 permissions.
func (s *TokenStore) Save(t TokenData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}
	return nil
}

// Load reads token data from disk. Returns an error if the file is missing or corrupt.
func (s *TokenStore) Load() (TokenData, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return TokenData{}, fmt.Errorf("read token file %s: %w", s.path, err)
	}

	var t TokenData
	if err := json.Unmarshal(raw, &t); err != nil {
		return TokenData{}, fmt.Errorf("parse token file %s: %w", s.path, err)
	}
	return t, nil
}

// IsExpired returns true when the loaded token's expiry is in the past or zero.
func (s *TokenStore) IsExpired() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, err := s.loadUnsafe()
	if err != nil {
		return true
	}
	return t.Expiry.IsZero() || time.Now().After(t.Expiry)
}

// ExpiresAt returns the loaded token's expiry time, or zero time if no file exists.
func (s *TokenStore) ExpiresAt() (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, err := s.loadUnsafe()
	if err != nil {
		return time.Time{}, nil // no file = no expiry
	}
	return t.Expiry, nil
}

// HasRefreshToken returns true when the loaded token has a non-empty refresh_token.
func (s *TokenStore) HasRefreshToken() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, err := s.loadUnsafe()
	if err != nil {
		return false
	}
	return t.RefreshToken != ""
}

// loadUnsafe reads and parses the token file without locking.
// Caller must hold s.mu.
func (s *TokenStore) loadUnsafe() (TokenData, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return TokenData{}, err
	}
	var t TokenData
	if err := json.Unmarshal(raw, &t); err != nil {
		return TokenData{}, err
	}
	return t, nil
}
