# Structural Inference Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add structural inference detector (category 4) that infers `require_index`, `min_children`, and `max_children` rules from directory structure.

**Architecture:** New `internal/infer/structural.go` walks directories, collects stats (index file presence, child counts), and returns `[]Inference` compatible with the analyze report. Integrates into `cmd/rootline/analyze.go` as detector #13 and into `cmd/rootline/init.go` for `.stem` generation.

**Tech Stack:** `os.ReadDir` for directory walking, `math` for floor/ceil padding. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-03-20-three-features-design.md` (Feature 3)

---

## Chunk 1: Structural Detector

### Task 1: Implement DetectStructural

**Files:**
- Create: `internal/infer/structural.go`
- Create: `internal/infer/structural_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/infer/structural_test.go
package infer

import (
	"os"
	"path/filepath"
	"testing"
)

// mkDir creates a directory and optional files inside it.
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
		if err := os.WriteFile(filepath.Join(base, f), []byte("# "+f+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDetectStructural_RequireIndex_Above90(t *testing.T) {
	root := t.TempDir()
	// Create 5 subdirs, all with README.md (100%).
	for _, name := range []string{"E01-a", "E02-b", "E03-c", "E04-d", "E05-e"} {
		mkDir(t, filepath.Join(root, name), nil, []string{"README.md"})
	}

	inferences := DetectStructural(root)

	found := false
	for _, inf := range inferences {
		if inf.Type == "add_structural_rule" && inf.Field == "require_index" {
			found = true
			if inf.Value != "README.md" {
				t.Errorf("expected value README.md, got %s", inf.Value)
			}
		}
	}
	if !found {
		t.Error("expected require_index inference for 100% presence")
	}
}

func TestDetectStructural_RequireIndex_Below90(t *testing.T) {
	root := t.TempDir()
	// 5 subdirs, only 4 have README.md (80% < 90%).
	for i, name := range []string{"E01-a", "E02-b", "E03-c", "E04-d", "E05-e"} {
		files := []string{}
		if i < 4 {
			files = []string{"README.md"}
		}
		mkDir(t, filepath.Join(root, name), nil, files)
	}

	inferences := DetectStructural(root)

	for _, inf := range inferences {
		if inf.Type == "add_structural_rule" && inf.Field == "require_index" {
			t.Error("should NOT infer require_index at 80% presence")
		}
	}
}

func TestDetectStructural_MinSampleSize(t *testing.T) {
	root := t.TempDir()
	// Only 2 subdirs — below min sample size of 3.
	mkDir(t, filepath.Join(root, "E01-a"), nil, []string{"README.md"})
	mkDir(t, filepath.Join(root, "E02-b"), nil, []string{"README.md"})

	inferences := DetectStructural(root)

	for _, inf := range inferences {
		if inf.Type == "add_structural_rule" {
			t.Errorf("should NOT infer structural rules with only 2 dirs, got %s=%s", inf.Field, inf.Value)
		}
	}
}

func TestDetectStructural_MinMaxChildren_WithPadding(t *testing.T) {
	root := t.TempDir()
	// Create 3 parent dirs with varying child counts: 3, 4, 5.
	for _, parent := range []string{"A", "B", "C"} {
		pdir := filepath.Join(root, parent)
		var children []string
		count := 3
		switch parent {
		case "B":
			count = 4
		case "C":
			count = 5
		}
		for i := 0; i < count; i++ {
			children = append(children, filepath.Join(pdir, "child-"+string(rune('a'+i))))
		}
		mkDir(t, pdir, nil, nil)
		for _, c := range children {
			mkDir(t, c, nil, nil)
		}
	}

	inferences := DetectStructural(root)

	var minVal, maxVal string
	for _, inf := range inferences {
		if inf.Field == "min_children" {
			minVal = inf.Value
		}
		if inf.Field == "max_children" {
			maxVal = inf.Value
		}
	}

	// min = floor(3 * 0.8) = floor(2.4) = 2
	if minVal != "2" {
		t.Errorf("expected min_children=2, got %s", minVal)
	}
	// max = ceil(5 * 1.2) = ceil(6.0) = 6
	if maxVal != "6" {
		t.Errorf("expected max_children=6, got %s", maxVal)
	}
}

func TestDetectStructural_EmptyDir(t *testing.T) {
	root := t.TempDir()

	inferences := DetectStructural(root)

	if len(inferences) != 0 {
		t.Errorf("expected no inferences for empty dir, got %d", len(inferences))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infer/ -run TestDetectStructural -v`
Expected: FAIL with "undefined: DetectStructural"

- [ ] **Step 3: Implement DetectStructural**

```go
// internal/infer/structural.go
package infer

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
)

const (
	structuralMinSampleSize    = 3
	structuralIndexThreshold   = 0.90
	structuralChildPaddingDown = 0.8
	structuralChildPaddingUp   = 1.2
)

// DetectStructural analyzes directory structure under scanRoot and infers
// structural rules: require_index, min_children, max_children.
// Returns inferences compatible with AnalyzeReport.
func DetectStructural(scanRoot string) []Inference {
	entries, err := os.ReadDir(scanRoot)
	if err != nil {
		return nil
	}

	// Collect non-hidden subdirectories of scanRoot.
	var subdirs []string
	for _, e := range entries {
		if e.IsDir() && !isHidden(e.Name()) {
			subdirs = append(subdirs, e.Name())
		}
	}

	if len(subdirs) < structuralMinSampleSize {
		return nil
	}

	var inferences []Inference

	// Detect require_index: check how many subdirs have README.md.
	indexCount := 0
	for _, sd := range subdirs {
		indexPath := filepath.Join(scanRoot, sd, "README.md")
		if _, err := os.Stat(indexPath); err == nil {
			indexCount++
		}
	}

	ratio := float64(indexCount) / float64(len(subdirs))
	if ratio >= structuralIndexThreshold {
		inferences = append(inferences, Inference{
			Type:    "add_structural_rule",
			Field:   "require_index",
			Value:   "README.md",
			Message: fmt.Sprintf("README.md found in %d/%d subdirectories (%.0f%%)", indexCount, len(subdirs), ratio*100),
		})
	}

	// Detect min/max_children: analyze child counts of subdirs that themselves have children.
	var childCounts []int
	for _, sd := range subdirs {
		sdPath := filepath.Join(scanRoot, sd)
		children, err := os.ReadDir(sdPath)
		if err != nil {
			continue
		}
		count := 0
		for _, c := range children {
			if c.IsDir() && !isHidden(c.Name()) {
				count++
			}
		}
		if count > 0 {
			childCounts = append(childCounts, count)
		}
	}

	if len(childCounts) >= structuralMinSampleSize {
		minCount := childCounts[0]
		maxCount := childCounts[0]
		for _, c := range childCounts[1:] {
			if c < minCount {
				minCount = c
			}
			if c > maxCount {
				maxCount = c
			}
		}

		paddedMin := int(math.Floor(float64(minCount) * structuralChildPaddingDown))
		if paddedMin < 1 {
			paddedMin = 1
		}
		paddedMax := int(math.Ceil(float64(maxCount) * structuralChildPaddingUp))

		inferences = append(inferences, Inference{
			Type:    "add_structural_rule",
			Field:   "min_children",
			Value:   fmt.Sprintf("%d", paddedMin),
			Message: fmt.Sprintf("minimum observed children: %d (padded to %d)", minCount, paddedMin),
		})
		inferences = append(inferences, Inference{
			Type:    "add_structural_rule",
			Field:   "max_children",
			Value:   fmt.Sprintf("%d", paddedMax),
			Message: fmt.Sprintf("maximum observed children: %d (padded to %d)", maxCount, paddedMax),
		})
	}

	return inferences
}

func isHidden(name string) bool {
	return len(name) > 0 && name[0] == '.'
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infer/ -run TestDetectStructural -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/infer/structural.go internal/infer/structural_test.go
git commit -m "feat(infer): add structural detector for require_index and min/max_children"
```

## Chunk 2: Integration with analyze and init

### Task 2: Wire into analyze command

**Files:**
- Modify: `cmd/rootline/analyze.go:85-122` (add category to list)

- [ ] **Step 1: Write failing test**

Add to `cmd/rootline/analyze_test.go` (or create if needed):

```go
func TestAnalyze_IncludesStructuralCategory(t *testing.T) {
	dir := setupProject(t, map[string]string{
		".git/HEAD":           "ref: refs/heads/main\n",
		"E01-a/README.md":    "---\ntipo: epic\n---\n# E01\n",
		"E02-b/README.md":    "---\ntipo: epic\n---\n# E02\n",
		"E03-c/README.md":    "---\ntipo: epic\n---\n# E03\n",
	})

	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"analyze", dir, "-o", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("analyze failed: %v", err)
	}

	var report infer.AnalyzeReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("parsing report: %v", err)
	}

	found := false
	for _, cat := range report.Categories {
		if cat.ID == "structural" {
			found = true
		}
	}
	if !found {
		t.Error("analyze report missing 'structural' category")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/rootline/ -run TestAnalyze_IncludesStructuralCategory -v`
Expected: FAIL (no structural category in output)

- [ ] **Step 3: Add structural category to analyze.go**

In `cmd/rootline/analyze.go`, add to the `categories` slice (after the traceability entry, ~line 121):

```go
{"structural", "Structural Rule Detection", func() []infer.Inference {
	return infer.DetectStructural(root)
}},
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/rootline/ -run TestAnalyze_IncludesStructuralCategory -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/rootline/analyze.go
git commit -m "feat(analyze): integrate structural detector as category #13"
```

### Task 3: Wire into init command

**Files:**
- Modify: `cmd/rootline/init.go` (add structural section to generated YAML)

- [ ] **Step 1: Write failing test**

```go
func TestInit_IncludesStructural(t *testing.T) {
	dir := setupProject(t, map[string]string{
		".git/HEAD":                       "ref: refs/heads/main\n",
		"E01-a/README.md":                "---\ntipo: epic\n---\n# E01\n",
		"E01-a/F01-x/README.md":          "---\ntipo: feature\n---\n# F01\n",
		"E01-a/F02-y/README.md":          "---\ntipo: feature\n---\n# F02\n",
		"E01-a/F03-z/README.md":          "---\ntipo: feature\n---\n# F03\n",
		"E02-b/README.md":                "---\ntipo: epic\n---\n# E02\n",
		"E02-b/F01-x/README.md":          "---\ntipo: feature\n---\n# F01\n",
		"E02-b/F02-y/README.md":          "---\ntipo: feature\n---\n# F02\n",
		"E02-b/F03-z/README.md":          "---\ntipo: feature\n---\n# F03\n",
		"E03-c/README.md":                "---\ntipo: epic\n---\n# E03\n",
		"E03-c/F01-x/README.md":          "---\ntipo: feature\n---\n# F01\n",
		"E03-c/F02-y/README.md":          "---\ntipo: feature\n---\n# F02\n",
		"E03-c/F03-z/README.md":          "---\ntipo: feature\n---\n# F03\n",
	})

	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"init", dir, "--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "structural:") {
		t.Error("init output should contain structural: section")
	}
	if !strings.Contains(output, "require_index:") {
		t.Error("init output should contain require_index")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/rootline/ -run TestInit_IncludesStructural -v`
Expected: FAIL (no structural section in output)

- [ ] **Step 3: Add structural generation to init.go**

Add a helper function and call it from both `runInitFlat` and `runInitHierarchical`:

```go
// generateStructuralYAML generates the structural: section based on directory analysis.
func generateStructuralYAML(scanRoot string) string {
	inferences := infer.DetectStructural(scanRoot)
	if len(inferences) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("structural:\n  subdirs:\n")
	for _, inf := range inferences {
		if inf.Type != "add_structural_rule" {
			continue
		}
		switch inf.Field {
		case "require_index":
			fmt.Fprintf(&b, "    require_index: %s\n", inf.Value)
		case "min_children":
			fmt.Fprintf(&b, "    min_children: %s\n", inf.Value)
		case "max_children":
			fmt.Fprintf(&b, "    max_children: %s\n", inf.Value)
		}
	}
	return b.String()
}
```

In `generateStemYAML`, append structural section before returning:

```go
// After the schema fields loop, before return:
structural := generateStructuralYAML(/* need to pass scanRoot */)
b.WriteString(structural)
```

Note: `generateStemYAML` and `generateHierarchicalRootYAML` need an extra `scanRoot string` parameter. Update their signatures and call sites.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/rootline/ -run TestInit_IncludesStructural -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/rootline/init.go
git commit -m "feat(init): generate structural section from directory analysis"
```

### Task 4: Verification

- [ ] **Step 1: Run all tests**

Run: `go test ./... -race`
Expected: All PASS

- [ ] **Step 2: Check coverage**

Run: `go test ./internal/infer/ -run TestDetectStructural -coverprofile=c.out && go tool cover -func=c.out | grep structural`
Expected: Good coverage for structural.go

- [ ] **Step 3: Integration test**

Run: `go build -o /tmp/rootline-test ./cmd/rootline/ && /tmp/rootline-test analyze docs/epics/ -o json | jq '.categories[] | select(.id=="structural")'`
Expected: Shows structural category with inferences for docs/epics/

- [ ] **Step 4: Build check**

Run: `go build ./...`
Expected: Clean build
