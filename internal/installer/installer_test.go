package installer

import (
	"os"
	"testing"

	brokiterrors "github.com/anivaryam/brokit/internal/errors"
	"github.com/stretchr/testify/assert"
)

// ─── Network errors ──────────────────────────────────────────────────────────

func TestWrapNetworkError_Nil(t *testing.T) {
	err := brokiterrors.WrapNetworkError(nil)
	assert.NoError(t, err)
}

func TestWrapNetworkError_NonNetworkError(t *testing.T) {
	original := os.ErrNotExist
	wrapped := brokiterrors.WrapNetworkError(original)
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
