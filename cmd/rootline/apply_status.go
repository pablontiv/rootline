package main

import (
	"errors"
	"fmt"
	"strings"
)

// errApplyFailure marks every failure the apply commands report through their
// exit status, so a caller can match on it with errors.Is instead of on text.
var errApplyFailure = errors.New("apply failed")

// applyExitError turns an apply run's own outcome into the process exit status.
// It returns nil only when the run carried through everything it accepted;
// otherwise the returned error travels up through RunE, and rootCmd's
// SilenceUsage plus main's os.Exit(1) turn it into a non-zero exit with no
// usage dump.
//
// rolledBackCount is a separate disjunct rather than a subset of errCount, and
// that distinction is the point of this function. A post-validation rollback
// that succeeds populates rolled_back[] and leaves errors[] empty, so a rule
// keyed on errors[] alone still reports success for a run that wrote an invalid
// value and had to take it back. The caller asked for a change that could not
// be made; the tidy cleanup does not make that a success.
//
// Skips and policy rejections are deliberately absent from the rule. The
// command refused those on purpose and said so in the payload, so they exit 0.
func applyExitError(errCount, rolledBackCount int) error {
	if errCount == 0 && rolledBackCount == 0 {
		return nil
	}

	parts := make([]string, 0, 2)
	if errCount > 0 {
		parts = append(parts, fmt.Sprintf("%d error%s", errCount, plural(errCount)))
	}
	if rolledBackCount > 0 {
		parts = append(parts, fmt.Sprintf("%d file%s rolled back", rolledBackCount, plural(rolledBackCount)))
	}

	// Deliberately short: the full payload is already on stdout, and cobra
	// prints this to stderr as "Error: ...". Repeating the payload here would
	// bury the result the caller actually needs to read.
	return fmt.Errorf("%w: %s", errApplyFailure, strings.Join(parts, ", "))
}

// plural returns the "s" that turns a count's noun plural, or "" for exactly one.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
