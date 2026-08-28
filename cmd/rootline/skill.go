package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pablontiv/rootline/internal/skilldist"
	"github.com/spf13/cobra"
)

var skillSource string
var skillApproval string
var skillReceipt string

var skillServiceFactory = newSkillService

type SkillEnvelope struct {
	Version      int                          `json:"version"`
	Kind         string                       `json:"kind"`
	Complete     bool                         `json:"complete"`
	Source       *skilldist.Source            `json:"source,omitempty"`
	PlanDigest   skilldist.Digest             `json:"plan_digest,omitempty"`
	Destinations []skilldist.DestinationState `json:"destinations"`
	Backups      []skilldist.Backup           `json:"backups"`
	Receipt      *skilldist.Receipt           `json:"receipt,omitempty"`
	ReceiptDrift bool                         `json:"receipt_drift"`
	Errors       []skilldist.OperationError   `json:"errors"`
}

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage the distributed rootline skill lifecycle",
}

var skillInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Plan or install the managed rootline skill",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		source, err := resolveSkillSource(skillSource)
		if err != nil {
			return err
		}
		service, err := skillServiceFactory()
		if err != nil {
			return err
		}
		result := service.Install(commandContext(cmd), source, skilldist.Digest(skillApproval))
		return outputSkillResult(cmd, "rootline/skill-install", result)
	},
}

var skillStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report managed rootline skill status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		source, err := resolveSkillSource(skillSource)
		if err != nil {
			return err
		}
		service, err := skillServiceFactory()
		if err != nil {
			return err
		}
		result := service.Status(commandContext(cmd), source)
		return outputSkillResult(cmd, "rootline/skill-status", result)
	},
}

var skillUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Plan or uninstall the managed rootline skill",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := skillServiceFactory()
		if err != nil {
			return err
		}
		result := service.Uninstall(commandContext(cmd), skilldist.Digest(skillApproval))
		return outputSkillResult(cmd, "rootline/skill-uninstall", result)
	},
}

var skillRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Plan or restore from a rootline skill receipt",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := skillServiceFactory()
		if err != nil {
			return err
		}
		result := service.Restore(commandContext(cmd), skillReceipt, skilldist.Digest(skillApproval))
		return outputSkillResult(cmd, "rootline/skill-restore", result)
	},
}

func init() {
	skillInstallCmd.Flags().StringVar(&skillSource, "source", "", "source Git repository (default: current working directory)")
	skillInstallCmd.Flags().StringVar(&skillApproval, "approve", "", "approved plan digest to apply")

	skillStatusCmd.Flags().StringVar(&skillSource, "source", "", "source Git repository (default: current working directory)")

	skillUninstallCmd.Flags().StringVar(&skillApproval, "approve", "", "approved plan digest to apply")

	skillRestoreCmd.Flags().StringVar(&skillApproval, "approve", "", "approved plan digest to apply")
	skillRestoreCmd.Flags().StringVar(&skillReceipt, "receipt", "", "receipt ID to restore")
	_ = skillRestoreCmd.MarkFlagRequired("receipt")

	skillCmd.AddCommand(skillInstallCmd, skillStatusCmd, skillUninstallCmd, skillRestoreCmd)
	rootCmd.AddCommand(skillCmd)
}

func commandContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

func resolveSkillSource(source string) (string, error) {
	if source != "" {
		return source, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve current Git repository: %w", err)
	}
	return cwd, nil
}

func newSkillService() (*skilldist.Service, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	stateRoot, err := defaultSkillStateRoot(home)
	if err != nil {
		return nil, err
	}
	return skilldist.New(skilldist.Options{HomeDir: home, StateDir: stateRoot})
}

func defaultSkillStateRoot(home string) (string, error) {
	if runtime.GOOS == "windows" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolve config directory: %w", err)
		}
		return filepath.Clean(configDir), nil
	}
	if xdgStateHome := os.Getenv("XDG_STATE_HOME"); xdgStateHome != "" {
		return filepath.Clean(xdgStateHome), nil
	}
	return filepath.Join(home, ".local", "state"), nil
}

func outputSkillResult(cmd *cobra.Command, kind string, result skilldist.Result) error {
	return outputJSON(cmd, skillEnvelopeFromResult(kind, result), result.Failed())
}

func skillEnvelopeFromResult(kind string, result skilldist.Result) SkillEnvelope {
	envelope := SkillEnvelope{
		Version:      1,
		Kind:         kind,
		Complete:     result.Complete,
		Source:       result.Source,
		Destinations: append([]skilldist.DestinationState{}, result.Destinations...),
		Backups:      append([]skilldist.Backup{}, result.Backups...),
		Receipt:      result.Receipt,
		ReceiptDrift: result.ReceiptDrift,
		Errors:       append([]skilldist.OperationError{}, result.Errors...),
	}
	if result.Plan != nil {
		envelope.PlanDigest = result.Plan.Digest
	}
	return envelope
}
