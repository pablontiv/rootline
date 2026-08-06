package fix

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// tempFilePrefix names the staging files WriteFileAtomic creates. It is a
// recognisable, dotted prefix so that a file surviving a hard kill is obviously
// rootline's and obviously not a document.
const tempFilePrefix = ".rootline-"

// WriteFileAtomic writes content to target so that target is never observed
// half-written. It stages the bytes in a sibling temp file and renames that over
// the target, which on a single filesystem is one atomic operation.
//
// os.WriteFile cannot offer this: it truncates the target and then writes, so a
// process that dies in between leaves a document truncated or partial — and the
// repair commands rewrite governed documents whose schema then rejects them.
//
// The staging file is created in the target's OWN directory rather than the
// system temp directory, because os.Rename across filesystems fails. Every
// failure path removes the staging file before returning, so a failed write
// leaves the directory exactly as it found it.
//
// perm is applied to the target explicitly. os.CreateTemp makes its file 0600,
// so without this the mode of the staging file would silently become the mode of
// the document.
//
// This makes each file atomic. It does NOT make a run atomic: see ApplyRepair
// for the run-level contract, which is best-effort with honest reporting.
func WriteFileAtomic(target string, content []byte, perm fs.FileMode) error {
	dir := filepath.Dir(target)

	tmp, err := os.CreateTemp(dir, tempFilePrefix+"*.tmp")
	if err != nil {
		return fmt.Errorf("staging a write for %s: %w", target, err)
	}
	tmpName := tmp.Name()

	// From here on every exit must clean up the staging file. cleanup is safe to
	// call after a successful rename: the name no longer exists and Remove's
	// error is deliberately ignored.
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("writing staged content for %s: %w", target, err)
	}

	// Flush before the rename. Without it the rename can be durable while the
	// contents are not, which is the same half-written file by another route.
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("flushing staged content for %s: %w", target, err)
	}

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("setting mode on staged content for %s: %w", target, err)
	}

	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("closing staged content for %s: %w", target, err)
	}

	if err := os.Rename(tmpName, target); err != nil {
		cleanup()
		return fmt.Errorf("replacing %s: %w", target, err)
	}

	return nil
}
