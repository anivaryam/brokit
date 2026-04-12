package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_MissingFile(t *testing.T) {
	s, err := Load("/nonexistent/path/state.json")
	require.NoError(t, err, "expected no error for missing file")
	assert.NotNil(t, s.Installed, "Installed map should be non-nil")
	assert.Equal(t, 0, len(s.Installed), "expected empty Installed")
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	data := `{"installed":{"tunnel":{"name":"tunnel","version":"v1.0.0"}}}`
	os.WriteFile(path, []byte(data), 0644)

	s, err := Load(path)
	require.NoError(t, err, "unexpected error")
	tool, ok := s.Get("tunnel")
	assert.True(t, ok, "expected tunnel to be installed")
	assert.Equal(t, "v1.0.0", tool.Version)
}

func TestLoad_CorruptJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	os.WriteFile(path, []byte("{not valid json"), 0644)

	_, err := Load(path)
	require.Error(t, err, "expected error for corrupt JSON")
}

func TestLoad_NullInstalled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	os.WriteFile(path, []byte(`{"installed":null}`), 0644)

	s, err := Load(path)
	require.NoError(t, err, "unexpected error")
	assert.NotNil(t, s.Installed, "Installed map should be non-nil even when JSON has null")
}

func TestSave_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "state.json")

	s := &State{Installed: make(map[string]InstalledTool)}
	s.Set("tunnel", "v1.0.0")

	require.NoError(t, s.Save(path), "Save failed")

	// Verify file exists and is valid JSON
	data, err := os.ReadFile(path)
	require.NoError(t, err, "cannot read saved file")

	var loaded State
	require.NoError(t, json.Unmarshal(data, &loaded), "saved file is not valid JSON")
	assert.Equal(t, "v1.0.0", loaded.Installed["tunnel"].Version)
}

func TestSave_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Write initial state
	s := &State{Installed: make(map[string]InstalledTool)}
	s.Set("tunnel", "v1.0.0")
	require.NoError(t, s.Save(path), "first Save failed")

	// Overwrite with new state
	s.Set("merge-port", "v2.0.0")
	require.NoError(t, s.Save(path), "second Save failed")

	// Verify no temp files left behind
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "state.json" {
			t.Errorf("unexpected file left behind: %s", e.Name())
		}
	}

	// Verify content
	loaded, err := Load(path)
	require.NoError(t, err, "Load failed")
	_, ok := loaded.Get("tunnel")
	assert.True(t, ok, "tunnel should be in state")
	_, ok = loaded.Get("merge-port")
	assert.True(t, ok, "merge-port should be state")
}

func TestSetGetRemove(t *testing.T) {
	s := &State{Installed: make(map[string]InstalledTool)}

	// Set
	s.Set("tunnel", "v1.0.0")
	tool, ok := s.Get("tunnel")
	assert.True(t, ok, "expected tunnel after Set")
	assert.Equal(t, "v1.0.0", tool.Version)

	// Remove
	s.Remove("tunnel")
	_, ok = s.Get("tunnel")
	assert.False(t, ok, "expected tunnel to be gone after Remove")
}
