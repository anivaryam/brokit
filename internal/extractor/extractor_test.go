package extractor

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	_, err := ExtractTarGz(dest, archive, "mybinary")
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dest, "mybinary"))
	require.NoError(t, err)
	assert.Equal(t, "binary content here", string(content))
}

func TestExtractTarGz_MissingBinary(t *testing.T) {
	dir := t.TempDir()
	archive := createTestTarGz(t, dir, map[string][]byte{
		"otherbinary": []byte("not the one"),
	})

	dest := t.TempDir()
	_, err := ExtractTarGz(dest, archive, "mybinary")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in archive")
}

func TestExtractTarGz_NestedPath(t *testing.T) {
	dir := t.TempDir()
	archive := createTestTarGz(t, dir, map[string][]byte{
		"subdir/mybinary": []byte("nested content"),
	})

	dest := t.TempDir()
	_, err := ExtractTarGz(dest, archive, "mybinary")
	require.NoError(t, err)
}

func TestExtractZip(t *testing.T) {
	dir := t.TempDir()
	archive := createTestZip(t, dir, map[string][]byte{
		"mybinary": []byte("zip binary content"),
	})

	dest := t.TempDir()
	_, err := ExtractZip(dest, archive, "mybinary")
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dest, "mybinary"))
	require.NoError(t, err)
	assert.Equal(t, "zip binary content", string(content))
}

func TestExtractZip_MissingBinary(t *testing.T) {
	dir := t.TempDir()
	archive := createTestZip(t, dir, map[string][]byte{
		"other": []byte("not it"),
	})

	dest := t.TempDir()
	_, err := ExtractZip(dest, archive, "mybinary")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in archive")
}
