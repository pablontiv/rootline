package rules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateDirectory_RequireIndex(t *testing.T) {
	dir := t.TempDir()
	// Create 3 subdirs, only 2 have README.md.
	for _, name := range []string{"sub1", "sub2", "sub3"} {
		mustMkdir(t, filepath.Join(dir, name))
	}
	mustWriteTestFile(t, filepath.Join(dir, "sub1", "README.md"), []byte("# Sub1"))
	mustWriteTestFile(t, filepath.Join(dir, "sub2", "README.md"), []byte("# Sub2"))
	// sub3 has no README.md

	stem := &StemFile{
		Path: "test/.stem",
		Structural: StructuralRules{
			Subdirs: SubdirRules{RequireIndex: "README.md", Severity: "warn"},
		},
	}

	errs := ValidateDirectory(context.Background(), dir, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	want := ValidationError{
		Rule:     "require_index",
		Field:    "directory",
		Message:  `subdirectory "sub3" is missing required index file "README.md"`,
		Source:   stem.Path,
		Severity: "warn",
	}
	if errs[0] != want {
		t.Errorf("error = %+v, want %+v", errs[0], want)
	}
}

func TestValidateDirectory_RequireIndexEntry(t *testing.T) {
	for _, tc := range []struct {
		name    string
		target  string
		symlink bool
		message string
	}{
		{name: "regular file", target: "file"},
		{
			name:    "directory",
			target:  "directory",
			message: `subdirectory "sub" required index file "README.md" is not a regular file`,
		},
		{name: "symlink to regular file", target: "file", symlink: true},
		{
			name:    "symlink to directory",
			target:  "directory",
			symlink: true,
			message: `subdirectory "sub" required index file "README.md" is not a regular file`,
		},
		{
			name:    "dangling symlink",
			target:  "missing",
			symlink: true,
			message: `subdirectory "sub" is missing required index file "README.md"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.symlink && runtime.GOOS == "windows" {
				t.Skip("symlink creation requires privileges on Windows")
			}
			dir := t.TempDir()
			mustMkdir(t, filepath.Join(dir, "sub"))
			indexPath := filepath.Join(dir, "sub", "README.md")
			targetPath := indexPath
			if tc.symlink {
				targetPath = filepath.Join(dir, "sub", "target")
			}
			switch tc.target {
			case "file":
				mustWriteTestFile(t, targetPath, []byte("# Index"))
			case "directory":
				mustMkdir(t, targetPath)
			}
			if tc.symlink {
				if err := os.Symlink(targetPath, indexPath); err != nil {
					t.Fatal(err)
				}
			}

			stem := &StemFile{
				Path: "test/.stem",
				Structural: StructuralRules{
					Subdirs: SubdirRules{RequireIndex: "README.md", Severity: "warn"},
				},
			}
			errs := ValidateDirectory(context.Background(), dir, stem)
			if tc.message == "" {
				if len(errs) != 0 {
					t.Fatalf("got errors for a regular index file: %v", errs)
				}
				return
			}
			if len(errs) != 1 {
				t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
			}
			want := ValidationError{
				Rule:     "require_index",
				Field:    "directory",
				Message:  tc.message,
				Source:   stem.Path,
				Severity: "warn",
			}
			if errs[0] != want {
				t.Errorf("error = %+v, want %+v", errs[0], want)
			}
		})
	}
}

func TestValidateDirectory_RequireIndexStatError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows may classify a non-directory path component as missing")
	}
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "sub"))
	mustWriteTestFile(t, filepath.Join(dir, "sub", "README.md"), []byte("# Index"))
	requiredIndex := filepath.Join("README.md", "child")
	// Traversing a regular file yields ENOTDIR without depending on permissions.
	_, statErr := os.Stat(filepath.Join(dir, "sub", requiredIndex))
	if statErr == nil || os.IsNotExist(statErr) {
		t.Fatalf("fixture stat error = %v, want a non-not-found error", statErr)
	}
	stem := &StemFile{
		Path: "test/.stem",
		Structural: StructuralRules{
			Subdirs: SubdirRules{RequireIndex: requiredIndex, Severity: "warn"},
		},
	}

	errs := ValidateDirectory(context.Background(), dir, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	want := ValidationError{
		Rule:     "structural",
		Field:    "directory",
		Message:  fmt.Sprintf("cannot stat required index file %q in subdirectory %q: %v", requiredIndex, "sub", statErr),
		Source:   stem.Path,
		Severity: "warn",
	}
	if errs[0] != want {
		t.Errorf("error = %+v, want %+v", errs[0], want)
	}
}

func TestValidateDirectory_MinChildren(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "only-one"))

	stem := &StemFile{
		Path: "test/.stem",
		Structural: StructuralRules{
			Subdirs: SubdirRules{MinChildren: 2},
		},
	}

	errs := ValidateDirectory(context.Background(), dir, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	if errs[0].Rule != "min_children" {
		t.Errorf("rule = %q, want min_children", errs[0].Rule)
	}
}

func TestValidateDirectory_MaxChildren(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		mustMkdir(t, filepath.Join(dir, filepath.Base(t.Name())+string(rune('a'+i))))
	}

	stem := &StemFile{
		Path: "test/.stem",
		Structural: StructuralRules{
			Subdirs: SubdirRules{MaxChildren: 3},
		},
	}

	errs := ValidateDirectory(context.Background(), dir, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	if errs[0].Rule != "max_children" {
		t.Errorf("rule = %q, want max_children", errs[0].Rule)
	}
}

func TestValidateDirectory_NoStructural(t *testing.T) {
	dir := t.TempDir()
	stem := &StemFile{Path: "test/.stem"}

	errs := ValidateDirectory(context.Background(), dir, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (no structural rules)", len(errs))
	}
}

func TestValidateDirectory_NilStem(t *testing.T) {
	dir := t.TempDir()
	errs := ValidateDirectory(context.Background(), dir, nil)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (nil stem)", len(errs))
	}
}

func TestValidateDirectory_DefaultSeverity(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "sub"))

	stem := &StemFile{
		Path: "test/.stem",
		Structural: StructuralRules{
			Subdirs: SubdirRules{RequireIndex: "README.md"},
		},
	}

	errs := ValidateDirectory(context.Background(), dir, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1", len(errs))
	}
	if errs[0].Severity != "error" {
		t.Errorf("severity = %q, want error (default)", errs[0].Severity)
	}
}
