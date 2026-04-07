package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// InstalledTool records an installed tool and its version.
type InstalledTool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// State tracks which tools are installed.
type State struct {
	Installed map[string]InstalledTool `json:"installed"`
}

// Load reads state from the given path. Returns empty state if the file doesn't exist.
func Load(path string) (*State, error) {
	s := &State{Installed: make(map[string]InstalledTool)}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	if s.Installed == nil {
		s.Installed = make(map[string]InstalledTool)
	}
	return s, nil
}

// Save writes state to the given path, creating parent directories as needed.
func (s *State) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Set records a tool as installed with the given version.
func (s *State) Set(name, version string) {
	s.Installed[name] = InstalledTool{Name: name, Version: version}
}

// Remove deletes a tool from the installed set.
func (s *State) Remove(name string) {
	delete(s.Installed, name)
}

// Get returns an installed tool's info. The bool is false if not installed.
func (s *State) Get(name string) (InstalledTool, bool) {
	t, ok := s.Installed[name]
	return t, ok
}
