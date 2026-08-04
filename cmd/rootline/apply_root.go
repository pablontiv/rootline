package main

import (
	"fmt"
	"path/filepath"
)

// resolveReportRoot decides which directory a report's document paths resolve
// against, and is deliberately shared by `repair apply` and `schema apply`. It
// takes four plain strings rather than a report struct because the two callers
// hold two unrelated types in two packages; passing primitives is what lets both
// commands run literally the same rule instead of two near-identical ones that
// can drift apart again.
//
// Precedence, highest first:
//
//  1. explicitRoot   — the --root flag, the caller's last word
//  2. reportRoot     — the absolute scan root the producer recorded
//  3. reportPath     — the scan root as the producer spelled it
//  4. reportFilePath — the directory holding the report file
//
// Rung 4 is `repair apply`'s pre-existing behaviour and stays reachable so
// reports written before rungs 2 and 3 existed keep resolving exactly as they
// did. It is also why the rest of the chain exists: rung 4 on its own made a
// report stored in a reports/ directory resolve to paths that do not exist,
// turning the run into a silent no-op.
//
// The result is always absolute. Every rung goes through filepath.Abs, which is
// a no-op on an already-absolute path — including rung 2, absolute only by the
// producer's convention, and rung 4, where a relative --report argument would
// otherwise yield a relative root.
func resolveReportRoot(explicitRoot, reportRoot, reportPath, reportFilePath string) (string, error) {
	var candidate string
	switch {
	case explicitRoot != "":
		candidate = explicitRoot
	case reportRoot != "":
		candidate = reportRoot
	case reportPath != "":
		candidate = reportPath
	case reportFilePath != "":
		candidate = filepath.Dir(reportFilePath)
	default:
		return "", fmt.Errorf("cannot resolve a root: the report carries no path or root, and no --root was given")
	}

	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("resolving root %q: %w", candidate, err)
	}
	return abs, nil
}
