package installer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/anivaryam/brokit/internal/registry"
	"github.com/anivaryam/brokit/internal/state"
)

// LogLevel controls output verbosity.
type LogLevel int

const (
	LogQuiet   LogLevel = -1
	LogNormal  LogLevel = 0
	LogVerbose LogLevel = 1
)

// Installer manages tool installation, updates, and removal.
type Installer struct {
	State     *state.State
	statePath string
	BinDir    string
	LogLevel  LogLevel
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// HTTP clients with appropriate timeouts.
var (
	apiClient      = &http.Client{Timeout: 30 * time.Second}
	downloadClient = &http.Client{Timeout: 5 * time.Minute}
)

// Base URLs — overridable in tests.
var (
	githubAPIBase = "https://api.github.com"
	githubBase    = "https://github.com"
)

// New creates an Installer, loading existing state and ensuring the bin directory exists.
func New() (*Installer, error) {
	sp, err := stateFilePath()
	if err != nil {
		return nil, fmt.Errorf("determining state path: %w", err)
	}
	s, err := state.Load(sp)
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	bd, err := defaultBinDir()
	if err != nil {
		return nil, fmt.Errorf("determining bin directory: %w", err)
	}
	if err := os.MkdirAll(bd, 0755); err != nil {
		return nil, fmt.Errorf("creating bin directory: %w", err)
	}

	return &Installer{State: s, statePath: sp, BinDir: bd}, nil
}

// ─── Logging helpers ─────────────────────────────────────────────────────────

func (inst *Installer) log(format string, args ...any) {
	if inst.LogLevel >= LogNormal {
		fmt.Printf(format, args...)
	}
}

func (inst *Installer) verbose(format string, args ...any) {
	if inst.LogLevel >= LogVerbose {
		fmt.Printf(format, args...)
	}
}

// ─── Install / Update / Remove ───────────────────────────────────────────────

// Install downloads and installs a tool.
func (inst *Installer) Install(name string, force bool) error {
	tool, ok := registry.Get(name)
	if !ok {
		return fmt.Errorf("unknown tool: %s\navailable: %s", name, strings.Join(registry.Names(), ", "))
	}

	if existing, ok := inst.State.Get(name); ok && !force {
		return fmt.Errorf("%s is already installed (%s), use 'brokit update %s' to update", name, existing.Version, name)
	}

	version, err := latestVersion(tool.Repo)
	if err != nil {
		return fmt.Errorf("fetching latest version: %w", err)
	}

	inst.log("Installing %s %s...\n", name, version)
	if err := inst.downloadAndInstall(tool, version); err != nil {
		return err
	}

	inst.State.Set(name, version)
	if err := inst.State.Save(inst.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	inst.log("Installed %s %s -> %s\n", name, version, inst.BinDir)
	inst.warnIfNotInPath()
	return nil
}

// Update fetches the latest version and reinstalls if newer.
func (inst *Installer) Update(name string) error {
	tool, ok := registry.Get(name)
	if !ok {
		return fmt.Errorf("unknown tool: %s", name)
	}

	existing, installed := inst.State.Get(name)
	if !installed {
		return fmt.Errorf("%s is not installed, use 'brokit install %s' first", name, name)
	}

	version, err := latestVersion(tool.Repo)
	if err != nil {
		return fmt.Errorf("fetching latest version: %w", err)
	}

	if version == existing.Version {
		inst.log("%s is already up to date (%s)\n", name, version)
		return nil
	}

	inst.log("Updating %s %s -> %s...\n", name, existing.Version, version)
	if err := inst.downloadAndInstall(tool, version); err != nil {
		return err
	}

	inst.State.Set(name, version)
	if err := inst.State.Save(inst.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	inst.log("Updated %s to %s\n", name, version)
	return nil
}

// UpdateTo updates a tool to a specific pre-fetched version (avoids re-fetching).
func (inst *Installer) UpdateTo(name, version string) error {
	tool, ok := registry.Get(name)
	if !ok {
		return fmt.Errorf("unknown tool: %s", name)
	}

	existing, installed := inst.State.Get(name)
	if !installed {
		return fmt.Errorf("%s is not installed", name)
	}

	if version == existing.Version {
		inst.log("%s is already up to date (%s)\n", name, version)
		return nil
	}

	inst.log("Updating %s %s -> %s...\n", name, existing.Version, version)
	if err := inst.downloadAndInstall(tool, version); err != nil {
		return err
	}

	inst.State.Set(name, version)
	if err := inst.State.Save(inst.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	inst.log("Updated %s to %s\n", name, version)
	return nil
}

// Remove deletes an installed tool's binary and state entry.
func (inst *Installer) Remove(name string) error {
	tool, ok := registry.Get(name)
	if !ok {
		return fmt.Errorf("unknown tool: %s", name)
	}

	if _, ok := inst.State.Get(name); !ok {
		return fmt.Errorf("%s is not installed", name)
	}

	bin := tool.Binary
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	path := filepath.Join(inst.BinDir, bin)

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: binary not found at %s\n", path)
		} else {
			return fmt.Errorf("removing binary: %w", err)
		}
	}

	inst.State.Remove(name)
	if err := inst.State.Save(inst.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	inst.log("Removed %s\n", name)

	// Warn if the command is still reachable via another PATH entry.
	if found, _ := exec.LookPath(bin); found != "" {
		fmt.Fprintf(os.Stderr, "Warning: %s is still available at %s (not managed by brokit)\n", name, found)
	}

	return nil
}

// SelfUpdate updates brokit itself to the latest version.
func (inst *Installer) SelfUpdate(currentVersion string) error {
	repo := "anivaryam/brokit"
	binary := "brokit"

	version, err := latestVersion(repo)
	if err != nil {
		return fmt.Errorf("fetching latest version: %w", err)
	}

	if version == currentVersion {
		inst.log("brokit is already up to date (%s)\n", version)
		return nil
	}

	inst.log("Updating brokit %s -> %s...\n", currentVersion, version)

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	tool := registry.Tool{
		Name:   "brokit",
		Repo:   repo,
		Binary: binary,
	}

	origBinDir := inst.BinDir
	inst.BinDir = filepath.Dir(execPath)
	defer func() { inst.BinDir = origBinDir }()

	if err := inst.downloadAndInstall(tool, version); err != nil {
		return err
	}

	inst.log("Updated brokit to %s\n", version)
	return nil
}

// InstalledNames returns names of all installed tools.
func (inst *Installer) InstalledNames() []string {
	names := make([]string, 0, len(inst.State.Installed))
	for name := range inst.State.Installed {
		names = append(names, name)
	}
	return names
}

// LatestVersion fetches the latest release version for the given repo.
func LatestVersion(repo string) (string, error) {
	return latestVersion(repo)
}

// ─── Internal helpers ────────────────────────────────────────────────────────

func (inst *Installer) warnIfNotInPath() {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if filepath.Clean(dir) == filepath.Clean(inst.BinDir) {
			return
		}
	}
	inst.log("\nWarning: %s is not in your PATH\n", inst.BinDir)
	if runtime.GOOS == "windows" {
		inst.log("Run this in PowerShell to add it:\n")
		inst.log("  [Environment]::SetEnvironmentVariable(\"Path\", $env:Path + \";%s\", \"User\")\n", inst.BinDir)
	} else {
		inst.log("Add this to your shell profile:\n")
		inst.log("  export PATH=\"%s:$PATH\"\n", inst.BinDir)
	}
}

func latestVersion(repo string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPIBase, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := apiClient.Do(req)
	if err != nil {
		return "", wrapNetworkError(err)
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

func wrapNetworkError(err error) error {
	if err == nil {
		return nil
	}
	var dnsErr *net.DNSError
	var opErr *net.OpError
	if errors.As(err, &dnsErr) {
		return fmt.Errorf("network error: cannot reach %s — check your internet connection", dnsErr.Name)
	}
	if errors.As(err, &opErr) {
		return fmt.Errorf("network error: %s — check your internet connection", opErr.Op)
	}
	return err
}

func (inst *Installer) downloadAndInstall(tool registry.Tool, version string) error {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	ext := "tar.gz"
	if osName == "windows" {
		ext = "zip"
	}

	url := fmt.Sprintf("%s/%s/releases/download/%s/%s_%s_%s.%s",
		githubBase, tool.Repo, version, tool.Binary, osName, arch, ext)

	inst.verbose("Download URL: %s\n", url)

	tmpDir, err := os.MkdirTemp("", "brokit-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, fmt.Sprintf("archive.%s", ext))

	// Download
	resp, err := downloadClient.Get(url)
	if err != nil {
		return wrapNetworkError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d (%s)", resp.StatusCode, url)
	}

	inst.verbose("HTTP %d, Content-Length: %d\n", resp.StatusCode, resp.ContentLength)

	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}

	// Wrap with progress writer for user feedback
	var src io.Reader = resp.Body
	if inst.LogLevel >= LogNormal {
		src = &progressReader{
			r:       resp.Body,
			total:   resp.ContentLength,
			name:    tool.Binary,
			version: version,
		}
	}

	written, err := io.Copy(f, src)
	if err != nil {
		f.Close()
		return fmt.Errorf("saving download: %w", err)
	}
	f.Close()

	// Clear progress line
	if inst.LogLevel >= LogNormal {
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 80))
	}

	// Validate download completeness
	if resp.ContentLength > 0 && written != resp.ContentLength {
		os.Remove(archivePath)
		return fmt.Errorf("incomplete download: got %d bytes, expected %d", written, resp.ContentLength)
	}

	// Extract
	binaryName := tool.Binary
	if osName == "windows" {
		binaryName += ".exe"
	}

	if ext == "zip" {
		err = extractZip(archivePath, tmpDir, binaryName)
	} else {
		err = extractTarGz(archivePath, tmpDir, binaryName)
	}
	if err != nil {
		return fmt.Errorf("extracting: %w", err)
	}

	// Install binary using atomic rename to handle "text file busy" on Linux
	srcPath := filepath.Join(tmpDir, binaryName)
	dst := filepath.Join(inst.BinDir, binaryName)

	tmpFile, err := os.CreateTemp(inst.BinDir, binaryName+".*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	srcFile, err := os.Open(srcPath)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("reading extracted binary: %w", err)
	}

	if _, err := io.Copy(tmpFile, srcFile); err != nil {
		srcFile.Close()
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing binary: %w", err)
	}
	srcFile.Close()
	tmpFile.Close()

	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("setting permissions: %w", err)
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("installing binary: %w", err)
	}

	return nil
}

// ─── Progress ────────────────────────────────────────────────────────────────

type progressReader struct {
	r       io.Reader
	read    int64
	total   int64
	name    string
	version string
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	pr.read += int64(n)
	if pr.total > 0 {
		fmt.Fprintf(os.Stderr, "\rDownloading %s %s... %s / %s",
			pr.name, pr.version,
			formatBytes(pr.read), formatBytes(pr.total))
	} else {
		fmt.Fprintf(os.Stderr, "\rDownloading %s %s... %s",
			pr.name, pr.version, formatBytes(pr.read))
	}
	return n, err
}

func formatBytes(b int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)
	switch {
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// ─── Archive extraction ──────────────────────────────────────────────────────

func extractTarGz(archive, destDir, target string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if filepath.Base(hdr.Name) == target && hdr.Typeflag == tar.TypeReg {
			out, err := os.Create(filepath.Join(destDir, target))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			return out.Close()
		}
	}
	return fmt.Errorf("binary %s not found in archive", target)
}

func extractZip(archive, destDir, target string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == target {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			out, err := os.Create(filepath.Join(destDir, target))
			if err != nil {
				rc.Close()
				return err
			}
			if _, err := io.Copy(out, rc); err != nil {
				out.Close()
				rc.Close()
				return err
			}
			rc.Close()
			return out.Close()
		}
	}
	return fmt.Errorf("binary %s not found in archive", target)
}
