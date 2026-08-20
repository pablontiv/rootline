package rules

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

// NormalizeValidationSources returns a cloned error slice with public document
// validation sources projected relative to the record's governance root.
func NormalizeValidationSources(errs []ValidationError, governanceRoot string) ([]ValidationError, error) {
	root, err := cleanAbsoluteGovernanceRoot(governanceRoot)
	if err != nil {
		return nil, err
	}

	out := slices.Clone(errs)
	for i := range out {
		source := out[i].Source
		symbolic, err := isSymbolicValidationSource(source)
		if err != nil {
			return nil, err
		}
		if symbolic {
			continue
		}
		if !isPathBackedValidationSource(source) {
			return nil, fmt.Errorf("unsupported validation source %q", source)
		}

		rel, err := normalizeValidationSource(source, root)
		if err != nil {
			return nil, err
		}
		out[i].Source = rel
	}
	return out, nil
}

func normalizeValidationSource(source, governanceRoot string) (string, error) {
	if hasWindowsDrive(source) && !filepath.IsAbs(source) {
		return "", fmt.Errorf("validation source outside governance root")
	}
	if isWindowsUNC(source) {
		return "", fmt.Errorf("validation source outside governance root")
	}

	if filepath.IsAbs(source) {
		rel, err := filepath.Rel(governanceRoot, source)
		if err != nil || escapesRoot(rel) {
			return "", fmt.Errorf("validation source outside governance root")
		}
		return normalizePublicPath(rel), nil
	}

	rel := path.Clean(normalizePublicPath(source))
	if escapesRoot(rel) {
		return "", fmt.Errorf("validation source outside governance root")
	}
	return rel, nil
}

func cleanAbsoluteGovernanceRoot(governanceRoot string) (string, error) {
	if governanceRoot == "" || hasWindowsDrive(governanceRoot) && !filepath.IsAbs(governanceRoot) || isWindowsUNC(governanceRoot) {
		return "", fmt.Errorf("invalid governance root")
	}
	clean := filepath.Clean(governanceRoot)
	if clean == "." || !filepath.IsAbs(clean) {
		return "", fmt.Errorf("invalid governance root")
	}
	return clean, nil
}

func isSymbolicValidationSource(source string) (bool, error) {
	if source == "" {
		return true, nil
	}
	switch source {
	case "schema", "scope", "links.checks":
		return true, nil
	}
	if strings.HasPrefix(source, "body.") {
		if _, err := extract.ParseBodySource(source); err != nil {
			return false, fmt.Errorf("unsupported validation source %q", source)
		}
		return true, nil
	}
	return false, nil
}

func isPathBackedValidationSource(source string) bool {
	if filepath.IsAbs(source) || hasWindowsDrive(source) || isWindowsUNC(source) || strings.ContainsAny(source, `/\\`) {
		return true
	}
	slashed := normalizePublicPath(source)
	return slashed == ".stem" || strings.HasSuffix(slashed, ".md") || strings.HasSuffix(slashed, ".markdown")
}

func normalizePublicPath(p string) string {
	p = strings.ReplaceAll(filepath.ToSlash(p), `\`, "/")
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	return p
}

func escapesRoot(rel string) bool {
	rel = path.Clean(normalizePublicPath(rel))
	return rel == ".." || strings.HasPrefix(rel, "../") || strings.HasPrefix(rel, "/")
}

func hasWindowsDrive(p string) bool {
	if len(p) < 2 {
		return false
	}
	c := p[0]
	return ((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) && p[1] == ':'
}

func isWindowsUNC(p string) bool {
	return strings.HasPrefix(p, `\\`)
}
