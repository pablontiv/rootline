package derive

import (
	"context"
	"fmt"
	"sort"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// deriveConfig holds optional configuration for DeriveRecord.
type deriveConfig struct {
	resolver RecordResolver
}

// DeriveOption configures optional behavior for DeriveRecord.
type DeriveOption func(*deriveConfig)

// WithResolver sets a RecordResolver for resolving linked field values.
func WithResolver(r RecordResolver) DeriveOption {
	return func(c *deriveConfig) {
		c.resolver = r
	}
}

// DeriveError records a non-fatal derivation failure for a single field.
type DeriveError struct {
	Field      string `json:"field"`
	Expression string `json:"expression"`
	Message    string `json:"message"`
}

func (e *DeriveError) Error() string {
	return fmt.Sprintf("derive %s (%s): %s", e.Field, e.Expression, e.Message)
}

// DeriveRecord evaluates all derive expressions from the effective .stem
// against a record's frontmatter and optional children.
//
// Each derive entry maps a field name to an expression string. The expression
// is compiled and evaluated with the record's frontmatter as environment,
// plus a "children" variable if provided. If a RecordResolver is provided,
// linked field values from wiki-links are also injected into the environment
// based on the stem's LinkSchema rules. Results are stored in record.Derived.
//
// A single field failure does not block other fields — errors are collected
// and returned together.
func DeriveRecord(_ context.Context, record *extract.Record, effective *rules.StemFile, children []*extract.Record, opts ...DeriveOption) (map[string]any, error) {
	if effective == nil || len(effective.Derive) == 0 {
		return map[string]any{}, nil
	}

	// Apply options.
	var cfg deriveConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	ev := NewEvaluatorWithBuiltins()
	env := BuildEnv(record.Frontmatter)

	// Inject linked field values from wiki-links if a resolver is available.
	InjectLinkedFields(env, record, effective, cfg.resolver)

	// Add children as []any for expr compatibility.
	if children != nil {
		childMaps := make([]any, len(children))
		for i, c := range children {
			childMaps[i] = childToMap(c)
		}
		env["children"] = childMaps
	}

	derived := make(map[string]any)
	var errs []DeriveError

	// Process fields in sorted order for deterministic output.
	fields := make([]string, 0, len(effective.Derive))
	for k := range effective.Derive {
		fields = append(fields, k)
	}
	sort.Strings(fields)

	for _, field := range fields {
		exprVal := effective.Derive[field]
		exprStr, ok := exprVal.(string)
		if !ok {
			errs = append(errs, DeriveError{
				Field:      field,
				Expression: fmt.Sprintf("%v", exprVal),
				Message:    fmt.Sprintf("expected string expression, got %T", exprVal),
			})
			continue
		}

		compiled, err := ev.Compile(exprStr)
		if err != nil {
			errs = append(errs, DeriveError{
				Field:      field,
				Expression: exprStr,
				Message:    err.Error(),
			})
			continue
		}

		result, err := ev.Eval(compiled, env)
		if err != nil {
			errs = append(errs, DeriveError{
				Field:      field,
				Expression: exprStr,
				Message:    err.Error(),
			})
			continue
		}

		derived[field] = result
	}

	// Populate record.Derived.
	record.Derived = derived

	if len(errs) > 0 {
		return derived, fmt.Errorf("derivation errors: %v", errs)
	}
	return derived, nil
}

// childToMap converts a child Record to a map for use in expressions.
func childToMap(c *extract.Record) map[string]any {
	m := make(map[string]any, len(c.Frontmatter)+2)
	for k, v := range c.Frontmatter {
		m[k] = v
	}
	m["path"] = c.Path
	m["type"] = c.Type
	return m
}
