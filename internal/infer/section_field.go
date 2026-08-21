package infer

import (
	"strings"
	"unicode"
)

// sectionFieldName converts heading text to the detector-owned logical field name.
func sectionFieldName(heading string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.TrimSpace(heading) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			lastUnderscore = false
		default:
			if b.Len() > 0 && !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	field := strings.Trim(b.String(), "_")
	if field == "" {
		return "section"
	}
	return field
}
