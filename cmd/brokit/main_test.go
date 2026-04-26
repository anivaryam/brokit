package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLI_HelpFlag(t *testing.T) {
	out, err := execCLI("--help")
	require.NoError(t, err, "help flag should succeed")
	assert.Contains(t, out, "Package manager for anivaryam's dev tools")
}

func TestCLI_VersionFlag(t *testing.T) {
	out, err := execCLI("--version")
	require.NoError(t, err, "version flag should succeed")
	assert.Contains(t, out, "brokit")
}

func TestCLI_InstallCommand_Help(t *testing.T) {
	out, err := execCLI("install", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "Install tools from GitHub releases")
}

func TestCLI_UpdateCommand_Help(t *testing.T) {
	out, err := execCLI("update", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "Check for new versions and update installed tools")
}

func TestCLI_RemoveCommand_Help(t *testing.T) {
	out, err := execCLI("remove", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "Remove installed tools")
}

func TestCLI_ListCommand_Help(t *testing.T) {
	out, err := execCLI("list", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "List available tools")
}

func TestCLI_ListCommand_Output(t *testing.T) {
	out, err := execCLI("list")
	require.NoError(t, err)
	assert.Contains(t, out, "TOOL")
	assert.Contains(t, out, "DESCRIPTION")
	assert.Contains(t, out, "STATUS")
	assert.Contains(t, out, "VERSION")
}

func TestCLI_Install_NoArgs(t *testing.T) {
	_, err := execCLI("install")
	require.Error(t, err, "install without args should fail")
}

func TestCLI_Update_NoArgs(t *testing.T) {
	_, err := execCLI("update")
	require.Error(t, err, "update without args should fail")
}

func TestCLI_Remove_NoArgs(t *testing.T) {
	_, err := execCLI("remove")
	require.Error(t, err, "remove without args should fail")
}

func TestCLI_Aliases(t *testing.T) {
	tests := []struct {
		alias string
		cmd   string
	}{
		{"i", "install"},
		{"u", "update"},
		{"ls", "list"},
		{"rm", "remove"},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			out, err := execCLI(tt.alias, "--help")
			require.NoError(t, err, "alias %s should work", tt.alias)
			assert.Contains(t, out, tt.cmd, "alias %s should show %s help", tt.alias, tt.cmd)
		})
	}
}

func TestCLI_VerboseFlag(t *testing.T) {
	out, err := execCLI("-v", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "TOOL")
}

func TestCLI_UnknownCommand(t *testing.T) {
	_, err := execCLI("unknown-command")
	require.Error(t, err, "unknown command should fail")
}

func execCLI(args ...string) (string, error) {
	binPath := filepath.Join(os.Getenv("PWD"), "bin", "brokit")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		binPath = "brokit"
	}
	cmd := exec.Command(binPath, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
