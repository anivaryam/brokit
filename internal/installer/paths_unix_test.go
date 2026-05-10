//go:build !windows

package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultBinDir_WithBROKIT_BIN(t *testing.T) {
	t.Setenv("BROKIT_BIN", "/custom/bin")
	dir, err := defaultBinDir()
	require.NoError(t, err)
	assert.Equal(t, "/custom/bin", dir)
}

func TestDefaultBinDir_WithHOME(t *testing.T) {
	t.Setenv("BROKIT_BIN", "")
	t.Setenv("HOME", "/home/testuser")
	dir, err := defaultBinDir()
	require.NoError(t, err)
	assert.Equal(t, "/home/testuser/.local/bin", dir)
}

func TestDefaultBinDir_EmptyHOME(t *testing.T) {
	t.Setenv("BROKIT_BIN", "")
	t.Setenv("HOME", "")
	_, err := defaultBinDir()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HOME")
}

func TestStateFilePath_WithHOME(t *testing.T) {
	t.Setenv("HOME", "/home/testuser")
	path, err := stateFilePath()
	require.NoError(t, err)
	assert.Contains(t, path, "brokit")
	assert.Contains(t, path, "state.json")
}

func TestStateFilePath_EmptyHOME(t *testing.T) {
	t.Setenv("HOME", "")
	_, err := stateFilePath()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HOME")
}
