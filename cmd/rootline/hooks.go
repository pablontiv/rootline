package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const hookMarker = "# rootline-managed"
const hookScript = `#!/bin/sh
# rootline-managed
# Pre-commit hook: validate staged markdown files
rootline validate --staged
`

var hooksForce bool

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage git pre-commit hooks",
	Long:  "Install, uninstall, or check status of the rootline git pre-commit hook.",
}

var hooksInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install pre-commit hook",
	RunE:  runHooksInstall,
}

var hooksUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove pre-commit hook",
	RunE:  runHooksUninstall,
}

var hooksStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if hook is installed",
	RunE:  runHooksStatus,
}

func init() {
	hooksInstallCmd.Flags().BoolVar(&hooksForce, "force", false, "overwrite existing non-rootline hook")
	hooksCmd.AddCommand(hooksInstallCmd)
	hooksCmd.AddCommand(hooksUninstallCmd)
	hooksCmd.AddCommand(hooksStatusCmd)
	rootCmd.AddCommand(hooksCmd)
}

func gitHooksDir() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--git-dir").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	gitDir := strings.TrimSpace(string(out))
	return filepath.Join(gitDir, "hooks"), nil
}

func preCommitPath() (string, error) {
	dir, err := gitHooksDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pre-commit"), nil
}

func isRootlineHook(path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), hookMarker)
}

func runHooksInstall(cmd *cobra.Command, args []string) error {
	hookPath, err := preCommitPath()
	if err != nil {
		return err
	}

	// Check existing hook
	if _, err := os.Stat(hookPath); err == nil {
		if !hooksForce && !isRootlineHook(hookPath) {
			return fmt.Errorf("pre-commit hook already exists (not rootline-managed); use --force to overwrite")
		}
	}

	// Ensure hooks directory exists
	if err := os.MkdirAll(filepath.Dir(hookPath), 0755); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}

	if err := os.WriteFile(hookPath, []byte(hookScript), 0755); err != nil {
		return fmt.Errorf("writing hook: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Installed pre-commit hook")
	return nil
}

func runHooksUninstall(cmd *cobra.Command, args []string) error {
	hookPath, err := preCommitPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No pre-commit hook installed")
		return nil
	}

	if !isRootlineHook(hookPath) {
		return fmt.Errorf("pre-commit hook exists but is not rootline-managed; refusing to remove")
	}

	if err := os.Remove(hookPath); err != nil {
		return fmt.Errorf("removing hook: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Removed pre-commit hook")
	return nil
}

func runHooksStatus(cmd *cobra.Command, args []string) error {
	hookPath, err := preCommitPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "not installed")
		return nil
	}

	if isRootlineHook(hookPath) {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "installed")
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "not installed (non-rootline hook exists)")
	}
	return nil
}
