package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet_ExistingTool(t *testing.T) {
	tool, ok := Get("tunnel")
	require.True(t, ok, "expected tool to exist")
	assert.Equal(t, "tunnel", tool.Name)
	assert.Equal(t, "anivaryam/tunnel", tool.Repo)
	assert.Equal(t, "tunnel", tool.Binary)
}

func TestGet_UnknownTool(t *testing.T) {
	_, ok := Get("nonexistent")
	assert.False(t, ok, "expected nonexistent tool to not be found")
}

func TestAll_ReturnsSortedTools(t *testing.T) {
	all := All()
	assert.NotEmpty(t, all, "expected at least one tool")
	for i := 1; i < len(all); i++ {
		assert.True(t, all[i].Name >= all[i-1].Name, "tools not sorted: %q comes after %q", all[i].Name, all[i-1].Name)
	}
}

func TestNames_ReturnsSortedNames(t *testing.T) {
	names := Names()
	assert.NotEmpty(t, names, "expected at least one name")
	for i := 1; i < len(names); i++ {
		assert.True(t, names[i] >= names[i-1], "names not sorted: %q comes after %q", names[i], names[i-1])
	}
}

func TestAll_NamesConsistency(t *testing.T) {
	all := All()
	names := Names()
	assert.Equal(t, len(all), len(names), "All() returned %d items, Names() returned %d", len(all), len(names))
	for i := range all {
		assert.Equal(t, all[i].Name, names[i], "index %d: All name %q != Names name %q", i, all[i].Name, names[i])
	}
}
