package registry

// Tool represents a tool in the registry.
type Tool struct {
	Name        string `toml:"name"`
	Repo        string `toml:"repo"`        // github.com/{owner}/{repo}
	Binary      string `toml:"binary"`      // Name of binary in release
	Description string `toml:"description"` // Human-readable description
}
