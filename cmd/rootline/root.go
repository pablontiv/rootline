package main

import (
	"fmt"
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
	Use:     "rootline",
	Short:   "File-based database and constraint engine for structured documentation",
	Long:    "Rootline treats the filesystem as a database: directories are tables,\nfiles are records, metadata is extracted from YAML frontmatter,\nand structure is inherited along the directory tree via .stem files.",
	Version: version,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", "output format (json|jsonl|csv|table)")
	rootCmd.PersistentFlags().StringSliceVar(&fieldPath, "field", nil, "dot-path field extraction (repeatable)")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
