//go:build linux

package skilldist

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestLinuxAtomicPublishExistErrnoMapsToRestoreConflict(t *testing.T) {
	if unix.RENAME_NOREPLACE == 0 {
		t.Fatal("linux no-replace rename flag is unavailable")
	}
	assertOperationErrorCode(t, normalizeAtomicPublishNoReplaceError(atomicPublishDestinationExistsError(unix.EEXIST), "/tmp/rootline", DestinationClaude), ErrRestoreConflict)
}
