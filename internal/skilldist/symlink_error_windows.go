//go:build windows

package skilldist

import (
	"errors"
	"syscall"
)

const windowsErrorPrivilegeNotHeld syscall.Errno = 1314

func platformSymlinkPermissionDenied(err error) bool {
	return errors.Is(err, windowsErrorPrivilegeNotHeld)
}
