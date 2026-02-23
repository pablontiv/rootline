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

// InferEstado infers a parent estado from child estados.
// All Completed → Completed, all Pending → Pending, mixed → In Progress.
// Empty slice → Pending.
func InferEstado(childEstados []string) string {
	if len(childEstados) == 0 {
		return "Pending"
	}

	allSame := true
	first := childEstados[0]
	for _, e := range childEstados[1:] {
		if e != first {
			allSame = false
			break
		}
	}

	if allSame {
		return first
	}
	return "In Progress"
}
