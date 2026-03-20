# Graph --open Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--open` flag to `rootline graph` that renders an interactive Mermaid diagram in the browser.

**Architecture:** New `internal/graph/html.go` handles HTML rendering (embedded template via `go:embed`) and cross-platform browser opening. The `cmd/rootline/graph.go` command gains an `--open` flag that generates Mermaid text, passes it to `RenderHTML`, writes a temp file, and opens it.

**Tech Stack:** Go `html/template`, `go:embed`, `os/exec` for browser, `os.CreateTemp` for temp files.

**Spec:** `docs/superpowers/specs/2026-03-20-three-features-design.md` (Feature 1)

---

## Chunk 1: HTML Rendering and Browser Opening

### Task 1: Create HTML template

**Files:**
- Create: `internal/graph/templates/graph.html`

- [ ] **Step 1: Create the HTML template file**

```html
<!DOCTYPE html>
<html><head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Rootline Graph</title>
  <script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
</head><body>
  <pre class="mermaid">{{.Content}}</pre>
  <script>mermaid.initialize({startOnLoad: true, theme: 'default'});</script>
</body></html>
```

- [ ] **Step 2: Commit**

```bash
git add internal/graph/templates/graph.html
git commit -m "feat(graph): add HTML template for mermaid rendering"
```

### Task 2: Implement RenderHTML and OpenBrowser

**Files:**
- Create: `internal/graph/html.go`
- Create: `internal/graph/html_test.go`

- [ ] **Step 1: Write failing tests for RenderHTML**

```go
// internal/graph/html_test.go
package graph

import (
	"os"
	"strings"
	"testing"
)

func TestRenderHTML_ContainsMermaidContent(t *testing.T) {
	content := "graph TD;\n  A --> B;"
	path, err := RenderHTML(content)
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading temp file: %v", err)
	}

	html := string(data)
	if !strings.Contains(html, content) {
		t.Errorf("HTML does not contain mermaid content")
	}
	if !strings.Contains(html, "mermaid.min.js") {
		t.Errorf("HTML does not include mermaid.js CDN")
	}
	if !strings.Contains(html, "viewport") {
		t.Errorf("HTML does not include viewport meta")
	}
}

func TestRenderHTML_CreatesValidHTMLFile(t *testing.T) {
	path, err := RenderHTML("graph TD;")
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}
	defer os.Remove(path)

	if !strings.HasSuffix(path, ".html") {
		t.Errorf("temp file does not end in .html: %s", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat temp file: %v", err)
	}
	if info.Size() == 0 {
		t.Errorf("temp file is empty")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/graph/ -run TestRenderHTML -v`
Expected: FAIL with "undefined: RenderHTML"

- [ ] **Step 3: Implement html.go**

```go
// internal/graph/html.go
package graph

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"runtime"
)

//go:embed templates/graph.html
var templateFS embed.FS

var htmlTemplate = template.Must(template.ParseFS(templateFS, "templates/graph.html"))

type templateData struct {
	Content string
}

// RenderHTML renders mermaid content into a temporary HTML file.
// Returns the path to the created file. Caller is responsible for cleanup.
func RenderHTML(mermaidContent string) (string, error) {
	f, err := os.CreateTemp("", "rootline-graph-*.html")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer f.Close()

	if err := htmlTemplate.Execute(f, templateData{Content: mermaidContent}); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("executing template: %w", err)
	}

	return f.Name(), nil
}

// OpenBrowser opens the given file path in the default browser.
func OpenBrowser(path string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", "", path).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/graph/ -run TestRenderHTML -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/graph/html.go internal/graph/html_test.go
git commit -m "feat(graph): implement RenderHTML and OpenBrowser"
```

### Task 3: Wire --open flag into graph command

**Files:**
- Modify: `cmd/rootline/graph.go:17-21` (add graphOpen var)
- Modify: `cmd/rootline/graph.go:31-36` (register flag)
- Modify: `cmd/rootline/graph.go:48-161` (add --open logic in runGraph)

- [ ] **Step 1: Write failing test for --open flag validation**

Add to `cmd/rootline/graph_test.go`:

```go
func TestGraph_OpenWithCheck_ReturnsError(t *testing.T) {
	dir := setupProject(t, map[string]string{
		".git/HEAD":  "ref: refs/heads/main\n",
		"README.md":  "---\ntipo: doc\n---\n# Test\n",
	})
	cmd := newRootCmd()
	cmd.SetArgs([]string{"graph", dir, "--open", "--check"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when using --open with --check")
	}
}

func TestGraph_OpenWithDot_ReturnsError(t *testing.T) {
	dir := setupProject(t, map[string]string{
		".git/HEAD":  "ref: refs/heads/main\n",
		"README.md":  "---\ntipo: doc\n---\n# Test\n",
	})
	cmd := newRootCmd()
	cmd.SetArgs([]string{"graph", dir, "--open", "--format", "dot"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when using --open with --format dot")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/rootline/ -run "TestGraph_Open" -v`
Expected: FAIL (flag not recognized)

- [ ] **Step 3: Add --open flag and validation to graph.go**

In `cmd/rootline/graph.go`, add the flag variable alongside existing ones (line ~17):

```go
var (
	graphFormat string
	graphCheck  bool
	graphOpen   bool
	graphWhere  []string
)
```

Register the flag in `init()` (line ~32):

```go
graphCmd.Flags().BoolVar(&graphOpen, "open", false, "render diagram in browser")
```

Add validation at the start of `runGraph` (after line 48, before scan):

```go
if graphOpen && graphCheck {
	return fmt.Errorf("cannot use --open with --check (no diagram to show)")
}
if graphOpen && graphFormat == "dot" {
	return fmt.Errorf("cannot use --open with --format dot (use --format mermaid or omit --format)")
}
```

Add the --open rendering path after the `graphCheck` block and before the JSON output block (after line 112):

```go
// --open mode: render HTML and open in browser.
if graphOpen {
	var sb strings.Builder
	fmt.Fprintln(&sb, "graph TD;")
	id := func(path string) string {
		r := strings.NewReplacer("/", "_", ".", "_", "-", "_", " ", "_")
		return r.Replace(path)
	}
	for path := range g.Nodes {
		fmt.Fprintf(&sb, "  %s[%q];\n", id(path), path)
	}
	for _, edges := range g.Edges {
		for _, e := range edges {
			fmt.Fprintf(&sb, "  %s --> |%s| %s;\n", id(e.Source), e.Type, id(e.Target))
		}
	}

	htmlPath, err := graph.RenderHTML(sb.String())
	if err != nil {
		return fmt.Errorf("rendering HTML: %w", err)
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Opened: %s\n", htmlPath)
	return graph.OpenBrowser(htmlPath)
}
```

Add `"github.com/pablontiv/rootline/internal/graph"` is already imported.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/rootline/ -run "TestGraph_Open" -v`
Expected: PASS

- [ ] **Step 5: Run full test suite**

Run: `go test ./internal/graph/ ./cmd/rootline/ -v -count=1`
Expected: All existing + new tests PASS

- [ ] **Step 6: Build and manual test**

Run: `go build -o /tmp/rootline-test ./cmd/rootline/ && /tmp/rootline-test graph docs/epics/ --open`
Expected: Browser opens with Mermaid diagram. Stderr shows `Opened: /tmp/rootline-graph-XXXX.html`

- [ ] **Step 7: Commit**

```bash
git add cmd/rootline/graph.go cmd/rootline/graph_test.go
git commit -m "feat(graph): add --open flag for browser rendering"
```

### Task 4: Verification

- [ ] **Step 1: Run all tests**

Run: `go test ./... -race`
Expected: All PASS

- [ ] **Step 2: Check coverage**

Run: `go test ./internal/graph/ -coverprofile=c.out && go tool cover -func=c.out | tail -5`
Expected: Coverage stays ≥85%

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: Clean build
