package graph

import (
	"embed"
	"html/template"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

//go:embed templates/graph.html
var templateFS embed.FS

// osCreateTemp is a variable to allow mocking in tests.
var osCreateTemp = os.CreateTemp

// RenderHTML renders mermaidContent into a temporary HTML file using the
// embedded graph.html template and returns the path to the created file.
func RenderHTML(mermaidContent string) (string, error) {
	html, err := RenderHTMLContent(mermaidContent)
	if err != nil {
		return "", err
	}

	f, err := osCreateTemp("", "rootline-graph-*.html")
	if err != nil {
		return "", err
	}

	if _, err := f.WriteString(html); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name()) //nolint:gosec
		return "", err
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name()) //nolint:gosec
		return "", err
	}

	return f.Name(), nil
}

// RenderHTMLContent renders mermaidContent into an HTML string using the
// embedded graph.html template.
func RenderHTMLContent(mermaidContent string) (string, error) {
	tmplData, err := templateFS.ReadFile("templates/graph.html")
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("graph").Parse(string(tmplData))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	data := struct{ Content template.HTML }{Content: template.HTML(mermaidContent)} //nolint:gosec
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", err
	}

	return sb.String(), nil
}

// OpenBrowser opens the file at path in the default browser.
// It uses xdg-open on Linux, open on macOS, and cmd /c start on Windows.
func OpenBrowser(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path) //nolint:gosec
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", path) //nolint:gosec
	default: // linux and others
		cmd = exec.Command("xdg-open", path) //nolint:gosec
	}
	return cmd.Start()
}
