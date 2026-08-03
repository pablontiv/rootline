package fix

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ContainmentPolicy selects how ContainPath treats an absolute input path.
type ContainmentPolicy int

const (
	// PolicyRejectAbsolute refuses absolute input paths outright. Repair reports
	// carry root-relative record paths by construction, so an absolute path in a
	// report is malformed input rather than a legitimate target.
	PolicyRejectAbsolute ContainmentPolicy = iota

	// PolicyAcceptAbsolute accepts an absolute input path and still requires it to
	// land inside the root. `schema propose` emits absolute .stem targets, so the
	// propose->apply contract depends on absolute paths staying valid.
	PolicyAcceptAbsolute
)

// ContainmentError reports a path that may not be written under the given root.
// It carries the untouched requested path and the resolved root so callers can
// render a message that names both.
type ContainmentError struct {
	Requested string // the path exactly as it appeared in the report
	Root      string // the resolved absolute scan root
	Message   string // the reason, e.g. "escapes root"
}

func (e *ContainmentError) Error() string {
	return fmt.Sprintf("containment violation: path %q %s (root %q)", e.Requested, e.Message, e.Root)
}

// containmentFailure builds the error for a rejected path. Messages are kept
// single-line because they are embedded verbatim in JSON output.
func containmentFailure(requested, root, message string) error {
	return &ContainmentError{Requested: requested, Root: root, Message: message}
}

// ContainPath checks that path resolves to a location inside root and returns
// the validated absolute target.
//
// Containment is lexical: the check operates on cleaned path text and never
// touches the filesystem, so it works for targets that do not exist yet (a
// create_stem write, for instance) where filepath.EvalSymlinks would fail.
//
// Absolute inputs are cleaned and compared to root as-is. They are deliberately
// NOT joined onto root first: filepath.Join(root, "/etc/.stem") silently yields
// root/etc/.stem, which would report an out-of-root target as contained.
//
// Limitation: a symlink inside the target path is not resolved, so a link that
// points outside root escapes at access time. Reports should come from trusted
// producers, and apply commands should run without privileged write access.
func ContainPath(root, path string, policy ContainmentPolicy) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", containmentFailure(path, root, fmt.Sprintf("root cannot be resolved: %v", err))
	}

	var target string
	if filepath.IsAbs(path) {
		if policy == PolicyRejectAbsolute {
			return "", containmentFailure(path, absRoot, "absolute paths are not allowed")
		}
		target = filepath.Clean(path)
	} else {
		target = filepath.Join(absRoot, path)
	}

	// filepath.Rel normalizes drive-letter case and handles UNC shares, and it
	// fails when the two paths live on different volumes — which is itself a
	// containment failure, not something to swallow.
	rel, err := filepath.Rel(absRoot, target)
	if err != nil {
		return "", containmentFailure(path, absRoot, fmt.Sprintf("cannot be resolved relative to root: %v", err))
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", containmentFailure(path, absRoot, "escapes root")
	}

	return target, nil
}
