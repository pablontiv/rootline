package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

// commandsExemptFromBoundaryPreflight lists commands that must run even when no
// governance boundary is declared. They fall into two categories.
//
// Schema-creating — `init` and `schema` write the very .stem files the preflight
// asks for, so gating them on a declared boundary would be circular.
//
// Schema-independent — `completion`, `hooks`, `help` and `migrate` resolve no
// schema at all, and neither does `analyze`: it infers a schema from the
// documents themselves. Gating it would block the one command that can tell an
// ungoverned tree what its schema should be, and it is the entry point of the
// `analyze --incremental` -> `schema apply` loop whose other end is already
// exempt. This is not a general "read-only is safe" rule: `query` and `stats`
// stay governed, because they skip schema resolution voluntarily and would
// otherwise report records governed by .stem files collected from outside an
// undeclared project.
var commandsExemptFromBoundaryPreflight = map[string]bool{
	"init":       true,
	"schema":     true,
	"completion": true,
	"hooks":      true,
	"help":       true,
	"migrate":    true,
	"analyze":    true,
}

// isExemptFromBoundaryPreflight reports whether cmd, or any command it hangs
// under, is exempt.
//
// Cobra reports a subcommand's own name, so `schema propose` answers "propose"
// and `hooks status` answers "status". Matching only the leaf name would
// silently un-exempt every subcommand of an exempt parent — which broke exactly
// the user the migration exists for: an existing project with a .stem but no
// marker could not reach `schema propose`.
func isExemptFromBoundaryPreflight(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if commandsExemptFromBoundaryPreflight[c.Name()] {
			return true
		}
	}
	return false
}

// boundaryPreflight runs once before any schema-governed command.
//
// It is wired at the root rather than at each of the sixteen WalkUp call sites:
// the question it answers — does this project declare where it starts — is a
// property of the tree, not of the individual command.
//
// A chain that terminated at the filesystem root instead of at a `root: true`
// marker has no declared boundary and may have collected .stem files from
// outside the project. On a terminal the user is offered the fix; without one
// the command fails, because prompting into a pipeline that cannot answer hangs
// it until timeout.
func boundaryPreflight(cmd *cobra.Command, args []string) error {
	if isExemptFromBoundaryPreflight(cmd) {
		return nil
	}

	target := "."
	if len(args) > 0 && args[0] != "" {
		target = args[0]
	}

	entries, err := rules.WalkUp(target)
	if err != nil {
		// A tree with no .stem at all is not a broken project. Governed
		// commands reject it in their own context and bootstrap commands are
		// allowed to work on one, so pass that case through.
		if errors.Is(err, rules.ErrNoSchemaFound) {
			return nil
		}

		// Anything else is a real IO or parse failure, and this is the only
		// place that sees every governed command. `query` and `stats` never
		// resolve a schema on their own — query does so only under --sort and
		// neither passes a scope resolver — so without failing here a corrupt
		// .stem is invisible to them: they return records from a tree whose
		// governance is broken and report success.
		return err
	}

	if !rules.ChainHasNoDeclaredBoundary(entries) {
		return nil
	}

	result := rules.AttemptRootMarkerMigration(entries, target, rules.IsTerminal())
	if result.Error != "" {
		return errors.New(result.Error)
	}
	if result.Applied {
		_, _ = fmt.Fprintln(os.Stderr, "Boundary declared. Continuing.")
	}
	return nil
}
