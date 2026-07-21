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
	PersistentPreRunE: boundaryPreflight,

	// A usage dump after a runtime failure buries the remediation the error
	// message carries. Flag-parsing errors still show usage, which is where it
	// helps. Errors themselves stay unsilenced: cobra writes them to the
	// command's error stream, which is what callers and tests read.
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", "output format (json|jsonl|csv|table)")
	rootCmd.PersistentFlags().StringSliceVar(&fieldPath, "field", nil, "dot-path field extraction (repeatable)")
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
