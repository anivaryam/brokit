package installer

import (
	"github.com/anivaryam/brokit/internal/registry"
	"github.com/anivaryam/brokit/internal/state"
)

// Registry defines the interface for tool registry operations.
type Registry interface {
	Get(name string) (registry.Tool, bool)
	All() []registry.Tool
	Names() []string
}

// StateManager defines the interface for installed tool state operations.
type StateManager interface {
	Get(name string) (state.InstalledTool, bool)
	Set(t state.InstalledTool) error
	Remove(name string) error
	List() []state.InstalledTool
	Save(path string) error
}

// VersionFetcher defines the interface for fetching the latest version of a tool.
type VersionFetcher interface {
	Latest(repo string) (string, error)
}
