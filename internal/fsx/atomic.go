// Package fsx provides shared filesystem operations with explicit durability
// guarantees.
package fsx

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

const tempFilePrefix = ".rootline-"

// WriteFileAtomic replaces target only after all content has been written to a
// sibling staging file. The caller-provided mode is applied before replacement.
func WriteFileAtomic(target string, content []byte, perm fs.FileMode) error {
	return writeFileAtomic(target, perm, func(dst io.Writer) error {
		_, err := dst.Write(content)
		return err
	})
}

// CopyFileAtomic streams src into a sibling staging file and replaces target
// only after the copy is complete. The source permission bits are preserved.
func CopyFileAtomic(src, target string) (retErr error) {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s: %w", src, err)
	}
	defer func() {
		if err := in.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("closing %s: %w", src, err)
		}
	}()

	info, err := in.Stat()
	if err != nil {
		return fmt.Errorf("stating %s: %w", src, err)
	}

	return writeFileAtomic(target, info.Mode().Perm(), func(dst io.Writer) error {
		_, err := io.Copy(dst, in)
		return err
	})
}

func writeFileAtomic(target string, perm fs.FileMode, write func(io.Writer) error) error {
	dir := filepath.Dir(target)
	if info, err := os.Stat(target); err == nil {
		perm = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stating %s: %w", target, err)
	}

	tmp, err := os.CreateTemp(dir, tempFilePrefix+"*.tmp")
	if err != nil {
		return fmt.Errorf("staging a write for %s: %w", target, err)
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if err := write(tmp); err != nil {
		return fmt.Errorf("writing staged content for %s: %w", target, err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("flushing staged content for %s: %w", target, err)
	}
	if err := tmp.Chmod(perm); err != nil {
		return fmt.Errorf("setting mode on staged content for %s: %w", target, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing staged content for %s: %w", target, err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("replacing %s: %w", target, err)
	}
	committed = true
	return nil
}
