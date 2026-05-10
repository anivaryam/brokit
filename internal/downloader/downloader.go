package downloader

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Masterminds/semver/v3"
	brokiterrors "github.com/anivaryam/brokit/internal/errors"
	"github.com/anivaryam/brokit/internal/installer"
)

var githubAPIBase = "https://api.github.com"

type githubRelease struct {
	TagName string `json:"tag_name"`
}

type Downloader struct {
	client    *http.Client
	userAgent string
}

func NewDownloader(client *http.Client, userAgent string) *Downloader {
	return &Downloader{client: client, userAgent: userAgent}
}

func (d *Downloader) Latest(repo string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPIBase, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", brokiterrors.WrapNetworkError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		return "", formatRateLimitError(resp)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, repo)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	if release.TagName == "" {
		return "", fmt.Errorf("no releases found for %s", repo)
	}
	return release.TagName, nil
}

func (d *Downloader) VersionExists(repo, version string) (bool, error) {
	latest, err := d.Latest(repo)
	if err != nil {
		return false, err
	}

	// Use semver for proper comparison
	v1, err := semver.NewVersion(version)
	if err != nil {
		return false, err
	}
	v2, err := semver.NewVersion(latest)
	if err != nil {
		return false, err
	}

	return v1.LessThan(v2), nil
}

var _ installer.VersionFetcher = (*Downloader)(nil)

func formatRateLimitError(resp *http.Response) error {
	remaining := resp.Header.Get("X-RateLimit-Remaining")
	resetStr := resp.Header.Get("X-RateLimit-Reset")

	msg := "GitHub API rate limit exceeded"

	if resetStr != "" {
		if resetUnix, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
			resetTime := time.Unix(resetUnix, 0)
			wait := time.Until(resetTime).Round(time.Second)
			if wait > 0 {
				msg += fmt.Sprintf(" (resets in %s)", wait)
			}
		}
	}

	if remaining == "0" {
		msg += "\nTip: set GITHUB_TOKEN to increase your rate limit to 5000 requests/hour"
	}

	return fmt.Errorf("%s", msg)
}
