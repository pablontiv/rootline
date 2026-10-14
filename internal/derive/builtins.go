package derive

import (
	"fmt"
	"regexp"
	"strconv"
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
				s, ok := params[0].(string)
				if !ok {
					return nil, fmt.Errorf("slugify: expected string, got %T", params[0])
				}
				return slugify(s), nil
			},
			new(func(string) string),
		),
		expr.Function("lower",
			func(params ...any) (any, error) {
				s, ok := params[0].(string)
				if !ok {
					return nil, fmt.Errorf("lower: expected string, got %T", params[0])
				}
				return strings.ToLower(s), nil
			},
			new(func(string) string),
		),
		expr.Function("upper",
			func(params ...any) (any, error) {
				s, ok := params[0].(string)
				if !ok {
					return nil, fmt.Errorf("upper: expected string, got %T", params[0])
				}
				return strings.ToUpper(s), nil
			},
			new(func(string) string),
		),
		expr.Function("trim",
			func(params ...any) (any, error) {
				s, ok := params[0].(string)
				if !ok {
					return nil, fmt.Errorf("trim: expected string, got %T", params[0])
				}
				return strings.TrimFunc(s, unicode.IsSpace), nil
			},
			new(func(string) string),
		),
		expr.Function("strlen",
			func(params ...any) (any, error) {
				s, ok := params[0].(string)
				if !ok {
					return nil, fmt.Errorf("strlen: expected string, got %T", params[0])
				}
				return len(s), nil
			},
			new(func(string) int),
		),
		expr.Function("concat",
			func(params ...any) (any, error) {
				var result strings.Builder
				var digits [20]byte
				for _, p := range params {
					switch v := p.(type) {
					case nil:
						// Preserve nil as empty text.
					case string:
						result.WriteString(v)
					case bool:
						result.WriteString(strconv.FormatBool(v))
					case int:
						result.Write(strconv.AppendInt(digits[:0], int64(v), 10))
					case int8:
						result.Write(strconv.AppendInt(digits[:0], int64(v), 10))
					case int16:
						result.Write(strconv.AppendInt(digits[:0], int64(v), 10))
					case int32:
						result.Write(strconv.AppendInt(digits[:0], int64(v), 10))
					case int64:
						result.Write(strconv.AppendInt(digits[:0], v, 10))
					case uint:
						result.Write(strconv.AppendUint(digits[:0], uint64(v), 10))
					case uint8:
						result.Write(strconv.AppendUint(digits[:0], uint64(v), 10))
					case uint16:
						result.Write(strconv.AppendUint(digits[:0], uint64(v), 10))
					case uint32:
						result.Write(strconv.AppendUint(digits[:0], uint64(v), 10))
					case uint64:
						result.Write(strconv.AppendUint(digits[:0], v, 10))
					case uintptr:
						result.Write(strconv.AppendUint(digits[:0], uint64(v), 10))
					default:
						return nil, fmt.Errorf("concat: unsupported argument type %T", p)
					}
				}
				return result.String(), nil
			},
			new(func(...any) string),
		),
	}
}

// NewEvaluatorWithBuiltins creates an Evaluator pre-loaded with all builtins.
// This is the recommended constructor for derivation use cases.
func NewEvaluatorWithBuiltins(extra ...expr.Option) *Evaluator {
	opts := BuiltinOptions()
	opts = append(opts, extra...)
	return NewEvaluator(opts...)
}
