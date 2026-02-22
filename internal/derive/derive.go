// Package derive evaluates derivation expressions defined in .stem files.
//
// A derivation expression is a string (e.g. "slugify(titulo)") that gets
// compiled once and evaluated against a Record's frontmatter to produce
// computed fields. The package wraps expr-lang/expr with an environment
// builder and contextual error handling.
package derive

import (
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// CompiledExpr holds a pre-compiled expression and its source text.
type CompiledExpr struct {
	Program    *vm.Program
	Expression string
}

// Evaluator compiles and runs derivation expressions.
type Evaluator struct {
	options []expr.Option
}

// NewEvaluator creates an Evaluator with default options.
// Additional expr.Option values can be passed to customise compilation.
func NewEvaluator(opts ...expr.Option) *Evaluator {
	defaults := []expr.Option{
		expr.AllowUndefinedVariables(),
	}
	return &Evaluator{options: append(defaults, opts...)}
}

// Compile parses and compiles an expression string.
func (e *Evaluator) Compile(expression string) (*CompiledExpr, error) {
	program, err := expr.Compile(expression, e.options...)
	if err != nil {
		return nil, fmt.Errorf("compiling derive expression %q: %w", expression, err)
	}
	return &CompiledExpr{Program: program, Expression: expression}, nil
}

// Eval runs a compiled expression against the given environment.
func (e *Evaluator) Eval(compiled *CompiledExpr, env map[string]any) (any, error) {
	result, err := expr.Run(compiled.Program, env)
	if err != nil {
		return nil, fmt.Errorf("evaluating derive expression %q: %w", compiled.Expression, err)
	}
	return result, nil
}

// BuildEnv constructs an evaluation environment from frontmatter fields.
// All frontmatter keys are promoted to top-level variables so expressions
// can reference them directly (e.g. "titulo" instead of "frontmatter.titulo").
func BuildEnv(frontmatter map[string]any) map[string]any {
	env := make(map[string]any, len(frontmatter))
	for k, v := range frontmatter {
		env[k] = v
	}
	return env
}
