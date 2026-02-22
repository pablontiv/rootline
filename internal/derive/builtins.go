package derive

import (
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/expr-lang/expr"
)

var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a string to a URL-friendly slug.
// Lowercase, replace spaces and special chars with hyphens, strip edges.
func slugify(s string) string {
	lower := strings.ToLower(s)
	slug := nonAlphanumRe.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

// BuiltinOptions returns expr.Option values that register all builtin
// functions in the expr compilation environment.
func BuiltinOptions() []expr.Option {
	return []expr.Option{
		expr.Function("slugify",
			func(params ...any) (any, error) {
				return slugify(params[0].(string)), nil
			},
			new(func(string) string),
		),
		expr.Function("lower",
			func(params ...any) (any, error) {
				return strings.ToLower(params[0].(string)), nil
			},
			new(func(string) string),
		),
		expr.Function("upper",
			func(params ...any) (any, error) {
				return strings.ToUpper(params[0].(string)), nil
			},
			new(func(string) string),
		),
		expr.Function("trim",
			func(params ...any) (any, error) {
				return strings.TrimFunc(params[0].(string), unicode.IsSpace), nil
			},
			new(func(string) string),
		),
		expr.Function("strlen",
			func(params ...any) (any, error) {
				return len(params[0].(string)), nil
			},
			new(func(string) int),
		),
		expr.Function("concat",
			func(params ...any) (any, error) {
				parts := make([]string, len(params))
				for i, p := range params {
					parts[i] = toString(p)
				}
				return strings.Join(parts, ""), nil
			},
			new(func(...any) string),
		),
	}
}

// toString converts any value to its string representation.
func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	rv := reflect.ValueOf(v)
	return strings.TrimRight(strings.TrimRight(
		strings.Replace(rv.String(), "<", "", 1), ">"), " ")
}

// NewEvaluatorWithBuiltins creates an Evaluator pre-loaded with all builtins.
// This is the recommended constructor for derivation use cases.
func NewEvaluatorWithBuiltins(extra ...expr.Option) *Evaluator {
	opts := BuiltinOptions()
	opts = append(opts, extra...)
	return NewEvaluator(opts...)
}
