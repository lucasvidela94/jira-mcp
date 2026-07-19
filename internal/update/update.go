// Package update implements a self-update command for the jira-mcp binary.
// It downloads the latest release asset from GitHub, verifies its checksum, and
// replaces the running binary.
package update

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const repo = "lucasvidela94/jira-mcp"

// SelfUpdater updates the current binary from GitHub releases.
type SelfUpdater struct {
	HTTPClient *http.Client
	Repo       string
}

// NewSelfUpdater creates a SelfUpdater with sensible defaults.
func NewSelfUpdater() *SelfUpdater {
	return &SelfUpdater{
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
		Repo:       repo,
	}
}

// Result reports the outcome of an update.
type Result struct {
	PreviousVersion string
	NewVersion      string
	Message         string
}

// Run performs the self-update.
func (u *SelfUpdater) Run(ctx context.Context, currentVersion string) (*Result, error) {
	currentPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("cannot locate current executable: %w", err)
	}
	currentPath, err = filepath.EvalSymlinks(currentPath)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve executable path: %w", err)
	}

	latest, err := u.latestVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot determine latest version: %w", err)
	}

	if strings.EqualFold(latest, currentVersion) {
		return &Result{
			PreviousVersion: currentVersion,
			NewVersion:      latest,
			Message:         fmt.Sprintf("jira-mcp is already up to date (%s)", currentVersion),
		}, nil
	}

	assetName := u.assetName(latest)
	binaryBytes, err := u.downloadAndVerify(ctx, latest, assetName)
	if err != nil {
		return nil, err
	}

	if err := u.apply(currentPath, binaryBytes); err != nil {
		return nil, fmt.Errorf("cannot replace binary: %w", err)
	}

	return &Result{
		PreviousVersion: currentVersion,
		NewVersion:      latest,
		Message:         fmt.Sprintf("updated jira-mcp from %s to %s", currentVersion, latest),
	}, nil
}

func (u *SelfUpdater) latestVersion(ctx context.Context) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := u.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode release: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("no tag_name in release")
	}
	return release.TagName, nil
}

func (u *SelfUpdater) assetName(version string) string {
	version = strings.TrimPrefix(version, "v")
	os := strings.ToLower(runtime.GOOS)
	arch := runtime.GOARCH
	ext := "tar.gz"
	if os == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("jira-mcp_%s_%s_%s.%s", version, os, arch, ext)
}

func (u *SelfUpdater) downloadAndVerify(ctx context.Context, version, assetName string) ([]byte, error) {
	baseURL := fmt.Sprintf("https://github.com/%s/releases/download/%s", u.Repo, version)
	assetURL := baseURL + "/" + assetName
	checksumURL := baseURL + "/checksums.txt"

	archiveBytes, err := u.download(ctx, assetURL)
	if err != nil {
		return nil, fmt.Errorf("download asset: %w", err)
	}

	expected, err := u.checksumFromFile(ctx, checksumURL, assetName)
	if err != nil {
		return nil, fmt.Errorf("fetch checksum: %w", err)
	}

	actual := sha256Sum(archiveBytes)
	if !strings.EqualFold(actual, expected) {
		return nil, fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}

	binaryBytes, err := extractBinary(archiveBytes, runtime.GOOS)
	if err != nil {
		return nil, fmt.Errorf("extract binary: %w", err)
	}
	return binaryBytes, nil
}

func (u *SelfUpdater) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := u.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (u *SelfUpdater) checksumFromFile(ctx context.Context, checksumURL, assetName string) (string, error) {
	data, err := u.download(ctx, checksumURL)
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[1] == assetName {
			return fields[0], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("checksum for %s not found", assetName)
}

func (u *SelfUpdater) apply(currentPath string, binaryBytes []byte) error {
	dir := filepath.Dir(currentPath)
	base := filepath.Base(currentPath)

	tmpFile := filepath.Join(dir, base+".new")
	if err := os.WriteFile(tmpFile, binaryBytes, 0755); err != nil {
		return fmt.Errorf("write new binary: %w", err)
	}

	backupFile := currentPath + ".bak"
	_ = os.Remove(backupFile)

	if err := os.Rename(currentPath, backupFile); err != nil {
		_ = os.Remove(tmpFile)
		if runtime.GOOS == "windows" {
			return u.scheduleWindowsReplace(currentPath, tmpFile)
		}
		return fmt.Errorf("rename current binary: %w", err)
	}

	if err := os.Rename(tmpFile, currentPath); err != nil {
		_ = os.Rename(backupFile, currentPath)
		_ = os.Remove(tmpFile)
		return fmt.Errorf("rename new binary: %w", err)
	}

	_ = os.Remove(backupFile)
	return nil
}

func (u *SelfUpdater) scheduleWindowsReplace(currentPath, newPath string) error {
	script := currentPath + ".update.ps1"
	content := fmt.Sprintf(`Start-Sleep -Seconds 1
Remove-Item -Path %q -Force -ErrorAction SilentlyContinue
Move-Item -Path %q -Destination %q -Force
Remove-Item -Path %q -Force -ErrorAction SilentlyContinue
`, currentPath, newPath, currentPath, script)

	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		return fmt.Errorf("write windows updater script: %w", err)
	}

	return fmt.Errorf("cannot replace running binary on Windows; run %s after exiting this process", script)
}

func extractBinary(archiveBytes []byte, goos string) ([]byte, error) {
	if goos == "windows" {
		return extractZip(archiveBytes)
	}
	return extractTarGz(archiveBytes)
}

func extractTarGz(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag == tar.TypeReg && h.Name == "jira-mcp" {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("jira-mcp binary not found in archive")
}

func extractZip(data []byte) ([]byte, error) {
	r := bytes.NewReader(data)
	zr, err := zip.NewReader(r, int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if f.Name == "jira-mcp.exe" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("jira-mcp.exe not found in archive")
}

func sha256Sum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
