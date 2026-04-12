package installer

import (
	"os"
	"testing"

	"github.com/anivaryam/brokit/internal/registry"
	"github.com/anivaryam/brokit/internal/state"
	"github.com/stretchr/testify/assert"
)

type mockRegistry struct{}

func (m mockRegistry) Get(name string) (registry.Tool, bool) { return registry.Get(name) }
func (m mockRegistry) All() []registry.Tool                  { return registry.All() }
func (m mockRegistry) Names() []string                       { return registry.Names() }

type mockState struct{}

func (m mockState) Get(name string) (state.InstalledTool, bool) { return state.InstalledTool{}, false }
func (m mockState) Set(t state.InstalledTool) error             { return nil }
func (m mockState) Remove(name string) error                    { return nil }
func (m mockState) List() []state.InstalledTool                 { return nil }

type mockFetcher struct{}

func (m mockFetcher) Latest(repo string) (string, error) { return "v1.0.0", nil }

// ─── Network errors ──────────────────────────────────────────────────────────

func TestWrapNetworkError_Nil(t *testing.T) {
	err := wrapNetworkError(nil)
	assert.NoError(t, err)
}

func TestWrapNetworkError_NonNetworkError(t *testing.T) {
	original := os.ErrNotExist
	wrapped := wrapNetworkError(original)
	assert.Equal(t, original, wrapped)
}

// ─── formatBytes ─────────────────────────────────────────────────────────────

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{5242880, "5.0 MB"},
		{10485760, "10.0 MB"},
	}
	for _, tt := range tests {
		got := formatBytes(tt.bytes)
		assert.Equal(t, tt.want, got)
	}
}
