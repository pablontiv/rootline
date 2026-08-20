package rules

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscoverStemStatePreservesParseErrorsAndInventory(t *testing.T) {
	root := t.TempDir()
	mustWriteStemStateFile(t, filepath.Join(root, ".stem"), "version: 2\nroot: true\n")
	mustWriteStemStateFile(t, filepath.Join(root, "docs", ".stem"), "version: [broken\n")
	mustWriteStemStateFile(t, filepath.Join(root, "docs", "a.md"), "# A\n")
	mustWriteStemStateFile(t, filepath.Join(root, ".git", ".stem"), "version: 2\n")
	mustWriteStemStateFile(t, filepath.Join(root, "node_modules", "pkg", ".stem"), "version: 2\n")

	state, err := DiscoverStemState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if state.Stems[filepath.Join(root, ".stem")] == nil {
		t.Fatal("root stem was not parsed")
	}
	if state.ParseErrors[filepath.Join(root, "docs", ".stem")] == nil {
		t.Fatal("malformed stem parse error was not retained")
	}
	if _, ok := state.Entries[filepath.Join(root, "docs", "a.md")]; !ok {
		t.Fatal("record inventory is incomplete")
	}
	if _, ok := state.Entries[filepath.Join(root, ".git", ".stem")]; ok {
		t.Fatal(".git must be excluded")
	}
	if _, ok := state.Entries[filepath.Join(root, "node_modules", "pkg", ".stem")]; ok {
		t.Fatal("node_modules must be excluded")
	}
}

func TestDiscoverStemStatePreservesMalformedRootParseError(t *testing.T) {
	root := t.TempDir()
	rootStem := filepath.Join(root, ".stem")
	mustWriteStemStateFile(t, rootStem, "version: [broken\n")
	mustWriteStemStateFile(t, filepath.Join(root, "record.md"), "# Record\n")

	state, err := DiscoverStemState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if state.ParseErrors[rootStem] == nil {
		t.Fatalf("malformed root stem parse error was not retained at %s", rootStem)
	}
	if state.Stems[rootStem] != nil {
		t.Fatal("malformed root stem must not be retained as a parsed stem")
	}
}

func TestDiscoverStemStateIncludesExternalAncestorContext(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "subtree")
	mustWriteStemStateFile(t, filepath.Join(parent, ".stem"), "version: 2\nroot: true\n")
	mustWriteStemStateFile(t, filepath.Join(root, "record.md"), "# Record\n")

	state, err := DiscoverStemState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	parentStem := filepath.Join(parent, ".stem")
	if state.Stems[parentStem] == nil {
		t.Fatal("external governing parent stem was not retained")
	}
	if containsStemStatePath(state.EvaluatedStemPaths(), parentStem) {
		t.Fatal("external ancestor stem must not be evaluated as scan-owned diagnostics")
	}
}

func TestStemStateOverlayIsImmutableAndClearsParseError(t *testing.T) {
	root := t.TempDir()
	mustWriteStemStateFile(t, filepath.Join(root, ".stem"), "version: 2\nroot: true\n")
	malformedPath := filepath.Join(root, "docs", ".stem")
	mustWriteStemStateFile(t, malformedPath, "version: [broken\n")

	state, err := DiscoverStemState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if state.ParseErrors[malformedPath] == nil {
		t.Fatal("test setup did not retain malformed .stem")
	}

	clone, err := state.Overlay(malformedPath, []byte("version: 2\n"))
	if err != nil {
		t.Fatal(err)
	}

	if clone.Stems[malformedPath] == nil {
		t.Fatal("overlay clone did not contain parsed stem")
	}
	if clone.ParseErrors[malformedPath] != nil {
		t.Fatal("overlay clone did not clear parse error")
	}
	if state.Stems[malformedPath] != nil {
		t.Fatal("overlay mutated original stems")
	}
	if state.ParseErrors[malformedPath] == nil {
		t.Fatal("overlay mutated original parse errors")
	}
}

func TestStemStateChainStopsAtNestedRootMarker(t *testing.T) {
	root := t.TempDir()
	mustWriteStemStateFile(t, filepath.Join(root, ".stem"), "version: 2\nroot: true\n")
	mustWriteStemStateFile(t, filepath.Join(root, "docs", ".stem"), "version: 2\nroot: true\n")
	mustWriteStemStateFile(t, filepath.Join(root, "docs", "topic", ".stem"), "version: 2\n")
	recordPath := filepath.Join(root, "docs", "topic", "a.md")
	mustWriteStemStateFile(t, recordPath, "# A\n")

	state, err := DiscoverStemState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	chain := state.Chain(recordPath)
	got := make([]string, 0, len(chain))
	for _, entry := range chain {
		got = append(got, entry.Path)
	}
	requireStemStatePaths(t, got, []string{
		filepath.Join(root, "docs", ".stem"),
		filepath.Join(root, "docs", "topic", ".stem"),
	})
}

func TestStemStateEvaluatedPathsAreSorted(t *testing.T) {
	root := t.TempDir()
	state := &StemState{
		Root: filepath.Clean(root),
		Stems: map[string]*StemFile{
			filepath.Join(root, "b", ".stem"): {Path: filepath.Join(root, "b", ".stem")},
			filepath.Join(root, "a", ".stem"): {Path: filepath.Join(root, "a", ".stem")},
		},
		ParseErrors: map[string]error{
			filepath.Join(root, "c", ".stem"): os.ErrInvalid,
		},
	}

	requireStemStatePaths(t, state.EvaluatedStemPaths(), []string{
		filepath.Join(root, "a", ".stem"),
		filepath.Join(root, "b", ".stem"),
		filepath.Join(root, "c", ".stem"),
	})
}

func TestStemStateMatchingFilesUsesImmediateNonDirectoryEntries(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "docs")
	state := &StemState{
		Root: filepath.Clean(root),
		Entries: map[string]StemStateEntry{
			filepath.Join(dir, "a.md"):           {IsDir: false},
			filepath.Join(dir, "b.txt"):          {IsDir: false},
			filepath.Join(dir, "nested", "c.md"): {IsDir: false},
			filepath.Join(dir, "draft.md"):       {IsDir: true},
			filepath.Join(dir, "z.md"):           {IsDir: false},
		},
	}

	requireStemStatePaths(t, state.MatchingFiles(dir, "*.md"), []string{
		filepath.Join(dir, "a.md"),
		filepath.Join(dir, "z.md"),
	})
}

func mustWriteStemStateFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsStemStatePath(paths []string, want string) bool {
	for _, path := range paths {
		if path == want {
			return true
		}
	}
	return false
}

func requireStemStatePaths(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %#v, want %#v", got, want)
	}
}
