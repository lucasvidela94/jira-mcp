package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

func TestSelfUpdater_assetName(t *testing.T) {
	u := NewSelfUpdater()
	got := u.assetName("v0.4.0")

	os := strings.ToLower(runtime.GOOS)
	ext := "tar.gz"
	if os == "windows" {
		ext = "zip"
	}
	want := fmt.Sprintf("jira-mcp_0.4.0_%s_%s.%s", os, runtime.GOARCH, ext)
	if got != want {
		t.Errorf("assetName() = %q, want %q", got, want)
	}
}

func TestSelfUpdater_checksumFromFile(t *testing.T) {
	u := NewSelfUpdater()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("abc123  file.tar.gz\nxyz789  other.tar.gz\n"))
	}))
	defer server.Close()

	got, err := u.checksumFromFile(context.Background(), server.URL, "file.tar.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "abc123" {
		t.Errorf("checksum = %q, want abc123", got)
	}

	_, err = u.checksumFromFile(context.Background(), server.URL, "missing.tar.gz")
	if err == nil {
		t.Fatal("expected error for missing checksum")
	}
}

func TestExtractTarGz(t *testing.T) {
	// Build a tar.gz containing a file named "jira-mcp".
	var archive bytes.Buffer
	gw := gzip.NewWriter(&archive)
	_ = gw.Close()

	_, err := extractTarGz(archive.Bytes())
	if err == nil {
		t.Fatal("expected error for empty tar.gz")
	}
}

func TestExtractTarGz_WithBinary(t *testing.T) {
	// Build a real tar.gz containing a file named "jira-mcp".
	var archive bytes.Buffer
	gw := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gw)

	content := []byte("fake-binary")
	hdr := &tar.Header{
		Name: "jira-mcp",
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := extractTarGz(archive.Bytes())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("extracted = %q, want %q", got, content)
	}
}

func TestSha256Sum(t *testing.T) {
	got := sha256Sum([]byte("hello"))
	h := sha256.New()
	h.Write([]byte("hello"))
	want := hex.EncodeToString(h.Sum(nil))
	if got != want {
		t.Errorf("sha256Sum() = %q, want %q", got, want)
	}
}
