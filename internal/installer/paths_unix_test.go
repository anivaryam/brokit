//go:build !windows

package installer

import (
	"strings"
	"testing"
)

func TestDefaultBinDir_WithBROKIT_BIN(t *testing.T) {
	t.Setenv("BROKIT_BIN", "/custom/bin")
	dir, err := defaultBinDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir != "/custom/bin" {
		t.Errorf("dir = %q, want %q", dir, "/custom/bin")
	}
}

func TestDefaultBinDir_WithHOME(t *testing.T) {
	t.Setenv("BROKIT_BIN", "")
	t.Setenv("HOME", "/home/testuser")
	dir, err := defaultBinDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir != "/home/testuser/.local/bin" {
		t.Errorf("dir = %q, want %q", dir, "/home/testuser/.local/bin")
	}
}

func TestDefaultBinDir_EmptyHOME(t *testing.T) {
	t.Setenv("BROKIT_BIN", "")
	t.Setenv("HOME", "")
	_, err := defaultBinDir()
	if err == nil {
		t.Fatal("expected error when HOME is empty")
	}
	if !strings.Contains(err.Error(), "HOME") {
		t.Errorf("error = %q, want mention of HOME", err)
	}
}

func TestStateFilePath_WithHOME(t *testing.T) {
	t.Setenv("HOME", "/home/testuser")
	path, err := stateFilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(path, "brokit") || !strings.Contains(path, "state.json") {
		t.Errorf("path = %q, expected to contain 'brokit' and 'state.json'", path)
	}
}

func TestStateFilePath_EmptyHOME(t *testing.T) {
	t.Setenv("HOME", "")
	_, err := stateFilePath()
	if err == nil {
		t.Fatal("expected error when HOME is empty")
	}
	if !strings.Contains(err.Error(), "HOME") {
		t.Errorf("error = %q, want mention of HOME", err)
	}
}
