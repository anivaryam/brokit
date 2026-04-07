package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFile(t *testing.T) {
	s, err := Load("/nonexistent/path/state.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if s.Installed == nil {
		t.Fatal("Installed map should be non-nil")
	}
	if len(s.Installed) != 0 {
		t.Errorf("expected empty Installed, got %d entries", len(s.Installed))
	}
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	data := `{"installed":{"tunnel":{"name":"tunnel","version":"v1.0.0"}}}`
	os.WriteFile(path, []byte(data), 0644)

	s, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tool, ok := s.Get("tunnel")
	if !ok {
		t.Fatal("expected tunnel to be installed")
	}
	if tool.Version != "v1.0.0" {
		t.Errorf("version = %q, want %q", tool.Version, "v1.0.0")
	}
}

func TestLoad_CorruptJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	os.WriteFile(path, []byte("{not valid json"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for corrupt JSON, got nil")
	}
}

func TestLoad_NullInstalled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	os.WriteFile(path, []byte(`{"installed":null}`), 0644)

	s, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Installed == nil {
		t.Fatal("Installed map should be non-nil even when JSON has null")
	}
}

func TestSave_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "state.json")

	s := &State{Installed: make(map[string]InstalledTool)}
	s.Set("tunnel", "v1.0.0")

	if err := s.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists and is valid JSON
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read saved file: %v", err)
	}

	var loaded State
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("saved file is not valid JSON: %v", err)
	}
	if loaded.Installed["tunnel"].Version != "v1.0.0" {
		t.Errorf("loaded version = %q, want %q", loaded.Installed["tunnel"].Version, "v1.0.0")
	}
}

func TestSave_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Write initial state
	s := &State{Installed: make(map[string]InstalledTool)}
	s.Set("tunnel", "v1.0.0")
	if err := s.Save(path); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}

	// Overwrite with new state
	s.Set("merge-port", "v2.0.0")
	if err := s.Save(path); err != nil {
		t.Fatalf("second Save failed: %v", err)
	}

	// Verify no temp files left behind
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "state.json" {
			t.Errorf("unexpected file left behind: %s", e.Name())
		}
	}

	// Verify content
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if _, ok := loaded.Get("tunnel"); !ok {
		t.Error("tunnel should be in state")
	}
	if _, ok := loaded.Get("merge-port"); !ok {
		t.Error("merge-port should be in state")
	}
}

func TestSetGetRemove(t *testing.T) {
	s := &State{Installed: make(map[string]InstalledTool)}

	// Set
	s.Set("tunnel", "v1.0.0")
	tool, ok := s.Get("tunnel")
	if !ok {
		t.Fatal("expected tunnel after Set")
	}
	if tool.Version != "v1.0.0" {
		t.Errorf("version = %q, want %q", tool.Version, "v1.0.0")
	}

	// Remove
	s.Remove("tunnel")
	_, ok = s.Get("tunnel")
	if ok {
		t.Fatal("expected tunnel to be gone after Remove")
	}
}
