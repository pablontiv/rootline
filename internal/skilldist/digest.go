package skilldist

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func DigestTree(root string) (Digest, error) {
	return digestTree(root, false)
}

func digestCanonicalTree(root string) (Digest, error) {
	return digestTree(root, true)
}

func digestTree(root string, rejectSymlinks bool) (Digest, error) {
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		kind := "file"
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			if rejectSymlinks {
				return operationError(ErrSourceNotCanonical, path, "", fmt.Errorf("canonical skill contains a symlink"))
			}
			kind = "symlink"
		case entry.IsDir():
			kind = "dir"
		case !info.Mode().IsRegular():
			return fmt.Errorf("tree contains unsupported file type %s at %s", info.Mode().Type(), path)
		}
		normalized := filepath.ToSlash(rel)
		if _, err := fmt.Fprintf(h, "%d:%s%d:%s:%04o:%d", len(normalized), normalized, len(kind), kind, info.Mode().Perm(), info.Size()); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if kind == "symlink" {
			target, err := rootHandle.Readlink(rel)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(h, "%d:%s", len(target), target)
			return err
		}
		file, err := rootHandle.Open(rel)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(h, file)
		closeErr := file.Close()
		return errors.Join(copyErr, closeErr)
	})
	closeErr := rootHandle.Close()
	if walkErr != nil {
		return "", walkErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return Digest("sha256:" + hex.EncodeToString(h.Sum(nil))), nil
}
