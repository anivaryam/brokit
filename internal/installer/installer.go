package installer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/anivaryam/brokit/internal/registry"
	"github.com/anivaryam/brokit/internal/state"
)

// Installer manages tool installation, updates, and removal.
type Installer struct {
	State     *state.State
	statePath string
	BinDir    string
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// New creates an Installer, loading existing state and ensuring the bin directory exists.
func New() (*Installer, error) {
	sp := stateFilePath()
	s, err := state.Load(sp)
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	bd := defaultBinDir()
	if err := os.MkdirAll(bd, 0755); err != nil {
		return nil, fmt.Errorf("creating bin directory: %w", err)
	}

	return &Installer{State: s, statePath: sp, BinDir: bd}, nil
}

// Install downloads and installs a tool.
func (inst *Installer) Install(name string) error {
	tool, ok := registry.Get(name)
	if !ok {
		return fmt.Errorf("unknown tool: %s\navailable: %s", name, strings.Join(registry.Names(), ", "))
	}

	if existing, ok := inst.State.Get(name); ok {
		return fmt.Errorf("%s is already installed (%s), use 'brokit update %s' to update", name, existing.Version, name)
	}

	version, err := latestVersion(tool.Repo)
	if err != nil {
		return fmt.Errorf("fetching latest version: %w", err)
	}

	fmt.Printf("Installing %s %s...\n", name, version)
	if err := inst.downloadAndInstall(tool, version); err != nil {
		return err
	}

	inst.State.Set(name, version)
	if err := inst.State.Save(inst.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("Installed %s %s -> %s\n", name, version, inst.BinDir)
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
		fmt.Printf("%s is already up to date (%s)\n", name, version)
		return nil
	}

	fmt.Printf("Updating %s %s -> %s...\n", name, existing.Version, version)
	if err := inst.downloadAndInstall(tool, version); err != nil {
		return err
	}

	inst.State.Set(name, version)
	if err := inst.State.Save(inst.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("Updated %s to %s\n", name, version)
	return nil
}

// Remove deletes an installed tool's binary and state entry.
func (inst *Installer) Remove(name string) error {
	if _, ok := registry.Get(name); !ok {
		return fmt.Errorf("unknown tool: %s", name)
	}

	if _, ok := inst.State.Get(name); !ok {
		return fmt.Errorf("%s is not installed", name)
	}

	bin := name
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	path := filepath.Join(inst.BinDir, bin)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing binary: %w", err)
	}

	inst.State.Remove(name)
	if err := inst.State.Save(inst.statePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("Removed %s\n", name)
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

func (inst *Installer) warnIfNotInPath() {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if filepath.Clean(dir) == filepath.Clean(inst.BinDir) {
			return
		}
	}
	fmt.Printf("\nWarning: %s is not in your PATH\n", inst.BinDir)
	if runtime.GOOS == "windows" {
		fmt.Printf("Run this in PowerShell to add it:\n")
		fmt.Printf("  [Environment]::SetEnvironmentVariable(\"Path\", $env:Path + \";%s\", \"User\")\n", inst.BinDir)
	} else {
		fmt.Printf("Add this to your shell profile:\n")
		fmt.Printf("  export PATH=\"%s:$PATH\"\n", inst.BinDir)
	}
}

func latestVersion(repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

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

func (inst *Installer) downloadAndInstall(tool registry.Tool, version string) error {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	ext := "tar.gz"
	if osName == "windows" {
		ext = "zip"
	}

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s_%s_%s.%s",
		tool.Repo, version, tool.Binary, osName, arch, ext)

	tmpDir, err := os.MkdirTemp("", "brokit-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, fmt.Sprintf("archive.%s", ext))

	// Download
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d (%s)", resp.StatusCode, url)
	}

	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return fmt.Errorf("saving download: %w", err)
	}
	f.Close()

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

	// Install binary
	src := filepath.Join(tmpDir, binaryName)
	dst := filepath.Join(inst.BinDir, binaryName)

	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading extracted binary: %w", err)
	}
	if err := os.WriteFile(dst, data, 0755); err != nil {
		return fmt.Errorf("writing binary: %w", err)
	}

	return nil
}

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
