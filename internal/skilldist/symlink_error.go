package skilldist

import (
	"errors"
	"os"
	"syscall"
)

func normalizeSymlinkCreationError(err error, path, destination string) *OperationError {
	if err == nil {
		return nil
	}
	if errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) || platformSymlinkPermissionDenied(err) {
		return operationError(ErrSymlinkPermission, path, destination, err)
	}
	if errors.Is(err, os.ErrExist) {
		return operationError(ErrRestoreConflict, path, destination, err)
	}
	return operationError(ErrVerificationFailed, path, destination, err)
}
