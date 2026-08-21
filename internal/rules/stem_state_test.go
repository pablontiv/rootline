package rules

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
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

func TestDiscoverStemStateRejectsEscapingStemSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}

	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	outsideStem := filepath.Join(parent, "outside", ".stem")
	rootStem := filepath.Join(root, ".stem")
	mustWriteStemStateFile(t, outsideStem, "version: 2\nroot: true\n")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideStem, rootStem); err != nil {
		if errors.Is(err, os.ErrPermission) {
			t.Skipf("symlink creation unsupported: %v", err)
		}
		t.Fatal(err)
	}

	state, err := DiscoverStemState(context.Background(), root)
	if err != nil {
		if !strings.Contains(err.Error(), rootStem) {
			t.Fatalf("discovery error %q does not identify escaping stem symlink %s", err, rootStem)
		}
		return
	}
	if state.Stems[rootStem] != nil {
		t.Fatal("escaping .stem symlink must not ingest governance outside Root")
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

func TestDiscoverStemStatePreservesMalformedExternalAncestorAndContinuesToRoot(t *testing.T) {
	grand := t.TempDir()
	parent := filepath.Join(grand, "parent")
	root := filepath.Join(parent, "child")
	mustWriteStemStateFile(t, filepath.Join(grand, ".stem"), "version: 2\nroot: true\n")
	malformed := filepath.Join(parent, ".stem")
	mustWriteStemStateFile(t, malformed, "version: [broken\n")
	mustWriteStemStateFile(t, filepath.Join(root, "record.md"), "# Record\n")

	state, err := DiscoverStemState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if state.ParseErrors[malformed] == nil {
		t.Fatal("external parse error was not retained")
	}
	if state.Stems[filepath.Join(grand, ".stem")] == nil {
		t.Fatal("collector did not continue to valid root marker")
	}
	if containsStemStatePath(state.EvaluatedStemPaths(), malformed) {
		t.Fatal("untouched external malformed ancestor became scan-owned")
	}
}

func TestDiscoverStemStateHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := DiscoverStemState(ctx, t.TempDir())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("DiscoverStemState() error = %v, want context.Canceled", err)
	}
}

func TestDiscoverStemStateReturnsExternalStemReadError(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "child")
	parentStem := filepath.Join(parent, ".stem")
	mustWriteStemStateFile(t, filepath.Join(root, "record.md"), "# Record\n")
	if err := os.Mkdir(parentStem, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := DiscoverStemState(context.Background(), root)
	if err == nil {
		t.Fatal("DiscoverStemState() error = nil, want read error")
	}
	if strings.Contains(err.Error(), "invalid YAML") {
		t.Fatalf("operational read error was reported as a parse error: %v", err)
	}
}

func TestDiscoverStemStateMalformedExternalRootTextDoesNotStopDiscovery(t *testing.T) {
	grand := t.TempDir()
	parent := filepath.Join(grand, "parent")
	root := filepath.Join(parent, "child")
	mustWriteStemStateFile(t, filepath.Join(grand, ".stem"), "version: 2\nroot: true\n")
	malformed := filepath.Join(parent, ".stem")
	mustWriteStemStateFile(t, malformed, "version: [broken\nroot: true\n")
	mustWriteStemStateFile(t, filepath.Join(root, "record.md"), "# Record\n")

	state, err := DiscoverStemState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if state.ParseErrors[malformed] == nil {
		t.Fatal("external parse error was not retained")
	}
	if state.Stems[filepath.Join(grand, ".stem")] == nil {
		t.Fatal("malformed external root text stopped discovery before the valid root marker")
	}
}

func TestStemStateChainTraversesExternalAncestorUntilRootMarker(t *testing.T) {
	grand := t.TempDir()
	parent := filepath.Join(grand, "parent")
	root := filepath.Join(parent, "child")
	mustWriteStemStateFile(t, filepath.Join(grand, ".stem"), "version: 2\nroot: true\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n")
	mustWriteStemStateFile(t, filepath.Join(parent, ".stem"), "version: 2\nschema:\n  title:\n    type: string\n")
	mustWriteStemStateFile(t, filepath.Join(root, "record.md"), "# Record\n")

	state, err := DiscoverStemState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	chain := state.Chain(filepath.Join(root, "record.md"))
	got := make([]string, 0, len(chain))
	for _, entry := range chain {
		got = append(got, entry.Path)
	}
	requireStemStatePaths(t, got, []string{
		filepath.Join(grand, ".stem"),
		filepath.Join(parent, ".stem"),
	})
	if containsStemStatePath(state.EvaluatedStemPaths(), filepath.Join(parent, ".stem")) {
		t.Fatal("untouched external ancestor must remain context-only")
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
