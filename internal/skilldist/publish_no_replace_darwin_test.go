//go:build darwin

package skilldist

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestDarwinAtomicPublishExistErrnoMapsToRestoreConflict(t *testing.T) {
	if unix.RENAME_EXCL == 0 {
		t.Fatal("darwin no-replace rename flag is unavailable")
	}
	assertOperationErrorCode(t, normalizeAtomicPublishNoReplaceError(atomicPublishDestinationExistsError(unix.EEXIST), "/tmp/rootline", DestinationClaude), ErrRestoreConflict)
}
