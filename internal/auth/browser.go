package auth

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openBrowser opens the given URL in the default browser for the current platform.
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default: // linux and others
		cmd = "xdg-open"
		args = []string{url}
	}

	c := exec.Command(cmd, args...)
	if err := c.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	// Don't wait for the browser to close — release the process
	_ = c.Process.Release()
	return nil
}
