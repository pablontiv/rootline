//go:build !windows

package skilldist

func platformSymlinkPermissionDenied(error) bool { return false }
