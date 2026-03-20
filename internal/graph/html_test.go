package graph

import (
	"os"
	"strings"
	"testing"
)

func TestRenderHTML_ContainsMermaidContent(t *testing.T) {
	mermaidContent := "graph TD;\n  A[\"node-a\"];\n  B[\"node-b\"];\n  A --> |ref| B;\n"

	htmlPath, err := RenderHTML(mermaidContent)
	if err != nil {
		t.Fatalf("RenderHTML returned error: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(htmlPath) })

	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", htmlPath, err)
	}
	html := string(data)

	// Mermaid content must be embedded
	if !strings.Contains(html, mermaidContent) {
		t.Errorf("expected HTML to contain mermaid content, got:\n%s", html)
	}

	// CDN script must be present
	if !strings.Contains(html, "cdn.jsdelivr.net/npm/mermaid") {
		t.Errorf("expected CDN script tag in HTML, got:\n%s", html)
	}

	// Viewport meta must be present
	if !strings.Contains(html, `name="viewport"`) {
		t.Errorf("expected viewport meta tag in HTML, got:\n%s", html)
	}
}

func TestRenderHTML_CreatesValidHTMLFile(t *testing.T) {
	mermaidContent := "graph TD;\n  X[\"x\"];\n"

	htmlPath, err := RenderHTML(mermaidContent)
	if err != nil {
		t.Fatalf("RenderHTML returned error: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(htmlPath) })

	// Must have .html suffix
	if !strings.HasSuffix(htmlPath, ".html") {
		t.Errorf("expected .html suffix in path %q", htmlPath)
	}

	// File must be non-empty
	info, err := os.Stat(htmlPath)
	if err != nil {
		t.Fatalf("Stat %s: %v", htmlPath, err)
	}
	if info.Size() == 0 {
		t.Errorf("expected non-empty HTML file, got size 0")
	}
}
