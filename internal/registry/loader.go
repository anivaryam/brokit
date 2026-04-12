package registry

import (
	"fmt"

	"github.com/spf13/viper"
)

// Load loads tools from a TOML config file.
// If the file doesn't exist, it returns the default tools.
// If the file exists but can't be parsed, it returns an error.
func Load(configPath string) ([]Tool, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("toml")

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// No config file - use defaults
			return DefaultTools, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	// Parse config
	var tools []Tool
	if err := v.Unmarshal(&tools); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	// Merge with defaults (user config takes precedence)
	toolMap := make(map[string]Tool)
	for _, t := range DefaultTools {
		toolMap[t.Name] = t
	}
	for _, t := range tools {
		toolMap[t.Name] = t
	}

	result := make([]Tool, 0, len(toolMap))
	for _, t := range toolMap {
		result = append(result, t)
	}

	return result, nil
}
