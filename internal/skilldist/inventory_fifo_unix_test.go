//go:build unix

package skilldist

import (
	"syscall"
	"testing"
)

func makeInventoryTestFIFO(t *testing.T, path string) {
	t.Helper()
	if err := syscall.Mkfifo(path, 0o644); err != nil {
		t.Skipf("fifo unavailable: %v", err)
	}
}
