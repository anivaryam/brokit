package registry

import (
	"sort"
	"sync"
)

var (
	toolsCache []Tool
	toolsOnce  sync.Once
)

func getTools() []Tool {
	toolsOnce.Do(func() {
		// Load from default config path (empty string uses defaults)
		toolsCache, _ = Load("")
	})
	return toolsCache
}

// Get returns a tool by name.
func Get(name string) (Tool, bool) {
	tools := getTools()
	for _, t := range tools {
		if t.Name == name {
			return t, true
		}
	}
	return Tool{}, false
}

// All returns all registered tools sorted by name.
func All() []Tool {
	tools := getTools()
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	sort.Strings(names)

	result := make([]Tool, len(names))
	for i, name := range names {
		for _, t := range tools {
			if t.Name == name {
				result[i] = t
				break
			}
		}
	}
	return result
}

// Names returns all tool names sorted.
func Names() []string {
	tools := getTools()
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	sort.Strings(names)
	return names
}
