package downloader

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Masterminds/semver/v3"
	brokiterrors "github.com/anivaryam/brokit/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLatestVersion_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(githubRelease{TagName: "v1.2.3"})
	}))
	defer ts.Close()

	orig := githubAPIBase
	githubAPIBase = ts.URL
	defer func() { githubAPIBase = orig }()

	client := NewClient("test-agent")
	dl := NewDownloader(client, "test-agent")

	ver, err := dl.Latest("test/repo")
	require.NoError(t, err)
	assert.Equal(t, "v1.2.3", ver)
}

func TestLatestVersion_RateLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "9999999999")
		w.WriteHeader(403)
	}))
	defer ts.Close()

	orig := githubAPIBase
	githubAPIBase = ts.URL
	defer func() { githubAPIBase = orig }()

	client := NewClient("test-agent")
	dl := NewDownloader(client, "test-agent")

	_, err := dl.Latest("test/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit")
	assert.Contains(t, err.Error(), "GITHUB_TOKEN")
}

func TestLatestVersion_EmptyTag(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(githubRelease{TagName: ""})
	}))
	defer ts.Close()

	orig := githubAPIBase
	githubAPIBase = ts.URL
	defer func() { githubAPIBase = orig }()

	client := NewClient("test-agent")
	dl := NewDownloader(client, "test-agent")

	_, err := dl.Latest("test/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no releases")
}

func TestLatestVersion_WithGitHubToken(t *testing.T) {
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(githubRelease{TagName: "v1.0.0"})
	}))
	defer ts.Close()

	orig := githubAPIBase
	githubAPIBase = ts.URL
	defer func() { githubAPIBase = orig }()

	t.Setenv("GITHUB_TOKEN", "test-token-123")

	client := NewClient("test-agent")
	dl := NewDownloader(client, "test-agent")

	_, err := dl.Latest("test/repo")
	require.NoError(t, err)
	assert.Equal(t, "Bearer test-token-123", gotAuth)
}

func TestVersionExists(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(githubRelease{TagName: "v2.0.0"})
	}))
	defer ts.Close()

	orig := githubAPIBase
	githubAPIBase = ts.URL
	defer func() { githubAPIBase = orig }()

	client := NewClient("test-agent")
	dl := NewDownloader(client, "test-agent")

	exists, err := dl.VersionExists("test/repo", "v1.0.0")
	require.NoError(t, err)
	assert.True(t, exists, "v1.0.0 should be less than v2.0.0")
}

func TestVersionExists_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(githubRelease{TagName: "v0.0.1"})
	}))
	defer ts.Close()

	orig := githubAPIBase
	githubAPIBase = ts.URL
	defer func() { githubAPIBase = orig }()

	client := NewClient("test-agent")
	dl := NewDownloader(client, "test-agent")

	exists, err := dl.VersionExists("test/repo", "v0.0.1")
	require.NoError(t, err)
	assert.False(t, exists, "v0.0.1 should not be less than v0.0.1")
}

func TestWrapNetworkError_Nil(t *testing.T) {
	err := brokiterrors.WrapNetworkError(nil)
	assert.NoError(t, err)
}

func TestWrapNetworkError_NonNetworkError(t *testing.T) {
	original := os.ErrNotExist
	wrapped := brokiterrors.WrapNetworkError(original)
	assert.Equal(t, original, wrapped)
}

func TestSemverComparisons(t *testing.T) {
	v1, _ := semver.NewVersion("v1.10.0")
	v2, _ := semver.NewVersion("v1.9.0")
	assert.True(t, v1.GreaterThan(v2), "v1.10.0 should be greater than v1.9.0")
}
