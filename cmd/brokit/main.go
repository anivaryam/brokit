package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/anivaryam/brokit/internal/downloader"
	"github.com/anivaryam/brokit/internal/installer"
	"github.com/anivaryam/brokit/internal/registry"
	"github.com/anivaryam/brokit/internal/state"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:           "brokit",
		Short:         "Package manager for anivaryam's dev tools",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Show detailed output")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Only show errors")

	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(removeCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(selfUpdateCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func newInstaller(cmd *cobra.Command) (*installer.Installer, *downloader.Downloader, error) {
	verbose, _ := cmd.Flags().GetBool("verbose")
	quiet, _ := cmd.Flags().GetBool("quiet")

	sp, err := stateFilePath()
	if err != nil {
		return nil, nil, err
	}
	s, err := state.Load(sp)
	if err != nil {
		return nil, nil, err
	}

	const userAgent = "brokit"
	dlClient := downloader.NewClient(userAgent)
	dl := downloader.NewDownloader(dlClient, userAgent)

	reg := &registryAdapter{}
	st := &stateAdapter{State: s, statePath: sp}

	inst := installer.NewInstaller(reg, st, dl)

	if quiet {
		inst.LogLevel = installer.LogQuiet
	} else if verbose {
		inst.LogLevel = installer.LogVerbose
	}
	return inst, dl, nil
}

type registryAdapter struct{}

func (r *registryAdapter) Get(name string) (registry.Tool, bool) {
	return registry.Get(name)
}

func (r *registryAdapter) All() []registry.Tool {
	return registry.All()
}

func (r *registryAdapter) Names() []string {
	return registry.Names()
}

type stateAdapter struct {
	*state.State
	statePath string
}

func (s *stateAdapter) Set(t state.InstalledTool) error {
	s.State.Set(t.Name, t.Version)
	return nil
}

func (s *stateAdapter) Remove(name string) error {
	s.State.Remove(name)
	return nil
}

func (s *stateAdapter) List() []state.InstalledTool {
	list := make([]state.InstalledTool, 0, len(s.State.Installed))
	for _, t := range s.State.Installed {
		list = append(list, t)
	}
	return list
}

func stateFilePath() (string, error) {
	_configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(_configDir, "brokit", "state.json"), nil
}

func installCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install <tool> [tool...]",
		Aliases: []string{"i"},
		Short:   "Install one or more tools",
		Long: `Install tools from GitHub releases.

Examples:
  brokit install tunnel
  brokit install merge-port proc-compose tunnel
  brokit install --all
  brokit install --force tunnel`,
		RunE: func(cmd *cobra.Command, args []string) error {
			all, _ := cmd.Flags().GetBool("all")
			force, _ := cmd.Flags().GetBool("force")
			inst, _, err := newInstaller(cmd)
			if err != nil {
				return err
			}

			if all {
				args = nil
				for _, tool := range registry.All() {
					if force {
						args = append(args, tool.Name)
					} else if _, ok := inst.State.Get(tool.Name); !ok {
						args = append(args, tool.Name)
					}
				}
				if len(args) == 0 {
					fmt.Println("All tools are already installed")
					return nil
				}
			}

			if len(args) == 0 {
				return fmt.Errorf("specify tools to install or use --all\navailable: %s", strings.Join(registry.Names(), ", "))
			}

			var errs []string
			for _, name := range args {
				if err := inst.Install(name, force); err != nil {
					errs = append(errs, err.Error())
				}
			}
			if len(errs) > 0 {
				return fmt.Errorf("%s", strings.Join(errs, "\n"))
			}
			return nil
		},
	}
	cmd.Flags().Bool("all", false, "Install all available tools")
	cmd.Flags().BoolP("force", "f", false, "Force reinstall even if already installed")
	return cmd
}

func updateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update [tool...]",
		Aliases: []string{"u", "up"},
		Short:   "Update installed tools",
		Long: `Check for new versions and update installed tools.

Examples:
  brokit update tunnel
  brokit update --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			all, _ := cmd.Flags().GetBool("all")
			inst, dl, err := newInstaller(cmd)
			if err != nil {
				return err
			}

			if all {
				names := inst.InstalledNames()
				if len(names) == 0 {
					fmt.Println("No tools installed")
					return nil
				}
				return parallelUpdate(inst, dl, names)
			}

			if len(args) == 0 {
				return fmt.Errorf("specify tools to update or use --all")
			}

			var errs []string
			for _, name := range args {
				if err := inst.Update(name); err != nil {
					errs = append(errs, err.Error())
				}
			}
			if len(errs) > 0 {
				return fmt.Errorf("%s", strings.Join(errs, "\n"))
			}
			return nil
		},
	}
	cmd.Flags().Bool("all", false, "Update all installed tools")
	return cmd
}

// parallelUpdate fetches all versions concurrently, then applies updates sequentially.
func parallelUpdate(inst *installer.Installer, dl *downloader.Downloader, names []string) error {
	type result struct {
		name    string
		version string
		err     error
	}

	results := make([]result, len(names))
	var wg sync.WaitGroup
	for i, name := range names {
		wg.Add(1)
		go func(idx int, n string) {
			defer wg.Done()
			tool, ok := registry.Get(n)
			if !ok {
				results[idx] = result{name: n, err: fmt.Errorf("unknown tool: %s", n)}
				return
			}
			ver, err := dl.Latest(tool.Repo)
			results[idx] = result{name: n, version: ver, err: err}
		}(i, name)
	}
	wg.Wait()

	var errs []string
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", r.name, r.err))
			continue
		}
		if err := inst.UpdateTo(r.name, r.version); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return nil
}

func removeCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <tool> [tool...]",
		Aliases: []string{"rm", "uninstall"},
		Short:   "Remove installed tools",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inst, _, err := newInstaller(cmd)
			if err != nil {
				return err
			}

			var errs []string
			for _, name := range args {
				if err := inst.Remove(name); err != nil {
					errs = append(errs, err.Error())
				}
			}
			if len(errs) > 0 {
				return fmt.Errorf("%s", strings.Join(errs, "\n"))
			}
			return nil
		},
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available tools and their install status",
		RunE: func(cmd *cobra.Command, args []string) error {
			inst, _, err := newInstaller(cmd)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TOOL\tDESCRIPTION\tSTATUS\tVERSION")
			for _, tool := range registry.All() {
				status := "not installed"
				ver := "-"
				if t, ok := inst.GetTool(tool.Name); ok {
					status = "installed"
					ver = t.Version
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", tool.Name, tool.Description, status, ver)
			}
			w.Flush()
			return nil
		},
	}
}

func selfUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "self-update",
		Short: "Update brokit itself to the latest version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			inst, _, err := newInstaller(cmd)
			if err != nil {
				return err
			}
			return inst.SelfUpdate(version)
		},
	}
}
