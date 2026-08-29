//go:build windows

package skilldist

import (
	"errors"

	"golang.org/x/sys/windows"
)

func atomicPublishNoReplace(candidate, destination string) error {
	from, err := windows.UTF16PtrFromString(candidate)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	err = windows.MoveFileEx(from, to, windows.MOVEFILE_WRITE_THROUGH)
	if err == nil {
		return nil
	}
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) || errors.Is(err, windows.ERROR_FILE_EXISTS) {
		return atomicPublishDestinationExistsError(err)
	}
	return err
}
