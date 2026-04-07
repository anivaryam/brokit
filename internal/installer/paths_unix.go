//go:build !windows

package installer

import (
	"fmt"
	"os"
	"path/filepath"
)

func defaultBinDir() (string, error) {
	if dir := os.Getenv("BROKIT_BIN"); dir != "" {
		return dir, nil
	}
	home := os.Getenv("HOME")
	if home == "" {
		return "", fmt.Errorf("HOME environment variable is not set")
	}
	return filepath.Join(home, ".local", "bin"), nil
}

func stateFilePath() (string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return "", fmt.Errorf("HOME environment variable is not set")
	}
	return filepath.Join(home, ".local", "share", "brokit", "state.json"), nil
}
