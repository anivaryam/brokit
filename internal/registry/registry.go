package registry

import "sort"

// Tool describes a tool available in the registry.
type Tool struct {
	Name        string
	Description string
	Repo        string // GitHub "owner/repo"
	Binary      string // binary name without extension
}

var tools = map[string]Tool{
	"env-vault": {
		Name:        "env-vault",
		Description: "Encrypted .env file manager powered by random-universe-cipher",
		Repo:        "anivaryam/env-vault",
		Binary:      "env-vault",
	},
	"merge-port": {
		Name:        "merge-port",
		Description: "Local reverse proxy that merges multiple ports into one",
		Repo:        "anivaryam/merge-port",
		Binary:      "merge-port",
	},
	"proc-compose": {
		Name:        "proc-compose",
		Description: "Process runner and manager with daemon support",
		Repo:        "anivaryam/proc-compose",
		Binary:      "proc-compose",
	},
	"proxy-relay": {
		Name:        "proxy-relay",
		Description: "Authenticated SOCKS5/HTTP proxy client for routing traffic through a remote server",
		Repo:        "anivaryam/proxy-relay",
		Binary:      "proxy-relay",
	},
	"tunnel": {
		Name:        "tunnel",
		Description: "Expose local services through a public tunnel",
		Repo:        "anivaryam/tunnel",
		Binary:      "tunnel",
	},
}

// Get returns a tool by name.
func Get(name string) (Tool, bool) {
	t, ok := tools[name]
	return t, ok
}

// All returns all registered tools sorted by name.
func All() []Tool {
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]Tool, len(names))
	for i, name := range names {
		result[i] = tools[name]
	}
	return result
}

// Names returns all tool names sorted.
func Names() []string {
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
