package main

import (
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags; "dev" for local builds.
var version = "dev"

var (
	outputFormat string
	fieldPath    []string
)

var rootCmd = &cobra.Command{
	Use:               "rootline",
	Short:             "File-based database and constraint engine for structured documentation",
	Long:              "Rootline treats the filesystem as a database: directories are tables,\nfiles are records, metadata is extracted from YAML frontmatter,\nand structure is inherited along the directory tree via .stem files.",
	Version:           version,
	PersistentPreRunE: rootPreflight,

	// A usage dump after a runtime failure buries the remediation the error
	// message carries. Flag-parsing errors still show usage, which is where it
	// helps. Errors themselves stay unsilenced: cobra writes them to the
	// command's error stream, which is what callers and tests read.
	SilenceUsage: true,
}

// rootPreflight validates the global flags every command inherits, then hands
// off to the governance boundary check.
//
// Flag validation runs first because it is cheap and unconditional: a caller
// who mistyped --output should read about the flag, not about a missing
// `root: true` marker in a directory the command was never going to scan.
func rootPreflight(cmd *cobra.Command, args []string) error {
	if err := validateOutputFormat(cmd); err != nil {
		return err
	}
	if err := validateFieldFlag(cmd); err != nil {
		return err
	}
	return boundaryPreflight(cmd, args)
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", "output format (json|jsonl|csv|table; not every command supports all four)")
	rootCmd.PersistentFlags().StringSliceVar(&fieldPath, "field", nil, "dot-path field extraction (repeatable; requires --output json, several paths yield a JSON array)")
}

// Execute runs the root command.
//
// Cobra has already reported the error on the command's error stream, so this
// only sets the exit status. Printing here as well would duplicate every
// message.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
