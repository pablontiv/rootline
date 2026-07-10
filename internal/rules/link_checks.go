package rules

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pablontiv/picokit/fuzzy"
	"github.com/pablontiv/rootline/internal/extract"
)

// HeadingCache caches heading slugs per target file within a validation run.
// Behavior is implemented alongside the anchors check.
type HeadingCache struct {
	slugs map[string][]string
}

// NewHeadingCache creates an empty heading cache.
func NewHeadingCache() *HeadingCache {
	return &HeadingCache{slugs: make(map[string][]string)}
}

// CheckLinks runs filesystem-backed link checks (links.checks in .stem)
// against a record's links. sourceAbsPath is the absolute path of the record
// file; relative targets resolve against its directory. Links whose style is
// not in the schema's effective styles are skipped, as are absolute targets
// (root-relative ADO form is out of scope).
func CheckLinks(links []extract.Link, schema LinkSchema, sourceAbsPath string, cache *HeadingCache) []ValidationError {
	if schema.Checks == nil {
		return nil
	}

	styles := make(map[string]bool)
	for _, s := range schema.EffectiveStyles() {
		styles[s] = true
	}

	var errs []ValidationError
	for _, link := range links {
		if !styles[linkStyle(link)] {
			continue
		}

		if schema.Checks.Encoding && strings.Contains(link.Target, " ") {
			errs = append(errs, ValidationError{
				Rule:     "link_encoding",
				Field:    "links",
				Message:  fmt.Sprintf("link target %q contains unencoded spaces (use %%20)", link.Target),
				Source:   "links.checks",
				Severity: "error",
			})
		}

		if strings.HasPrefix(link.Target, "/") {
			continue
		}

		if schema.Checks.Resolve || schema.Checks.Anchors {
			resolved, suggestion, ok := resolveCaseSensitive(filepath.Dir(sourceAbsPath), link.Target)
			if !ok {
				if schema.Checks.Resolve {
					msg := fmt.Sprintf("link target %q does not resolve to an existing file (case-sensitive)", link.Target)
					if suggestion != "" {
						msg += fmt.Sprintf(" (did you mean %q?)", suggestion)
					}
					errs = append(errs, ValidationError{
						Rule:       "link_resolve",
						Field:      "links",
						Message:    msg,
						Source:     "links.checks",
						Severity:   "error",
						Suggestion: suggestion,
					})
				}
				continue
			}
			if schema.Checks.Anchors && link.Anchor != "" {
				errs = append(errs, checkAnchor(link, resolved, cache)...)
			}
		}
	}
	return errs
}

// checkAnchor is implemented with the anchors check (see link_checks anchors task).
func checkAnchor(_ extract.Link, _ string, _ *HeadingCache) []ValidationError {
	return nil
}

// resolveCaseSensitive resolves a relative link target against baseDir,
// requiring exact-case matches for every target path component (APFS is
// case-insensitive; ADO and git are not). Directory targets resolve to their
// README.md. Returns the resolved absolute path, a fuzzy suggestion for the
// first unmatched component (may be empty), and whether resolution succeeded.
func resolveCaseSensitive(baseDir, target string) (string, string, bool) {
	decoded, err := url.PathUnescape(target)
	if err != nil {
		decoded = target
	}

	cur := baseDir
	for _, comp := range strings.Split(filepath.ToSlash(filepath.Clean(decoded)), "/") {
		switch comp {
		case "", ".":
			continue
		case "..":
			cur = filepath.Dir(cur)
			continue
		}
		entry, suggestion, ok := findEntry(cur, comp)
		if !ok {
			return "", suggestion, false
		}
		cur = filepath.Join(cur, entry)
	}

	info, err := os.Stat(cur)
	if err != nil {
		return "", "", false
	}
	if info.IsDir() {
		entry, suggestion, ok := findEntry(cur, "README.md")
		if !ok {
			return "", suggestion, false
		}
		cur = filepath.Join(cur, entry)
	}
	return cur, "", true
}

// findEntry looks for an exact-case directory entry, returning a fuzzy
// suggestion from the directory's entries when absent.
func findEntry(dir, name string) (string, string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", false
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Name() == name {
			return name, "", true
		}
		names = append(names, e.Name())
	}
	suggestion := fuzzy.Match(name, names)
	return "", suggestion, false
}
