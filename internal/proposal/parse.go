package proposal

import (
	"regexp"
	"strings"
)

var blockedByRe = regexp.MustCompile(`^(.+?)\s*\(blocked by\s+(.+)\)$`)
var idLikeRe = regexp.MustCompile(`^[A-Z]\d{2,}([/-][A-Z]\d{2,})*$`)

// ParseBlockingInfo extracts structured blocking information from a value
// like "Pending (blocked by T001 + E04/F05 + human)".
// Returns the base value, ID-like targets, and free-text notes.
func ParseBlockingInfo(value string) (base string, targets []string, notes []string) {
	m := blockedByRe.FindStringSubmatch(value)
	if m == nil {
		return value, nil, nil
	}

	base = strings.TrimSpace(m[1])
	parts := strings.Split(m[2], "+")

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if idLikeRe.MatchString(p) {
			targets = append(targets, p)
		} else {
			notes = append(notes, p)
		}
	}
	return base, targets, notes
}
