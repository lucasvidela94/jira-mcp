package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/lucasvidela94/jira-mcp/internal/auth"
)

// AuthStatus writes the current OAuth token status to the writer.
// Returns an error if no valid token exists.
func AuthStatus(w io.Writer, store *auth.TokenStore) error {
	if store.IsExpired() {
		return fmt.Errorf("no valid token: token is expired or not found")
	}

	token, err := store.Load()
	if err != nil {
		return fmt.Errorf("cannot read token: %w", err)
	}

	status := "valid"
	if store.IsExpired() {
		status = "expired"
	}

	fmt.Fprintf(w, "Status: %s\n", status)
	fmt.Fprintf(w, "Cloud ID: %s\n", token.CloudID)
	fmt.Fprintf(w, "Site URL: %s\n", token.SiteURL)
	if token.AccessToken != "" {
		masked := token.AccessToken[:min(8, len(token.AccessToken))] + "..."
		fmt.Fprintf(w, "Access Token: %s\n", masked)
	}
	if store.HasRefreshToken() {
		fmt.Fprintln(w, "Refresh Token: present")
	} else {
		fmt.Fprintln(w, "Refresh Token: absent")
	}

	return nil
}

// AuthLogout removes the stored token file.
func AuthLogout(w io.Writer, tokenPath string) error {
	// Check if the file exists
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		fmt.Fprintln(w, "No stored token to remove.")
		return nil
	}

	if err := os.Remove(tokenPath); err != nil {
		return fmt.Errorf("remove token file: %w", err)
	}
	fmt.Fprintln(w, "Token removed. Run 'jira-mcp auth' to authenticate again.")
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
