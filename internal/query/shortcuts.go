package query

import "strings"

// defaultShortcuts maps common field names to their full paths.
var defaultShortcuts = map[string]string{
	"path": "path",
	"body": "body",
	"type": "type",
}

// ResolveShortcut expands a field shortcut to its full dot-path.
// Known top-level fields (path, body, type) stay as-is.
// Fields already containing a dot are treated as literal paths.
// All other bare names are expanded to "frontmatter.<name>".
func ResolveShortcut(field string) string {
	// Already a dot-path — pass through
	if strings.Contains(field, ".") {
		return field
	}

	// Known top-level fields
	if resolved, ok := defaultShortcuts[field]; ok {
		return resolved
	}

	// Bare name → frontmatter shortcut
	return "frontmatter." + field
}

// resolveConditionShortcuts recursively resolves field shortcuts
// in a condition tree.
func resolveConditionShortcuts(cond *Condition) {
	if cond == nil {
		return
	}
	cond.Field = ResolveShortcut(cond.Field)
	for i := range cond.Children {
		resolveConditionShortcuts(&cond.Children[i])
	}
}
