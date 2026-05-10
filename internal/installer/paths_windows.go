//go:build windows

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
	appdata := os.Getenv("LOCALAPPDATA")
	if appdata == "" {
		return "", fmt.Errorf("LOCALAPPDATA environment variable is not set")
	}
	return filepath.Join(appdata, "brokit", "bin"), nil
}

func stateFilePath() (string, error) {
	appdata := os.Getenv("LOCALAPPDATA")
	if appdata == "" {
		return "", fmt.Errorf("LOCALAPPDATA environment variable is not set")
	}
	return filepath.Join(appdata, "brokit", "state.json"), nil
}
