//go:build windows

package skilldist

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestWindowsAtomicPublishExistErrnoMapsToRestoreConflict(t *testing.T) {
	if windows.MOVEFILE_WRITE_THROUGH == 0 {
		t.Fatal("windows move-file write-through flag is unavailable")
	}
	assertOperationErrorCode(t, normalizeAtomicPublishNoReplaceError(atomicPublishDestinationExistsError(windows.ERROR_ALREADY_EXISTS), `C:\\rootline`, DestinationClaude), ErrRestoreConflict)
	assertOperationErrorCode(t, normalizeAtomicPublishNoReplaceError(atomicPublishDestinationExistsError(windows.ERROR_FILE_EXISTS), `C:\\rootline`, DestinationClaude), ErrRestoreConflict)
}
