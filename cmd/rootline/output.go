package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

// advertisedFormats is the closed set the --output help text promises. Any
// other value is a caller mistake, not a fallback: before this list existed,
// `-o sdlkfj`, `-o JSON` and `-o ""` all exited 0 with whatever the command's
// default branch happened to be.
var advertisedFormats = []string{"json", "jsonl", "csv", "table"}

// formatAgnostic marks a command that never consults --output: it emits human
// text or writes files, and has no envelope a format could shape. Declaring it
// is not the same as leaving it out — the coverage test below demands an entry
// for every command, so "this one ignores --output" is a recorded decision
// rather than an oversight.
var formatAgnostic []string

// commandOutputFormats declares, per command path, which advertised formats
// that command can actually produce.
//
// A central table rather than a switch per command. Twenty-three independent
// equality tests are twenty-three places for the next command to forget its
// default arm, which is how this defect arrived; one table plus
// TestCommandOutputFormats_CoversEveryCommand makes forgetting a test failure.
// It also reaches commands whose bodies still contain the old bare test — the
// preflight rejects the format before their dispatch runs.
var commandOutputFormats = map[string][]string{
	"rootline":                 formatAgnostic,
	"rootline analyze":         {"json", "table"},
	"rootline completion":      formatAgnostic,
	"rootline describe":        {"json", "table"},
	"rootline explain":         {"json", "table"},
	"rootline fix":             {"json", "table"},
	"rootline graph":           {"json", "table"},
	"rootline help":            formatAgnostic,
	"rootline hooks":           formatAgnostic,
	"rootline hooks install":   formatAgnostic,
	"rootline hooks status":    formatAgnostic,
	"rootline hooks uninstall": formatAgnostic,
	"rootline init":            formatAgnostic,
	"rootline migrate":         {"json", "table"},
	"rootline new":             formatAgnostic,
	"rootline query":           {"json", "jsonl", "csv", "table"},
	"rootline repair":          {"json", "table"},
	"rootline repair apply":    {"json", "table"},
	"rootline schema":          formatAgnostic,
	"rootline schema apply":    {"json", "table"},
	"rootline schema propose":  {"json", "table"},
	"rootline set":             formatAgnostic,
	"rootline stats":           {"json", "table"},
	"rootline tree":            {"json", "table"},
	"rootline validate":        {"json", "table"},
}

// isAdvertisedFormat reports whether v is one of the four documented values.
func isAdvertisedFormat(v string) bool {
	return slices.Contains(advertisedFormats, v)
}

// validateOutputFormat rejects an --output the CLI cannot honour, in two steps:
// the global enum first, then what this particular command implements.
//
// An undeclared command passes. That is deliberate: the coverage test already
// fails CI for a missing entry, so the runtime does not need to punish a user
// for a maintainer's omission.
func validateOutputFormat(cmd *cobra.Command) error {
	if !isAdvertisedFormat(outputFormat) {
		return fmt.Errorf("unknown output format %q (use %s)", outputFormat, orList(advertisedFormats))
	}

	supported, declared := commandOutputFormats[cmd.CommandPath()]
	if !declared || supported == nil {
		return nil
	}
	if !slices.Contains(supported, outputFormat) {
		return fmt.Errorf("%s does not support output format %q (use %s)",
			cmd.CommandPath(), outputFormat, orList(supported))
	}
	return nil
}

// effectiveFieldPaths returns the --field values the caller actually asked for.
//
// Cobra can leave empty strings behind when a command is re-executed in the
// same process, and an empty path is not a request for extraction — treating it
// as one would make the flag fire on runs that never passed it.
func effectiveFieldPaths() []string {
	var paths []string
	for _, p := range fieldPath {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}

// validateFieldFlag rejects a --field the command cannot honour.
//
// Extraction reads the JSON envelope, and it was applied inside outputJSON
// only: the table, CSV, JSONL and diagram writers never saw it. So --field
// worked under the default -o json and stopped working the moment a caller
// added -o table, with no diagnostic and no scoping in --help. This follows the
// project's existing precedent for incompatible pairs, where --outbound-type
// without --has-outbound errors cleanly.
func validateFieldFlag(cmd *cobra.Command) error {
	if len(effectiveFieldPaths()) == 0 {
		return nil
	}

	if supported, declared := commandOutputFormats[cmd.CommandPath()]; declared && supported == nil {
		return fmt.Errorf("%s does not support --field: it emits no JSON envelope to extract from", cmd.CommandPath())
	}
	if outputFormat != "json" {
		return fmt.Errorf("--field requires --output json: extraction reads the JSON envelope, and %q has none", outputFormat)
	}
	return nil
}

// orList renders a value list the way the existing `unknown format %q (use dot
// or mermaid)` message does.
func orList(values []string) string {
	switch len(values) {
	case 0:
		return ""
	case 1:
		return values[0]
	}
	return strings.Join(values[:len(values)-1], ", ") + " or " + values[len(values)-1]
}
