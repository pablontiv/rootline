package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

// commandsExemptFromBoundaryPreflight lists commands that must run even when no
// governance boundary is declared.
//
// `init` and `schema` create the very .stem files the preflight asks for, so
// gating them on a declared boundary would be circular. The rest do not resolve
// schemas at all.
var commandsExemptFromBoundaryPreflight = map[string]bool{
	"init":       true,
	"schema":     true,
	"completion": true,
	"hooks":      true,
	"help":       true,
	"migrate":    true,
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
	if commandsExemptFromBoundaryPreflight[cmd.Name()] {
		return nil
	}

	target := "."
	if len(args) > 0 && args[0] != "" {
		target = args[0]
	}

	entries, err := rules.WalkUp(target)
	if err != nil {
		// A tree with no .stem at all is ErrNoSchemaFound, which each command
		// reports in its own context. Anything else is a real IO or parse
		// failure and likewise belongs to the command.
		if errors.Is(err, rules.ErrNoSchemaFound) {
			return nil
		}
		return nil
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
