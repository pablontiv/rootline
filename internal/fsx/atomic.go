// Package fsx provides shared filesystem operations with explicit durability
// guarantees.
package fsx

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

// StatInRoot stats target using the same rooted path semantics as
// WriteFileAtomicInRoot. It accepts only lexical descendants of root, then lets
// os.Root reject symlink traversal that would escape the anchored root.
func StatInRoot(root, target string) (fs.FileInfo, error) {
	rel, err := rootRelativePath(root, target)
	if err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("opening root %s: %w", root, err)
	}
	defer func() { _ = r.Close() }()
	info, err := r.Stat(rel)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", target, err)
	}
	return info, nil
}

// WriteFileAtomicInRoot atomically replaces target while confining every
// filesystem operation to root. The target must be a lexical descendant of root,
// and any symlink traversal in its parent path that would escape root is
// rejected by os.Root before a staging file is created.
func WriteFileAtomicInRoot(root, target string, content []byte, perm fs.FileMode) error {
	rel, err := rootRelativePath(root, target)
	if err != nil {
		return err
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("opening root %s: %w", root, err)
	}
	defer func() { _ = r.Close() }()
	return writeFileAtomicRoot(r, rel, perm, func(dst io.Writer) error {
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

func writeFileAtomicRoot(root *os.Root, relTarget string, perm fs.FileMode, write func(io.Writer) error) error {
	relTarget = filepath.Clean(relTarget)
	relDir := filepath.Dir(relTarget)
	base := filepath.Base(relTarget)

	dirRoot, err := root.OpenRoot(relDir)
	if err != nil {
		return fmt.Errorf("opening rooted parent %s: %w", relDir, err)
	}
	defer func() { _ = dirRoot.Close() }()

	if info, err := dirRoot.Stat(base); err == nil {
		perm = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stating %s: %w", relTarget, err)
	}

	tmp, tmpName, err := createRootTemp(dirRoot)
	if err != nil {
		return fmt.Errorf("staging a rooted write for %s: %w", relTarget, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tmp.Close()
			_ = dirRoot.Remove(tmpName)
		}
	}()

	if err := write(tmp); err != nil {
		return fmt.Errorf("writing staged content for %s: %w", relTarget, err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("flushing staged content for %s: %w", relTarget, err)
	}
	if err := tmp.Chmod(perm); err != nil {
		return fmt.Errorf("setting mode on staged content for %s: %w", relTarget, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing staged content for %s: %w", relTarget, err)
	}
	if err := dirRoot.Rename(tmpName, base); err != nil {
		return fmt.Errorf("replacing %s: %w", relTarget, err)
	}
	committed = true
	return nil
}

func createRootTemp(root *os.Root) (*os.File, string, error) {
	var lastErr error
	for range 100 {
		name, err := randomRootTempName()
		if err != nil {
			return nil, "", err
		}
		file, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			return file, name, nil
		}
		if !os.IsExist(err) {
			return nil, "", err
		}
		lastErr = err
	}
	return nil, "", fmt.Errorf("creating unique temporary file: %w", lastErr)
}

func randomRootTempName() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return tempFilePrefix + hex.EncodeToString(b[:]) + ".tmp", nil
}

func rootRelativePath(root, target string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolving root %s: %w", root, err)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolving target %s: %w", target, err)
	}
	absRoot = filepath.Clean(absRoot)
	absTarget = filepath.Clean(absTarget)
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return "", fmt.Errorf("resolving %s relative to %s: %w", absTarget, absRoot, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("target %s escapes root %s", absTarget, absRoot)
	}
	return rel, nil
}
