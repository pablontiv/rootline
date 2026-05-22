package query

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/pablontiv/picokit/fuzzy"
)

// FieldWarning reports an unknown field name in a where expression with an optional suggestion.
type FieldWarning struct {
	Field      string `json:"field"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// CheckFieldNames compiles the expression without AllowUndefinedVariables
// against the known field set. Any undefined variable triggers a warning
// with a fuzzy suggestion. Because the expr compiler reports only the first
// unknown name per compilation, we iteratively add discovered unknowns to
// the environment and recompile until no more errors appear.
func CheckFieldNames(whereExpr string, knownFields []string) []FieldWarning {
	// Build a dummy env with all known fields set to empty string.
	env := make(map[string]any, len(knownFields))
	for _, f := range knownFields {
		env[f] = ""
	}

	var warnings []FieldWarning
	seen := make(map[string]bool)

	for {
		_, err := expr.Compile(whereExpr, expr.AsBool(), expr.Env(env))
		if err == nil {
			break
		}

		name := extractUnknownName(err.Error())
		if name == "" || seen[name] {
			break
		}
		seen[name] = true

		// Add the unknown field to the env so the next compile can find further unknowns.
		env[name] = ""

		w := FieldWarning{Field: name}
		suggestion := fuzzy.Match(name, knownFields)
		if suggestion != "" {
			w.Suggestion = suggestion
			w.Message = fmt.Sprintf("unknown field %q in where expression (did you mean %q?)", name, suggestion)
		} else {
			w.Message = fmt.Sprintf("unknown field %q in where expression", name)
		}
		warnings = append(warnings, w)
	}

	return warnings
}

// extractUnknownName pulls the field name from an expr error of the form
// "unknown name FIELD (LINE:COL)\n | ...".
func extractUnknownName(errMsg string) string {
	// Only look at the first line.
	line := errMsg
	if idx := strings.Index(errMsg, "\n"); idx >= 0 {
		line = errMsg[:idx]
	}
	line = strings.TrimSpace(line)

	const prefix = "unknown name "
	idx := strings.Index(line, prefix)
	if idx < 0 {
		return ""
	}
	rest := line[idx+len(prefix):]
	// Field name ends at space or paren.
	end := strings.IndexAny(rest, " (")
	if end < 0 {
		return rest
	}
	return rest[:end]
}
