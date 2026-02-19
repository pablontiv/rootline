// Package query implements the query engine for filtering records.
//
// It supports field-based filtering with operators (eq, ne, contains, etc.)
// and produces JSON output conforming to versioned contracts.
package query

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

// Operator is a query condition operator.
type Operator string

const (
	OpEq       Operator = "eq"
	OpNe       Operator = "ne"
	OpIn       Operator = "in"
	OpContains Operator = "contains"
	OpExists   Operator = "exists"
	OpAnd      Operator = "and"
)

// Condition represents a single filter condition.
type Condition struct {
	Op       Operator
	Field    string
	Value    string   // for eq, ne, contains
	Values   []string // for in
	Children []Condition // for and
}

// Query defines a query against a set of records.
type Query struct {
	Where *Condition
	Limit int
	Count bool
}

// QueryResult is the versioned JSON output for a standard query.
type QueryResult struct {
	Version int              `json:"version"`
	Kind    string           `json:"kind"`
	Meta    QueryMeta        `json:"meta"`
	Rows    []*extract.Record `json:"rows"`
}

// QueryMeta holds metadata about the query result.
type QueryMeta struct {
	Count int `json:"count"`
}

// CountResult is the versioned JSON output for a count query.
type CountResult struct {
	Version int       `json:"version"`
	Kind    string    `json:"kind"`
	Meta    QueryMeta `json:"meta"`
	Count   int       `json:"count"`
}

// Execute filters records against a query and returns the result.
// Field shortcuts are resolved before evaluation.
func Execute(records []*extract.Record, q *Query) (any, error) {
	// Resolve shortcuts in condition tree
	resolveConditionShortcuts(q.Where)

	var filtered []*extract.Record

	for _, rec := range records {
		if q.Where == nil || matches(rec, q.Where) {
			filtered = append(filtered, rec)
		}
	}

	if q.Count {
		return &CountResult{
			Version: 1,
			Kind:    "rootline/count",
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

// ToJSON serializes a query result (QueryResult or CountResult) to JSON.
func ToJSON(result any) ([]byte, error) {
	return json.Marshal(result)
}

// matches checks if a record satisfies a condition.
func matches(rec *extract.Record, cond *Condition) bool {
	switch cond.Op {
	case OpEq:
		return matchEq(rec, cond.Field, cond.Value)
	case OpNe:
		return matchNe(rec, cond.Field, cond.Value)
	case OpIn:
		return matchIn(rec, cond.Field, cond.Values)
	case OpContains:
		return matchContains(rec, cond.Field, cond.Value)
	case OpExists:
		return matchExists(rec, cond.Field)
	case OpAnd:
		return matchAnd(rec, cond.Children)
	default:
		return false
	}
}

func matchEq(rec *extract.Record, field, value string) bool {
	val, ok := resolveField(rec, field)
	if !ok {
		return false // missing field = no match
	}
	// Array field: match if contains value
	if arr, ok := toStringSlice(val); ok {
		for _, item := range arr {
			if item == value {
				return true
			}
		}
		return false
	}
	return fmt.Sprintf("%v", val) == value
}

func matchNe(rec *extract.Record, field, value string) bool {
	val, ok := resolveField(rec, field)
	if !ok {
		return true // missing field != value
	}
	return fmt.Sprintf("%v", val) != value
}

func matchIn(rec *extract.Record, field string, values []string) bool {
	val, ok := resolveField(rec, field)
	if !ok {
		return false
	}
	// Array field: match if intersection
	if arr, ok := toStringSlice(val); ok {
		for _, item := range arr {
			for _, v := range values {
				if item == v {
					return true
				}
			}
		}
		return false
	}
	s := fmt.Sprintf("%v", val)
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}

func matchContains(rec *extract.Record, field, substr string) bool {
	if field == "body" {
		return strings.Contains(rec.Body, substr)
	}
	val, ok := resolveField(rec, field)
	if !ok {
		return false
	}
	// Array field: match if any element contains substring
	if arr, ok := toStringSlice(val); ok {
		for _, item := range arr {
			if strings.Contains(item, substr) {
				return true
			}
		}
		return false
	}
	return strings.Contains(fmt.Sprintf("%v", val), substr)
}

func matchExists(rec *extract.Record, field string) bool {
	_, ok := resolveField(rec, field)
	return ok
}

func matchAnd(rec *extract.Record, children []Condition) bool {
	for i := range children {
		if !matches(rec, &children[i]) {
			return false
		}
	}
	return true
}

// resolveField gets a value from a record by field path.
// Supports "path", "body", "type", and frontmatter fields.
func resolveField(rec *extract.Record, field string) (any, bool) {
	switch field {
	case "path":
		return rec.Path, true
	case "body":
		return rec.Body, true
	case "type":
		return rec.Type, true
	}

	// Try frontmatter directly
	if val, ok := rec.Frontmatter[field]; ok {
		return val, true
	}

	// Try dot-path (e.g., "frontmatter.estado")
	if strings.HasPrefix(field, "frontmatter.") {
		key := field[len("frontmatter."):]
		if val, ok := rec.Frontmatter[key]; ok {
			return val, true
		}
	}

	return nil, false
}

// toStringSlice attempts to convert an any to []string.
func toStringSlice(val any) ([]string, bool) {
	switch v := val.(type) {
	case []any:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result, true
	case []string:
		return v, true
	}
	return nil, false
}
