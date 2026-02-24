package infer

import (
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

// makeHierarchyRecords creates records with paths relative to a scan root.
func makeHierarchyRecords(scanRoot string, paths []string, frontmatters []map[string]any) []*extract.Record {
	records := make([]*extract.Record, len(paths))
	for i, p := range paths {
		var fm map[string]any
		if i < len(frontmatters) {
			fm = frontmatters[i]
		}
		records[i] = &extract.Record{
			Path:        filepath.Join(scanRoot, p),
			Frontmatter: fm,
		}
	}
	return records
}

func TestDetectLevels_FourLevelHierarchy(t *testing.T) {
	root := "/tmp/epics"
	records := makeHierarchyRecords(root, []string{
		"E01-infra/README.md",
		"E02-platform/README.md",
		"E01-infra/F01-network/README.md",
		"E01-infra/F02-storage/README.md",
		"E02-platform/F01-auth/README.md",
		"E02-platform/F02-api/README.md",
		"E01-infra/F01-network/S001-core/README.md",
		"E01-infra/F01-network/S002-dns/README.md",
		"E01-infra/F01-network/S001-core/T001-setup.md",
		"E01-infra/F01-network/S001-core/T002-config.md",
	}, nil)

	levels := DetectLevels(records, root)

	if len(levels) != 4 {
		t.Fatalf("expected 4 levels, got %d", len(levels))
	}

	expected := []struct {
		prefix string
		digits int
		depth  int
	}{
		{"E", 2, 0},
		{"F", 2, 1},
		{"S", 3, 2},
		{"T", 3, 3},
	}

	for i, exp := range expected {
		if levels[i].Prefix != exp.prefix {
			t.Errorf("level %d: prefix=%q, want %q", i, levels[i].Prefix, exp.prefix)
		}
		if levels[i].Digits != exp.digits {
			t.Errorf("level %d: digits=%d, want %d", i, levels[i].Digits, exp.digits)
		}
		if levels[i].Depth != exp.depth {
			t.Errorf("level %d: depth=%d, want %d", i, levels[i].Depth, exp.depth)
		}
	}
}

func TestDetectLevels_NoPatterns(t *testing.T) {
	root := "/tmp/docs"
	records := makeHierarchyRecords(root, []string{
		"intro/README.md",
		"guides/setup.md",
		"guides/deploy.md",
	}, nil)

	levels := DetectLevels(records, root)
	if len(levels) != 0 {
		t.Errorf("expected 0 levels for flat dirs, got %d", len(levels))
	}
}

func TestDetectLevels_SinglePattern(t *testing.T) {
	root := "/tmp/docs"
	records := makeHierarchyRecords(root, []string{
		"E01-infra/README.md",
		"E02-platform/README.md",
	}, nil)

	levels := DetectLevels(records, root)
	// Single level is detected but AnalyzeHierarchy will reject (needs ≥2).
	if len(levels) != 1 {
		t.Errorf("expected 1 level, got %d", len(levels))
	}
}

func TestDetectLevels_MixedNaming(t *testing.T) {
	root := "/tmp/docs"
	records := makeHierarchyRecords(root, []string{
		"E01-infra/README.md",
		"E02-platform/README.md",
		"guides/README.md", // non-pattern dir
		"E01-infra/F01-net/README.md",
		"E01-infra/F02-store/README.md",
	}, nil)

	levels := DetectLevels(records, root)
	if len(levels) != 2 {
		t.Errorf("expected 2 levels (E and F), got %d", len(levels))
	}
}

func TestDetectLevels_SingleDirNotALevel(t *testing.T) {
	root := "/tmp/docs"
	records := makeHierarchyRecords(root, []string{
		"E01-infra/README.md",
		"E01-infra/F01-net/README.md", // only 1 F dir → not a level
	}, nil)

	levels := DetectLevels(records, root)
	// E has only 1 dir, F has only 1 dir → neither qualifies.
	if len(levels) != 0 {
		t.Errorf("expected 0 levels (single dirs), got %d", len(levels))
	}
}

func TestDetectLevels_RecordAssignment(t *testing.T) {
	root := "/tmp/epics"
	records := makeHierarchyRecords(root, []string{
		"E01-a/README.md",
		"E02-b/README.md",
		"E01-a/F01-x/README.md",
		"E01-a/F02-y/README.md",
	}, nil)

	levels := DetectLevels(records, root)
	if len(levels) != 2 {
		t.Fatalf("expected 2 levels, got %d", len(levels))
	}

	// E-level records: E01/README.md and E02/README.md
	// But records are assigned to the DEEPEST matching level.
	// E01/README.md is inside E01 (depth 0) and has no deeper match → E level.
	// E01/F01/README.md matches both E (depth 0) and F (depth 1) → F level.
	if len(levels[0].Records) != 2 {
		t.Errorf("E-level: expected 2 records, got %d", len(levels[0].Records))
	}
	if len(levels[1].Records) != 2 {
		t.Errorf("F-level: expected 2 records, got %d", len(levels[1].Records))
	}
}

func TestDetectSequencePattern_Standard(t *testing.T) {
	prefix, digits, ok := detectSequencePattern([]string{"E03-foo", "E04-bar"})
	if !ok || prefix != "E" || digits != 2 {
		t.Errorf("got (%q, %d, %v), want (E, 2, true)", prefix, digits, ok)
	}
}

func TestDetectSequencePattern_ThreeDigits(t *testing.T) {
	prefix, digits, ok := detectSequencePattern([]string{"S001-core", "S002-dns"})
	if !ok || prefix != "S" || digits != 3 {
		t.Errorf("got (%q, %d, %v), want (S, 3, true)", prefix, digits, ok)
	}
}

func TestDetectSequencePattern_NoPattern(t *testing.T) {
	_, _, ok := detectSequencePattern([]string{"docs", "src", "lib"})
	if ok {
		t.Error("expected no pattern detected")
	}
}

func TestDetectSequencePattern_Inconsistent(t *testing.T) {
	_, _, ok := detectSequencePattern([]string{"E01-foo", "F02-bar"})
	if ok {
		t.Error("expected no pattern for inconsistent prefixes")
	}
}

func TestAnalyzeHierarchy_Detected(t *testing.T) {
	root := "/tmp/epics"
	records := makeHierarchyRecords(root, []string{
		"E01-a/README.md",
		"E02-b/README.md",
		"E01-a/F01-x/README.md",
		"E01-a/F02-y/README.md",
		"E01-a/F01-x/T001-task1.md",
		"E01-a/F01-x/T002-task2.md",
		"E01-a/F02-y/T001-task1.md",
		"E01-a/F02-y/T002-task2.md",
	}, []map[string]any{
		{"estado": "Completed"},
		{"estado": "Pending"},
		{"estado": "In Progress"},
		{"estado": "Pending"},
		{"estado": "Completed", "tipo": "software-module", "ejecutable_en": "go test"},
		{"estado": "Pending", "tipo": "ci-cd", "ejecutable_en": "make"},
		{"estado": "In Progress", "tipo": "software-module", "ejecutable_en": "go build"},
		{"estado": "Blocked", "tipo": "documentation", "ejecutable_en": "mdbook"},
	})

	result := AnalyzeHierarchy(records, root)

	if !result.Detected {
		t.Fatal("expected hierarchy to be detected")
	}

	if len(result.Levels) < 2 {
		t.Fatalf("expected ≥2 levels, got %d", len(result.Levels))
	}

	// "estado" is present at all levels → should be in root.
	if _, ok := result.Root.Schema["estado"]; !ok {
		t.Error("expected 'estado' in root schema (present at all levels)")
	}

	// "tipo" and "ejecutable_en" are only in the deepest level (T tasks).
	// They should NOT be in root.
	if _, ok := result.Root.Schema["tipo"]; ok {
		t.Error("expected 'tipo' NOT in root (only present at task level)")
	}
	if _, ok := result.Root.Schema["ejecutable_en"]; ok {
		t.Error("expected 'ejecutable_en' NOT in root (only present at task level)")
	}
}

func TestAnalyzeHierarchy_EnumValuesDiffer(t *testing.T) {
	root := "/tmp/epics"
	// tipo has disjoint enum values at E-level vs F-level (zero overlap).
	records := makeHierarchyRecords(root, []string{
		"E01-a/README.md",
		"E02-b/README.md",
		"E01-a/F01-x/README.md",
		"E01-a/F02-y/README.md",
	}, []map[string]any{
		{"tipo": "feature"},
		{"tipo": "historia"},
		{"tipo": "software-module"},
		{"tipo": "ci-cd"},
	})

	result := AnalyzeHierarchy(records, root)
	if !result.Detected {
		t.Fatal("expected hierarchy to be detected")
	}

	// tipo has disjoint enum values at E-level vs F-level → should NOT be in root.
	if _, ok := result.Root.Schema["tipo"]; ok {
		t.Error("expected 'tipo' NOT in root when enum values are disjoint between levels")
	}

	// Each level should have tipo in OnlyHere with its own values.
	for _, ls := range result.Levels {
		if _, ok := ls.OnlyHere["tipo"]; !ok {
			t.Errorf("expected 'tipo' in OnlyHere for level %s", ls.Level.Prefix)
		}
	}
}

func TestAnalyzeHierarchy_EnumValuesSame(t *testing.T) {
	root := "/tmp/epics"
	records := makeHierarchyRecords(root, []string{
		"E01-a/README.md",
		"E02-b/README.md",
		"E01-a/F01-x/README.md",
		"E01-a/F02-y/README.md",
	}, []map[string]any{
		{"estado": "Pending"},
		{"estado": "Completed"},
		{"estado": "Pending"},
		{"estado": "Completed"},
	})

	result := AnalyzeHierarchy(records, root)
	if !result.Detected {
		t.Fatal("expected hierarchy to be detected")
	}

	// estado has same enum values at both levels → should be in root.
	if _, ok := result.Root.Schema["estado"]; !ok {
		t.Error("expected 'estado' in root when enum values are the same across levels")
	}
}

func TestAnalyzeHierarchy_NotDetected(t *testing.T) {
	root := "/tmp/flat"
	records := makeHierarchyRecords(root, []string{
		"docs/a.md",
		"docs/b.md",
	}, []map[string]any{
		{"title": "A"},
		{"title": "B"},
	})

	result := AnalyzeHierarchy(records, root)
	if result.Detected {
		t.Error("expected Detected=false for flat directory")
	}
}

func TestAnalyzeHierarchy_SingleLevelNotDetected(t *testing.T) {
	root := "/tmp/epics"
	records := makeHierarchyRecords(root, []string{
		"E01-a/README.md",
		"E02-b/README.md",
	}, []map[string]any{
		{"estado": "Pending"},
		{"estado": "Completed"},
	})

	result := AnalyzeHierarchy(records, root)
	if result.Detected {
		t.Error("expected Detected=false for single level")
	}
}

func TestAnalyzeHierarchy_SequenceIds(t *testing.T) {
	root := "/tmp/epics"
	records := makeHierarchyRecords(root, []string{
		"E01-a/README.md",
		"E02-b/README.md",
		"E01-a/F01-x/README.md",
		"E01-a/F02-y/README.md",
	}, []map[string]any{
		{"estado": "Pending"},
		{"estado": "Completed"},
		{"estado": "Pending"},
		{"estado": "Completed"},
	})

	result := AnalyzeHierarchy(records, root)

	if !result.Detected {
		t.Fatal("expected hierarchy detected")
	}

	// Check that levels have sequence ids.
	for _, ls := range result.Levels {
		idField, ok := ls.OnlyHere["id"]
		if !ok {
			t.Errorf("level %s: expected 'id' in OnlyHere", ls.Level.Prefix)
			continue
		}
		if idField.Type != "sequence" {
			t.Errorf("level %s: id type=%q, want 'sequence'", ls.Level.Prefix, idField.Type)
		}
		if idField.Prefix != ls.Level.Prefix {
			t.Errorf("level %s: id prefix=%q, want %q", ls.Level.Prefix, idField.Prefix, ls.Level.Prefix)
		}
	}
}

func TestAnalyzeHierarchy_FieldAtAllLevelsGoesToRoot(t *testing.T) {
	root := "/tmp/epics"
	records := makeHierarchyRecords(root, []string{
		"E01-a/README.md",
		"E02-b/README.md",
		"E01-a/F01-x/README.md",
		"E01-a/F02-y/README.md",
	}, []map[string]any{
		{"estado": "Pending", "cliente": "acme"},
		{"estado": "Completed", "cliente": "corp"},
		{"estado": "Pending", "cliente": "acme"},
		{"estado": "Completed", "cliente": "corp"},
	})

	result := AnalyzeHierarchy(records, root)
	if !result.Detected {
		t.Fatal("expected hierarchy detected")
	}

	// Both "estado" and "cliente" are present at all levels → root.
	for _, name := range []string{"estado", "cliente"} {
		if _, ok := result.Root.Schema[name]; !ok {
			t.Errorf("expected %q in root schema", name)
		}
	}

	// Neither should be in any level's OnlyHere.
	for _, ls := range result.Levels {
		for _, name := range []string{"estado", "cliente"} {
			if _, ok := ls.OnlyHere[name]; ok {
				t.Errorf("level %s: expected %q NOT in OnlyHere", ls.Level.Prefix, name)
			}
		}
	}
}

func TestAnalyzeHierarchy_RootFilesIgnored(t *testing.T) {
	root := "/tmp/epics"
	// File directly at root (no pattern dir) should not break detection.
	records := makeHierarchyRecords(root, []string{
		"README.md", // root file
		"E01-a/README.md",
		"E02-b/README.md",
		"E01-a/F01-x/README.md",
		"E01-a/F02-y/README.md",
	}, []map[string]any{
		{"title": "Root"},
		{"estado": "Pending"},
		{"estado": "Completed"},
		{"estado": "Pending"},
		{"estado": "Completed"},
	})

	result := AnalyzeHierarchy(records, root)
	if !result.Detected {
		t.Fatal("expected hierarchy detected despite root file")
	}
	if len(result.Levels) != 2 {
		t.Errorf("expected 2 levels, got %d", len(result.Levels))
	}
}
