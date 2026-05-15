package proposal

import "strings"

// valueMapping maps body text values (lowercase) to canonical enum values.
var valueMapping = map[string]string{
	"completada": "Completed",
	"activa":     "In Progress",
	"activo":     "In Progress",
	"pendiente":  "Pending",
}

// mapValue maps a body text value to its canonical enum equivalent.
// Returns the mapped value if found, otherwise the original.
func mapValue(val string) string {
	if mapped, ok := valueMapping[strings.ToLower(val)]; ok {
		return mapped
	}
	return val
}
