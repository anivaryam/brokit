package installer

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/anivaryam/brokit/internal/extractor"
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
	registry  Registry
	state     StateManager
	fetcher   VersionFetcher
	State     *state.State
	statePath string
	BinDir    string
	LogLevel  LogLevel
}

// NewInstaller creates an Installer with the given dependencies.
func NewInstaller(registry Registry, state StateManager, fetcher VersionFetcher) *Installer {
	bd, _ := defaultBinDir()
	return &Installer{
		registry: registry,
		state:    state,
		fetcher:  fetcher,
		BinDir:   bd,
	}
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
	tool, ok := inst.registry.Get(name)
	if !ok {
		return fmt.Errorf("unknown tool: %s\navailable: %s", name, strings.Join(inst.registry.Names(), ", "))
	}

	if existing, ok := inst.state.Get(name); ok && !force {
		return fmt.Errorf("%s is already installed (%s), use 'brokit update %s' to update", name, existing.Version, name)
	}

	version, err := inst.fetcher.Latest(tool.Repo)
	if err != nil {
		return fmt.Errorf("fetching latest version: %w", err)
	}

	inst.log("Installing %s %s...\n", name, version)
	if err := inst.downloadAndInstall(tool, version); err != nil {
		return err
	}

	inst.state.Set(state.InstalledTool{Name: name, Version: version})
	if err := inst.State.Save(inst.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	inst.log("Installed %s %s -> %s\n", name, version, inst.BinDir)
	inst.warnIfNotInPath()
	return nil
}

// Update fetches the latest version and reinstalls if newer.
func (inst *Installer) Update(name string) error {
	tool, ok := inst.registry.Get(name)
	if !ok {
		return fmt.Errorf("unknown tool: %s", name)
	}

	existing, installed := inst.state.Get(name)
	if !installed {
		return fmt.Errorf("%s is not installed, use 'brokit install %s' first", name, name)
	}

	version, err := inst.fetcher.Latest(tool.Repo)
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

	inst.state.Set(state.InstalledTool{Name: name, Version: version})
	if err := inst.State.Save(inst.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	inst.log("Updated %s to %s\n", name, version)
	return nil
}

// UpdateTo updates a tool to a specific pre-fetched version (avoids re-fetching).
func (inst *Installer) UpdateTo(name, version string) error {
	tool, ok := inst.registry.Get(name)
	if !ok {
		return fmt.Errorf("unknown tool: %s", name)
	}

	existing, installed := inst.state.Get(name)
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

	inst.state.Set(state.InstalledTool{Name: name, Version: version})
	if err := inst.State.Save(inst.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	inst.log("Updated %s to %s\n", name, version)
	return nil
}

// ErrBinaryNotFound is returned when the binary file does not exist.
var ErrBinaryNotFound = errors.New("binary not found")

// Remove deletes an installed tool's binary and state entry.
func (inst *Installer) Remove(name string) error {
	tool, ok := inst.registry.Get(name)
	if !ok {
		return fmt.Errorf("unknown tool: %s", name)
	}

	if _, ok := inst.state.Get(name); !ok {
		return fmt.Errorf("%s is not installed", name)
	}

	bin := tool.Binary
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	path := filepath.Join(inst.BinDir, bin)

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrBinaryNotFound, path)
		}
		return fmt.Errorf("removing binary: %w", err)
	}

	inst.state.Remove(name)
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

	version, err := inst.fetcher.Latest(repo)
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
	installed := inst.state.List()
	names := make([]string, 0, len(installed))
	for _, tool := range installed {
		names = append(names, tool.Name)
	}
	return names
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
	githubBase := "https://github.com"
	downloadClient := &http.Client{Timeout: 5 * time.Minute}

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

	var srcPath string
	if ext == "zip" {
		srcPath, err = extractor.ExtractZip(tmpDir, archivePath, binaryName)
	} else {
		srcPath, err = extractor.ExtractTarGz(tmpDir, archivePath, binaryName)
	}
	if err != nil {
		return fmt.Errorf("extracting: %w", err)
	}

	// Install binary using atomic rename to handle "text file busy" on Linux
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
