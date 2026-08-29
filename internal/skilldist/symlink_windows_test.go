//go:build windows

package skilldist

import (
	"syscall"
	"testing"
)

func TestWindowsPrivilegeNotHeldMapsToSymlinkPermissionDenied(t *testing.T) {
	err := normalizeSymlinkCreationError(syscall.Errno(1314), "/tmp/rootline", string(DestinationClaude))
	if err == nil || err.Code != ErrSymlinkPermission {
		t.Fatalf("normalized error = %#v, want symlink permission", err)
	}
}
