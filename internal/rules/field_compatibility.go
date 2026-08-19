package rules

import "fmt"

type FieldCompatibilityIssue struct {
	Constraint string
	Operation  string
	Value      any
	Message    string
}

func CheckFieldCompatibility(parent, child SchemaField) []FieldCompatibilityIssue {
	var issues []FieldCompatibilityIssue

	if issue, ok := checkFieldSourceCompatibility(parent, child); ok {
		issues = append(issues, issue)
	}
	if issue, ok := checkFieldTypeCompatibility(parent, child); ok {
		issues = append(issues, issue)
	}
	if issue, ok := checkFieldValuesCompatibility(parent, child); ok {
		issues = append(issues, issue)
	}
	if issue, ok := checkFieldRequiredCompatibility(parent, child); ok {
		issues = append(issues, issue)
	}
	if issue, ok := checkFieldSeverityCompatibility(parent, child); ok {
		issues = append(issues, issue)
	}

	return issues
}

func checkFieldSourceCompatibility(parent, child SchemaField) (FieldCompatibilityIssue, bool) {
	childSource, declared := childFieldSource(child)
	if !declared {
		return FieldCompatibilityIssue{}, false
	}
	if childSource == parent.Extract {
		return FieldCompatibilityIssue{}, false
	}
	if childSource == "" {
		return FieldCompatibilityIssue{
			Constraint: "source",
			Operation:  "removal",
			Value:      childSource,
			Message:    fmt.Sprintf("source removes inherited source %q", parent.Extract),
		}, true
	}
	return FieldCompatibilityIssue{
		Constraint: "source",
		Operation:  "change",
		Value:      childSource,
		Message:    fmt.Sprintf("source changes from %q to %q", parent.Extract, childSource),
	}, true
}

func childFieldSource(field SchemaField) (string, bool) {
	if field.Extract != "" {
		return field.Extract, true
	}
	if field.declaration.SourcePresent {
		return "", true
	}
	return "", false
}

func checkFieldTypeCompatibility(parent, child SchemaField) (FieldCompatibilityIssue, bool) {
	if parent.Type == child.Type || (parent.Type == "string" && child.Type == "enum") {
		return FieldCompatibilityIssue{}, false
	}
	return FieldCompatibilityIssue{
		Constraint: "type",
		Operation:  "widening",
		Value:      child.Type,
		Message:    fmt.Sprintf("type changes from %q to %q", parent.Type, child.Type),
	}, true
}

func checkFieldValuesCompatibility(parent, child SchemaField) (FieldCompatibilityIssue, bool) {
	if parent.Type != "enum" || child.Type != "enum" || len(parent.Values) == 0 || len(child.Values) == 0 {
		return FieldCompatibilityIssue{}, false
	}
	added := compatibilityAddedValues(child.Values, parent.Values)
	if len(added) == 0 {
		return FieldCompatibilityIssue{}, false
	}
	return FieldCompatibilityIssue{
		Constraint: "values",
		Operation:  "extension",
		Value:      added,
		Message:    fmt.Sprintf("enum extends inherited values with %v", added),
	}, true
}

func compatibilityAddedValues(child, parent []string) []string {
	parentValues := make(map[string]struct{}, len(parent))
	for _, value := range parent {
		parentValues[value] = struct{}{}
	}
	var added []string
	for _, value := range child {
		if _, ok := parentValues[value]; !ok {
			added = append(added, value)
		}
	}
	return added
}

func checkFieldRequiredCompatibility(parent, child SchemaField) (FieldCompatibilityIssue, bool) {
	if !parent.Required || child.Required || child.RequiredMatch != nil {
		return FieldCompatibilityIssue{}, false
	}
	return FieldCompatibilityIssue{
		Constraint: "required",
		Operation:  "loosening",
		Value:      false,
		Message:    "required loosens from true to false",
	}, true
}

func checkFieldSeverityCompatibility(parent, child SchemaField) (FieldCompatibilityIssue, bool) {
	parentSeverity, parentOK := normalizedSeverity(parent.Severity)
	childSeverity, childOK := normalizedSeverity(child.Severity)
	if !parentOK || !childOK || severityOrder[childSeverity] >= severityOrder[parentSeverity] {
		return FieldCompatibilityIssue{}, false
	}
	return FieldCompatibilityIssue{
		Constraint: "severity",
		Operation:  "loosening",
		Value:      childSeverity,
		Message:    fmt.Sprintf("severity loosens from %q to %q", parentSeverity, childSeverity),
	}, true
}

func normalizedSeverity(severity string) (string, bool) {
	if severity == "" {
		severity = "error"
	}
	_, ok := severityOrder[severity]
	return severity, ok
}
