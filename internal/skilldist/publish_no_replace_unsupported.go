//go:build !darwin && !linux && !windows

package skilldist

import (
	"fmt"
	"runtime"
)

func atomicPublishNoReplace(candidate, destination string) error {
	return fmt.Errorf("atomic no-replace publication unsupported on %s/%s for %q -> %q", runtime.GOOS, runtime.GOARCH, candidate, destination)
}
