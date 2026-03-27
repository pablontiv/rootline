package query

import (
	"fmt"
	"strings"
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
