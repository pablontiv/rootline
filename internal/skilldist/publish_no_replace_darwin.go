//go:build darwin

package skilldist

import (
	"errors"

	"golang.org/x/sys/unix"
)

func atomicPublishNoReplace(candidate, destination string) error {
	err := unix.RenameatxNp(unix.AT_FDCWD, candidate, unix.AT_FDCWD, destination, unix.RENAME_EXCL)
	if err == nil {
		return nil
	}
	if errors.Is(err, unix.EEXIST) || errors.Is(err, unix.ENOTEMPTY) {
		return atomicPublishDestinationExistsError(err)
	}
	return err
}
