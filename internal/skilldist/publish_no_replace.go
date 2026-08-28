package skilldist

import (
	"errors"
	"fmt"
)

var errAtomicPublishDestinationExists = errors.New("atomic no-replace destination exists")

func atomicPublishDestinationExistsError(err error) error {
	if err == nil {
		return errAtomicPublishDestinationExists
	}
	return fmt.Errorf("%w: %w", errAtomicPublishDestinationExists, err)
}

func normalizeAtomicPublishNoReplaceError(err error, destination string, id DestinationID) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errAtomicPublishDestinationExists) {
		return operationError(ErrRestoreConflict, destination, string(id), fmt.Errorf("destination already exists at publication: %w", err))
	}
	return err
}
