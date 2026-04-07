package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/anivaryam/brokit/internal/installer"
	"github.com/anivaryam/brokit/internal/registry"
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

	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(removeCmd())
	rootCmd.AddCommand(listCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
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
  brokit install --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			all, _ := cmd.Flags().GetBool("all")
			inst, err := installer.New()
			if err != nil {
				return err
			}

			if all {
				args = nil
				for _, tool := range registry.All() {
					if _, ok := inst.State.Get(tool.Name); !ok {
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
				if err := inst.Install(name); err != nil {
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
			inst, err := installer.New()
			if err != nil {
				return err
			}

			if all {
				args = inst.InstalledNames()
				if len(args) == 0 {
					fmt.Println("No tools installed")
					return nil
				}
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

func removeCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <tool> [tool...]",
		Aliases: []string{"rm", "uninstall"},
		Short:   "Remove installed tools",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inst, err := installer.New()
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
		Short: "List available tools and their install status",
		RunE: func(cmd *cobra.Command, args []string) error {
			inst, err := installer.New()
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TOOL\tDESCRIPTION\tSTATUS\tVERSION")
			for _, tool := range registry.All() {
				status := "not installed"
				ver := "-"
				if t, ok := inst.State.Get(tool.Name); ok {
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
