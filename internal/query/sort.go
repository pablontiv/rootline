package query

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// SortKey represents a single sort criterion.
type SortKey struct {
	Field string
	Desc  bool
}

// ParseSortKeys parses a comma-separated sort specification string.
// Format: "field1:asc,field2:desc,field3" (direction defaults to asc).
func ParseSortKeys(spec string) ([]SortKey, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}

	parts := strings.Split(spec, ",")
	keys := make([]SortKey, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		segments := strings.SplitN(part, ":", 3)
		if len(segments) > 2 {
			return nil, fmt.Errorf("invalid sort key %q: too many colons", part)
		}

		field := strings.TrimSpace(segments[0])
		if field == "" {
			return nil, fmt.Errorf("invalid sort key %q: empty field name", part)
		}

		desc := false
		if len(segments) == 2 {
			dir := strings.TrimSpace(strings.ToLower(segments[1]))
			switch dir {
			case "asc":
				desc = false
			case "desc":
				desc = true
			default:
				return nil, fmt.Errorf("invalid sort direction %q in key %q: must be asc or desc", dir, part)
			}
		}

		keys = append(keys, SortKey{Field: field, Desc: desc})
	}

	return keys, nil
}

// SortRecords sorts records in-place by the given sort keys.
// schema may be nil if no enum ordering is needed.
// Uses sort.SliceStable for deterministic ordering of equal elements.
func SortRecords(records []*extract.Record, keys []SortKey, schema map[string]rules.SchemaField) {
	if len(keys) == 0 || len(records) < 2 {
		return
	}

	// Pre-build enum index maps for fields that are enum type with values.
	enumIndexes := buildEnumIndexes(keys, schema)

	sort.SliceStable(records, func(i, j int) bool {
		for _, key := range keys {
			// Check nil/missing first -- nil always sorts last regardless of direction.
			_, okA := records[i].EffectiveField(key.Field)
			_, okB := records[j].EffectiveField(key.Field)
			if !okA && !okB {
				continue
			}
			if !okA {
				return false // a is nil, sorts after b
			}
			if !okB {
				return true // b is nil, sorts after a
			}

			cmp := compareField(records[i], records[j], key.Field, enumIndexes[key.Field])
			if cmp == 0 {
				continue
			}
			if key.Desc {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
}

// buildEnumIndexes creates value-to-position maps for enum sort keys.
func buildEnumIndexes(keys []SortKey, schema map[string]rules.SchemaField) map[string]map[string]int {
	indexes := make(map[string]map[string]int)
	if schema == nil {
		return indexes
	}
	for _, key := range keys {
		sf, ok := schema[key.Field]
		if !ok || sf.Type != "enum" || len(sf.Values) == 0 {
			continue
		}
		idx := make(map[string]int, len(sf.Values))
		for i, v := range sf.Values {
			idx[v] = i
		}
		indexes[key.Field] = idx
	}
	return indexes
}

// compareField compares two records on a single field.
// Returns -1, 0, or 1.
// nil/missing values always compare as "greater" (sort last).
func compareField(a, b *extract.Record, field string, enumIndex map[string]int) int {
	va, okA := a.EffectiveField(field)
	vb, okB := b.EffectiveField(field)

	// Handle nil/missing: nil sorts last (regardless of direction -- caller handles inversion).
	if !okA && !okB {
		return 0
	}
	if !okA {
		return 1 // a is nil, sorts after b
	}
	if !okB {
		return -1 // b is nil, sorts after a
	}

	// Enum ordering: if we have an index map, use positional comparison.
	if enumIndex != nil {
		sa := fmt.Sprintf("%v", va)
		sb := fmt.Sprintf("%v", vb)
		ia, foundA := enumIndex[sa]
		ib, foundB := enumIndex[sb]
		if !foundA {
			ia = len(enumIndex) // unknown values sort after known
		}
		if !foundB {
			ib = len(enumIndex)
		}
		if ia < ib {
			return -1
		}
		if ia > ib {
			return 1
		}
		return 0
	}

	// Numeric comparison: try to parse both as float64.
	fa, numA := toFloat64(va)
	fb, numB := toFloat64(vb)
	if numA && numB {
		if fa < fb {
			return -1
		}
		if fa > fb {
			return 1
		}
		return 0
	}

	// String fallback: lexicographic comparison.
	sa := fmt.Sprintf("%v", va)
	sb := fmt.Sprintf("%v", vb)
	return strings.Compare(sa, sb)
}

// toFloat64 attempts to extract a float64 from a value.
// Handles int, float64, and string representations.
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		return f, err == nil
	}
	return 0, false
}
