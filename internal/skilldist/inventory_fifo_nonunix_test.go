//go:build !unix

package skilldist

import "testing"

func makeInventoryTestFIFO(t *testing.T, path string) {
	t.Helper()
	t.Skip("fifo unavailable on this platform")
}
