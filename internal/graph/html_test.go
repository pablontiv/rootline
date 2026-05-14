package graph

import (
	"errors"
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

func TestRenderHTML_HTMLStructure(t *testing.T) {
	htmlPath, err := RenderHTML("graph TD;")
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(htmlPath) })

	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	html := string(data)

	for _, expected := range []string{"<!DOCTYPE html>", "<html>", "</html>", "mermaid.initialize"} {
		if !strings.Contains(html, expected) {
			t.Errorf("HTML missing %q", expected)
		}
	}
}

func TestOpenBrowser_NonExistentFile(t *testing.T) {
	// OpenBrowser starts a process async — on CI with no display,
	// xdg-open or equivalent will fail but Start() may still succeed.
	// We just verify the function doesn't panic.
	_ = OpenBrowser("/tmp/nonexistent-rootline-test-file.html")
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

func TestRenderHTMLContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		contains []string
	}{
		{
			name:    "simple graph",
			content: "graph TD; A-->B;",
			contains: []string{
				"graph TD; A-->B;",
				"<!DOCTYPE html>",
				"mermaid.initialize",
			},
		},
		{
			name:    "empty content",
			content: "",
			contains: []string{
				"<pre class=\"mermaid\"></pre>",
			},
		},
		{
			name:    "with html-like characters",
			content: "graph TD; A[\"<b>Bold</b>\"] --> B;",
			contains: []string{
				"graph TD; A[\"<b>Bold</b>\"] --> B;",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderHTMLContent(tt.content)
			if err != nil {
				t.Fatalf("RenderHTMLContent() error = %v", err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("RenderHTMLContent() = %v, want to contain %v", got, want)
				}
			}
		})
	}
}

func TestRenderHTML_ErrorPaths(t *testing.T) {
	// Backup and restore the mockable function
	oldCreateTemp := osCreateTemp
	t.Cleanup(func() { osCreateTemp = oldCreateTemp })

	// Mock os.CreateTemp to return an error
	osCreateTemp = func(dir, pattern string) (*os.File, error) {
		return nil, errors.New("mocked error")
	}

	_, err := RenderHTML("graph TD; A-->B;")
	if err == nil {
		t.Fatal("expected error from RenderHTML, got nil")
	}
	if !strings.Contains(err.Error(), "mocked error") {
		t.Errorf("expected error containing 'mocked error', got: %v", err)
	}
}
