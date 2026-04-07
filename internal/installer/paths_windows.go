//go:build windows

package installer

import (
	"os"
	"path/filepath"
)

func defaultBinDir() string {
	if dir := os.Getenv("BROKIT_BIN"); dir != "" {
		return dir
	}
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "brokit", "bin")
}

func stateFilePath() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "brokit", "state.json")
}
