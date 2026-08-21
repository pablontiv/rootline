package fsx

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type AtomicTarget struct {
	logicalPath  string
	physicalPath string
	parentPath   string
	parent       *os.Root
	name         string
	closeOnce    sync.Once
	closeErr     error
}

func ResolveAtomicTarget(allowedRoot, logicalTarget string) (*AtomicTarget, error) {
	logicalTarget, err := filepath.Abs(logicalTarget)
	if err != nil {
		return nil, fmt.Errorf("resolving target %s: %w", logicalTarget, err)
	}
	physicalRoot, err := filepath.EvalSymlinks(allowedRoot)
	if err != nil {
		return nil, fmt.Errorf("resolving allowed root %s: %w", allowedRoot, err)
	}
	physicalRoot, err = filepath.Abs(physicalRoot)
	if err != nil {
		return nil, fmt.Errorf("normalizing allowed root %s: %w", allowedRoot, err)
	}
	physicalParent, err := filepath.EvalSymlinks(filepath.Dir(logicalTarget))
	if err != nil {
		return nil, fmt.Errorf("resolving parent for %s: %w", logicalTarget, err)
	}
	physicalParent, err = filepath.Abs(physicalParent)
	if err != nil {
		return nil, fmt.Errorf("normalizing parent for %s: %w", logicalTarget, err)
	}
	if !pathAtOrBelow(physicalRoot, physicalParent) {
		return nil, fmt.Errorf("target %s escapes root %s", logicalTarget, allowedRoot)
	}
	parent, err := os.OpenRoot(physicalParent)
	if err != nil {
		return nil, fmt.Errorf("opening physical parent %s: %w", physicalParent, err)
	}
	target := &AtomicTarget{
		logicalPath:  filepath.Clean(logicalTarget),
		physicalPath: filepath.Join(filepath.Clean(physicalParent), filepath.Base(logicalTarget)),
		parentPath:   filepath.Clean(physicalParent),
		parent:       parent,
		name:         filepath.Base(logicalTarget),
	}
	if err := target.verifyParentIdentity(); err != nil {
		_ = parent.Close()
		return nil, err
	}
	return target, nil
}

func pathAtOrBelow(root, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (t *AtomicTarget) verifyParentIdentity() error {
	opened, err := t.parent.Stat(".")
	if err != nil {
		return fmt.Errorf("stating opened parent for %s: %w", t.logicalPath, err)
	}
	named, err := os.Stat(t.parentPath)
	if err != nil {
		return fmt.Errorf("stating physical parent for %s: %w", t.logicalPath, err)
	}
	if !os.SameFile(opened, named) {
		return fmt.Errorf("physical parent changed for %s", t.logicalPath)
	}
	return nil
}

func (t *AtomicTarget) LogicalPath() string        { return t.logicalPath }
func (t *AtomicTarget) PhysicalPath() string       { return t.physicalPath }
func (t *AtomicTarget) Stat() (fs.FileInfo, error) { return t.parent.Stat(t.name) }
func (t *AtomicTarget) WriteFileAtomic(content []byte, perm fs.FileMode) error {
	if err := t.verifyParentIdentity(); err != nil {
		return err
	}
	return writeFileAtomicRoot(t.parent, t.name, perm, func(dst io.Writer) error {
		_, err := dst.Write(content)
		return err
	})
}
func (t *AtomicTarget) Close() error {
	t.closeOnce.Do(func() { t.closeErr = t.parent.Close() })
	return t.closeErr
}
