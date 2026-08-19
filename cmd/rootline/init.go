package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/fsx"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/pablontiv/rootline/internal/stemyaml"
	"github.com/pablontiv/rootline/internal/templates"
	"github.com/spf13/cobra"
)

var (
	initDryRun   bool
	initForce    bool
	initTemplate string
)

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Generate .stem from existing documents",
	Long:  "Scan markdown files and infer a .stem schema from frontmatter patterns.\nUse --dry-run to preview without writing.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initDryRun, "dry-run", false, "print to stdout without writing file")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing .stem file")
	initCmd.Flags().StringVar(&initTemplate, "template", "", "fetch .stem from remote repo (owner/repo[@tag])")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	target := "."
	if len(args) > 0 {
		target = args[0]
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if initTemplate != "" {
		files, err := templates.FetchTemplate(initTemplate, absTarget, initForce, initDryRun)
		if err != nil {
			return err
		}
		if initDryRun {
			for _, f := range files {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would copy: %s\n", f)
			}
			return nil
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Installed %d .stem file(s) from %s\n", len(files), initTemplate)
		return nil
	}

	// Scan for records with AST parsing enabled so section patterns can be detected.
	reg := extract.NewASTRegistry()
	records, err := index.Scan(ctx, absTarget, reg)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", target, err)
	}

	if len(records) == 0 {
		return fmt.Errorf("no markdown files found in %s", target)
	}

	// Try hierarchical detection first.
	hierarchy := infer.AnalyzeHierarchy(records, absTarget)
	if hierarchy.Detected {
		return runInitHierarchical(cmd, absTarget, target, hierarchy, records)
	}

	// Fall back to flat mode.
	return runInitFlat(cmd, absTarget, target, records)
}

func runInitFlat(cmd *cobra.Command, absTarget, target string, records []*extract.Record) error {
	stemPath := filepath.Join(absTarget, ".stem")

	// Check for existing .stem
	if !initForce && !initDryRun {
		if _, err := os.Stat(stemPath); err == nil {
			return fmt.Errorf(".stem already exists in %s (use --force to overwrite)", target)
		}
	}

	// Generate flat schema using the new schema generation service
	opts := infer.InferOptions{
		SectionThreshold:  0.80, // prescriptive for init
		IncludeStructural: true,
	}
	stemFile, err := infer.GenerateFlatSchema(cmd.Context(), absTarget, records, opts)
	if err != nil {
		return fmt.Errorf("generating flat schema: %w", err)
	}

	// Mark as root to establish schema boundary
	stemFile.Root = true

	// Analyze frontmatter for warning about mixed content
	analyzed := infer.Analyze(records)
	if analyzed.TotalFiles > 0 && analyzed.FilesWithout > 0 {
		ratio := float64(analyzed.FilesWithout) / float64(analyzed.TotalFiles)
		if ratio > 0.2 {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
				"Warning: %d of %d files have no frontmatter. Consider running init on a more specific subdirectory.\n",
				analyzed.FilesWithout, analyzed.TotalFiles)
		}
	}

	// Generate YAML from StemFile
	yaml, err := stemFileToYAML(stemFile, absTarget)
	if err != nil {
		return fmt.Errorf("serializing flat schema: %w", err)
	}

	if initDryRun {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), yaml)
		return nil
	}

	if err := fsx.WriteFileAtomic(stemPath, []byte(yaml), 0o644); err != nil {
		return fmt.Errorf("writing .stem: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %s (%d fields inferred from %d files)\n",
		stemPath, len(stemFile.Schema), len(records))
	return nil
}

func runInitHierarchical(cmd *cobra.Command, absTarget, target string, hierarchy *infer.HierarchyResult, records []*extract.Record) error {
	// Generate hierarchical schema using the new schema generation service
	opts := infer.InferOptions{
		SectionThreshold:  0.80,
		IncludeStructural: true,
	}
	stemMap, err := infer.GenerateHierarchicalSchema(cmd.Context(), absTarget, records, opts)
	if err != nil {
		return fmt.Errorf("generating hierarchical schema: %w", err)
	}

	// Mark only the root-level .stem as the boundary (children inherit through merge)
	if rootStem := stemMap["."]; rootStem != nil {
		rootStem.Root = true
	}

	// Convert StemFiles to file content for writing
	stemFiles := make([]stemFile, 0, len(stemMap))
	for _, rootStem := range stemMap {
		yaml, err := stemFileToYAML(rootStem, absTarget)
		if err != nil {
			return fmt.Errorf("serializing hierarchical schema: %w", err)
		}
		stemFiles = append(stemFiles, stemFile{
			path:    filepath.Join(absTarget, ".stem"),
			content: yaml,
		})
	}

	// Emit notes for auto-generated aggregates
	if len(stemMap) > 0 {
		rootStem := stemMap["."]
		if rootStem != nil && len(rootStem.Aggregate) > 0 {
			for name := range rootStem.Aggregate {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Note: auto-generated aggregate for '%s'\n", name)
			}
		}
	}

	// Check for existing .stem files.
	if !initForce && !initDryRun {
		for _, sf := range stemFiles {
			if _, err := os.Stat(sf.path); err == nil {
				return fmt.Errorf(".stem already exists at %s (use --force to overwrite)", sf.path)
			}
		}
	}

	if initDryRun {
		for i, sf := range stemFiles {
			if i > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
			}
			relPath, _ := filepath.Rel(absTarget, sf.path)
			if relPath == "" {
				relPath = sf.path
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "# --- %s ---\n", relPath)
			_, _ = fmt.Fprint(cmd.OutOrStdout(), sf.content)
		}
		return nil
	}

	// Write all .stem files.
	for _, sf := range stemFiles {
		if err := fsx.WriteFileAtomic(sf.path, []byte(sf.content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", sf.path, err)
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created .stem with %d levels (detected from %d files)\n",
		len(hierarchy.Levels), len(records))
	return nil
}

type stemFile struct {
	path    string
	content string
}

// stemFileToYAML converts a StemFile to YAML string representation.
func stemFileToYAML(stem *rules.StemFile, scanRoot string) (string, error) {
	var b strings.Builder
	b.WriteString("version: 2\n")
	if stem.Root {
		b.WriteString("root: true\n")
	}
	b.WriteString("scope:\n  match: \"*.md\"\nschema:\n")

	keys := make([]string, 0, len(stem.Schema))
	for k := range stem.Schema {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		if err := stemyaml.AppendSchemaField(&b, name, stem.Schema[name]); err != nil {
			return "", fmt.Errorf("serializing field %q: %w", name, err)
		}
	}

	// Emit aggregate section if present
	if len(stem.Aggregate) > 0 {
		aggKeys := make([]string, 0, len(stem.Aggregate))
		for k := range stem.Aggregate {
			aggKeys = append(aggKeys, k)
		}
		sort.Strings(aggKeys)

		b.WriteString("aggregate:\n")
		for _, name := range aggKeys {
			val := stem.Aggregate[name]
			fmt.Fprintf(&b, "  %s: |\n", name)
			for _, line := range strings.Split(fmt.Sprintf("%v", val), "\n") {
				fmt.Fprintf(&b, "    %s\n", line)
			}
		}
	}

	// Append structural inference section if present
	if !stem.Structural.IsEmpty() {
		b.WriteString("structural:\n  subdirs:\n")
		if stem.Structural.Subdirs.RequireIndex != "" {
			fmt.Fprintf(&b, "    require_index: %s\n", stem.Structural.Subdirs.RequireIndex)
		}
		if stem.Structural.Subdirs.MinChildren > 0 {
			fmt.Fprintf(&b, "    min_children: %d\n", stem.Structural.Subdirs.MinChildren)
		}
		if stem.Structural.Subdirs.MaxChildren > 0 {
			fmt.Fprintf(&b, "    max_children: %d\n", stem.Structural.Subdirs.MaxChildren)
		}
	}

	out := b.String()
	if _, err := rules.ParseStem(filepath.Join(scanRoot, ".stem"), []byte(out)); err != nil {
		return "", err
	}
	return out, nil
}
