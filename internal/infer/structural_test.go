package infer

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// mkDir creates a base dir, subdirs, and files for structural tests.
func mkDir(t *testing.T, base string, subdirs []string, files []string) {
	t.Helper()
	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatal(err)
	}
	for _, sd := range subdirs {
		if err := os.MkdirAll(filepath.Join(base, sd), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(base, f), []byte("# "+f), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDetectStructural_RequireIndex_Above90(t *testing.T) {
	dir := t.TempDir()
	// 5 subdirs, all with README.md (100% >= 90%)
	subdirs := []string{"a", "b", "c", "d", "e"}
	mkDir(t, dir, subdirs, nil)
	for _, sd := range subdirs {
		mkDir(t, filepath.Join(dir, sd), nil, []string{"README.md"})
	}

	inferences := DetectStructural(dir)

	found := false
	for _, inf := range inferences {
		if inf.Type == "add_structural_rule" && inf.Field == "require_index" && inf.Value == "README.md" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected require_index inference, got: %v", inferences)
	}
}

func TestDetectStructural_RequireIndex_Below90(t *testing.T) {
	dir := t.TempDir()
	// 5 subdirs, only 4 with README.md (80% < 90%)
	subdirs := []string{"a", "b", "c", "d", "e"}
	mkDir(t, dir, subdirs, nil)
	for _, sd := range subdirs[:4] {
		mkDir(t, filepath.Join(dir, sd), nil, []string{"README.md"})
	}
	// subdir "e" has no README.md

	inferences := DetectStructural(dir)

	for _, inf := range inferences {
		if inf.Type == "add_structural_rule" && inf.Field == "require_index" {
			t.Errorf("expected no require_index inference at 80%%, got: %v", inf)
		}
	}
}

func TestDetectStructural_MinSampleSize(t *testing.T) {
	dir := t.TempDir()
	// Only 2 subdirs (< 3 minimum sample size)
	subdirs := []string{"a", "b"}
	mkDir(t, dir, subdirs, nil)
	for _, sd := range subdirs {
		mkDir(t, filepath.Join(dir, sd), nil, []string{"README.md"})
	}

	inferences := DetectStructural(dir)

	if len(inferences) != 0 {
		t.Errorf("expected no inferences with only 2 subdirs, got: %v", inferences)
	}
}

func TestDetectStructural_MinMaxChildren_WithPadding(t *testing.T) {
	dir := t.TempDir()
	// 3 parent dirs with 3, 4, 5 children respectively
	// min = floor(3 * 0.8) = floor(2.4) = 2
	// max = ceil(5 * 1.2) = ceil(6.0) = 6
	parents := []string{"p1", "p2", "p3"}
	mkDir(t, dir, parents, nil)

	children1 := []string{"p1/c1", "p1/c2", "p1/c3"}
	children2 := []string{"p2/c1", "p2/c2", "p2/c3", "p2/c4"}
	children3 := []string{"p3/c1", "p3/c2", "p3/c3", "p3/c4", "p3/c5"}
	for _, c := range append(append(children1, children2...), children3...) {
		if err := os.MkdirAll(filepath.Join(dir, c), 0755); err != nil {
			t.Fatal(err)
		}
	}

	inferences := DetectStructural(dir)

	var minInf, maxInf *Inference
	for i, inf := range inferences {
		if inf.Type == "add_structural_rule" && inf.Field == "min_children" {
			minInf = &inferences[i]
		}
		if inf.Type == "add_structural_rule" && inf.Field == "max_children" {
			maxInf = &inferences[i]
		}
	}

	if minInf == nil {
		t.Errorf("expected min_children inference, got: %v", inferences)
	} else if minInf.Value != "2" {
		t.Errorf("expected min_children=2 (floor(3*0.8)), got: %s", minInf.Value)
	}

	if maxInf == nil {
		t.Errorf("expected max_children inference, got: %v", inferences)
	} else if maxInf.Value != "6" {
		t.Errorf("expected max_children=6 (ceil(5*1.2)), got: %s", maxInf.Value)
	}
}

func TestDetectStructural_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	// No subdirs at all

	inferences := DetectStructural(dir)

	if len(inferences) != 0 {
		t.Errorf("expected no inferences for empty dir, got: %v", inferences)
	}
}

func TestDetectStructural_NamingInconsistency(t *testing.T) {
	root := t.TempDir()
	for i := 1; i <= 8; i++ {
		name := fmt.Sprintf("E%02d-feature-%d", i, i)
		mkDir(t, root, []string{name}, nil)
	}
	mkDir(t, root, []string{"notes"}, nil)
	mkDir(t, root, []string{"archive"}, nil)

	got := DetectStructural(root)
	found := false
	for _, inf := range got {
		if inf.Type == "naming_inconsistency" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected naming_inconsistency inference for outliers")
	}
}

func TestDetectStructural_NoNamingInconsistency_AllMatch(t *testing.T) {
	root := t.TempDir()
	for i := 1; i <= 5; i++ {
		name := fmt.Sprintf("E%02d-feature-%d", i, i)
		mkDir(t, root, []string{name}, nil)
	}
	got := DetectStructural(root)
	for _, inf := range got {
		if inf.Type == "naming_inconsistency" {
			t.Error("should not flag when all children match")
		}
	}
}

func TestDetectStructural_NoNamingInconsistency_NoPattern(t *testing.T) {
	root := t.TempDir()
	mkDir(t, root, []string{"alpha", "beta", "gamma", "delta"}, nil)
	got := DetectStructural(root)
	for _, inf := range got {
		if inf.Type == "naming_inconsistency" {
			t.Error("should not flag when no dominant pattern exists")
		}
	}
}
