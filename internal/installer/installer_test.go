package installer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── Archive extraction ──────────────────────────────────────────────────────

func createTestTarGz(t *testing.T, dir string, files map[string][]byte) string {
	t.Helper()
	path := filepath.Join(dir, "test.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create tar.gz: %v", err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
			Mode:     0755,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("write tar content: %v", err)
		}
	}
	tw.Close()
	gw.Close()
	f.Close()
	return path
}

func createTestZip(t *testing.T, dir string, files map[string][]byte) string {
	t.Helper()
	path := filepath.Join(dir, "test.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(f)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatalf("write zip content: %v", err)
		}
	}
	zw.Close()
	f.Close()
	return path
}

func TestExtractTarGz(t *testing.T) {
	dir := t.TempDir()
	archive := createTestTarGz(t, dir, map[string][]byte{
		"mybinary": []byte("binary content here"),
	})

	dest := t.TempDir()
	err := extractTarGz(archive, dest, "mybinary")
	if err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dest, "mybinary"))
	if err != nil {
		t.Fatalf("read extracted binary: %v", err)
	}
	if string(content) != "binary content here" {
		t.Errorf("content = %q, want %q", content, "binary content here")
	}
}

func TestExtractTarGz_MissingBinary(t *testing.T) {
	dir := t.TempDir()
	archive := createTestTarGz(t, dir, map[string][]byte{
		"otherbinary": []byte("not the one"),
	})

	dest := t.TempDir()
	err := extractTarGz(archive, dest, "mybinary")
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
	if !strings.Contains(err.Error(), "not found in archive") {
		t.Errorf("error = %q, want 'not found in archive'", err)
	}
}

func TestExtractTarGz_NestedPath(t *testing.T) {
	dir := t.TempDir()
	// Binary in a subdirectory — should still be found via filepath.Base
	archive := createTestTarGz(t, dir, map[string][]byte{
		"subdir/mybinary": []byte("nested content"),
	})

	dest := t.TempDir()
	err := extractTarGz(archive, dest, "mybinary")
	if err != nil {
		t.Fatalf("extractTarGz with nested path: %v", err)
	}
}

func TestExtractZip(t *testing.T) {
	dir := t.TempDir()
	archive := createTestZip(t, dir, map[string][]byte{
		"mybinary": []byte("zip binary content"),
	})

	dest := t.TempDir()
	err := extractZip(archive, dest, "mybinary")
	if err != nil {
		t.Fatalf("extractZip: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dest, "mybinary"))
	if err != nil {
		t.Fatalf("read extracted binary: %v", err)
	}
	if string(content) != "zip binary content" {
		t.Errorf("content = %q, want %q", content, "zip binary content")
	}
}

func TestExtractZip_MissingBinary(t *testing.T) {
	dir := t.TempDir()
	archive := createTestZip(t, dir, map[string][]byte{
		"other": []byte("not it"),
	})

	dest := t.TempDir()
	err := extractZip(archive, dest, "mybinary")
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
	if !strings.Contains(err.Error(), "not found in archive") {
		t.Errorf("error = %q, want 'not found in archive'", err)
	}
}

// ─── latestVersion ───────────────────────────────────────────────────────────

func TestLatestVersion_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(githubRelease{TagName: "v1.2.3"})
	}))
	defer ts.Close()

	orig := githubAPIBase
	githubAPIBase = ts.URL
	defer func() { githubAPIBase = orig }()

	ver, err := latestVersion("test/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "v1.2.3" {
		t.Errorf("version = %q, want %q", ver, "v1.2.3")
	}
}

func TestLatestVersion_RateLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "9999999999")
		w.WriteHeader(403)
	}))
	defer ts.Close()

	orig := githubAPIBase
	githubAPIBase = ts.URL
	defer func() { githubAPIBase = orig }()

	_, err := latestVersion("test/repo")
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("error = %q, want 'rate limit'", err)
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Errorf("error should suggest GITHUB_TOKEN, got: %q", err)
	}
}

func TestLatestVersion_EmptyTag(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(githubRelease{TagName: ""})
	}))
	defer ts.Close()

	orig := githubAPIBase
	githubAPIBase = ts.URL
	defer func() { githubAPIBase = orig }()

	_, err := latestVersion("test/repo")
	if err == nil {
		t.Fatal("expected error for empty tag")
	}
	if !strings.Contains(err.Error(), "no releases") {
		t.Errorf("error = %q, want 'no releases'", err)
	}
}

func TestLatestVersion_WithGitHubToken(t *testing.T) {
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(githubRelease{TagName: "v1.0.0"})
	}))
	defer ts.Close()

	orig := githubAPIBase
	githubAPIBase = ts.URL
	defer func() { githubAPIBase = orig }()

	t.Setenv("GITHUB_TOKEN", "test-token-123")

	_, err := latestVersion("test/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer test-token-123" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-token-123")
	}
}

// ─── Network errors ──────────────────────────────────────────────────────────

func TestWrapNetworkError_Nil(t *testing.T) {
	if err := wrapNetworkError(nil); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestWrapNetworkError_NonNetworkError(t *testing.T) {
	original := os.ErrNotExist
	wrapped := wrapNetworkError(original)
	if wrapped != original {
		t.Errorf("non-network error should pass through unchanged")
	}
}

// ─── formatBytes ─────────────────────────────────────────────────────────────

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{5242880, "5.0 MB"},
		{10485760, "10.0 MB"},
	}
	for _, tt := range tests {
		got := formatBytes(tt.bytes)
		if got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}
