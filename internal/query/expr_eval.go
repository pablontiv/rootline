package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/pablontiv/rootline/internal/extract"
)

var exprCache = NewExprCache(64)

// CompileWhere compiles a where expression string into an expr program.
// The expression uses standard operators: ==, !=, in, contains, &&, ||.
func CompileWhere(whereExpr string) (*vm.Program, error) {
	return exprCache.Compile(whereExpr)
}

// BuildEnv constructs an expr environment from a Record.
// Frontmatter fields are promoted to top-level keys alongside path, body, and type.
// Derived fields override frontmatter (no prefix).
// If domainAliases is non-nil, each domain name is added as an alias
// pointing to the value of the corresponding field.
func BuildEnv(rec *extract.Record, domainAliases map[string]string) map[string]any {
	env := map[string]any{
		"path": rec.Path,
		"body": rec.Body,
		"type": rec.Type,
	}
	for k, v := range rec.Frontmatter {
		env[k] = v
	}
	for k, v := range rec.Derived {
		env[k] = v
	}
	if rec.Sections != nil {
		env["sections"] = rec.Sections
	}
	// Inject domain aliases: domain_name → field_value
	for domain, fieldName := range domainAliases {
		if _, exists := env[domain]; exists {
			continue // don't shadow existing fields
		}
		if val, ok := env[fieldName]; ok {
			env[domain] = val
		}
	}
	return env
}

// MatchRecord evaluates a compiled where expression against a record.
// Returns true if the record matches the expression.
// domainAliases maps domain names to field names for virtual alias resolution.
func MatchRecord(_ context.Context, program *vm.Program, rec *extract.Record, domainAliases map[string]string) (bool, error) {
	env := BuildEnv(rec, domainAliases)
	output, err := expr.Run(program, env)
	if err != nil {
		// Undefined variable access returns false (no match), not error
		if strings.Contains(err.Error(), "undefined") {
			return false, nil
		}
		return false, fmt.Errorf("evaluating where expression: %w", err)
	}
	result, ok := output.(bool)
	if !ok {
		return false, fmt.Errorf("where expression must return bool, got %T", output)
	}
	return result, nil
}

// ExecuteExpr filters records using an expr-based where expression.
// This is the expr-based alternative to Execute with Condition trees.
func ExecuteExpr(ctx context.Context, records []*extract.Record, whereExpr string, q *Query) (any, error) {
	var program *vm.Program
	if whereExpr != "" {
		var err error
		program, err = CompileWhere(whereExpr)
		if err != nil {
			return nil, err
		}
	}

	var filtered []*extract.Record
	for _, rec := range records {
		if program == nil {
			filtered = append(filtered, rec)
			continue
		}
		match, err := MatchRecord(ctx, program, rec, nil)
		if err != nil {
			return nil, err
		}
		if match {
			filtered = append(filtered, rec)
		}
	}

	if q.Count {
		return &CountResult{
			Version: 1,
			Kind:    "rootline/count",
			Meta:    QueryMeta{Count: len(filtered)},
			Count:   len(filtered),
		}, nil
	}

	if q.Limit > 0 && len(filtered) > q.Limit {
		filtered = filtered[:q.Limit]
	}

	if filtered == nil {
		filtered = []*extract.Record{}
	}

	return &QueryResult{
		Version: 1,
		Kind:    "rootline/query",
		Meta:    QueryMeta{Count: len(filtered)},
		Rows:    filtered,
	}, nil
}
