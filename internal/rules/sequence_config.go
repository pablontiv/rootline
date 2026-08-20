package rules

import (
	"fmt"
	"sort"
	"strings"
)

type sequenceConfigError struct {
	actual string
	detail string
}

func (e sequenceConfigError) Error() string {
	if e.detail == "" {
		return e.actual
	}
	return e.detail
}

func applySequenceConfig(field SchemaField, config map[string]any) (SchemaField, error) {
	if config == nil {
		return field, sequenceConfigError{actual: "malformed sequence config", detail: "config must be a map"}
	}
	if unknown := unknownSequenceConfigKeys(config); len(unknown) > 0 {
		return field, sequenceConfigError{
			actual: "unsupported sequence config",
			detail: fmt.Sprintf("unsupported sequence config key(s): %s", strings.Join(unknown, ", ")),
		}
	}

	resolved := field
	if value, ok := config["prefix"]; ok {
		prefix, ok := value.(string)
		if !ok || prefix == "" {
			return field, sequenceConfigError{actual: "incomplete sequence config", detail: "prefix must be a non-empty string"}
		}
		resolved.Prefix = prefix
	}
	if value, ok := config["digits"]; ok {
		digits, ok := strictConfigDigits(value)
		if !ok || digits <= 0 {
			return field, sequenceConfigError{actual: "incomplete sequence config", detail: "digits must be a positive integer"}
		}
		resolved.Digits = digits
	}
	if resolved.Prefix == "" {
		return field, sequenceConfigError{actual: "incomplete sequence config", detail: "prefix must be effective and non-empty"}
	}
	if resolved.Digits <= 0 {
		return field, sequenceConfigError{actual: "incomplete sequence config", detail: "digits must be effective and positive"}
	}
	return resolved, nil
}

func unknownSequenceConfigKeys(config map[string]any) []string {
	unknown := make([]string, 0)
	for key := range config {
		switch key {
		case "prefix", "digits":
			continue
		default:
			unknown = append(unknown, key)
		}
	}
	sort.Strings(unknown)
	return unknown
}

func strictConfigDigits(value any) (int, bool) {
	maxInt := uint64(^uint(0) >> 1)
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		if int64(int(v)) != v {
			return 0, false
		}
		return int(v), true
	case uint:
		if uint64(v) > maxInt {
			return 0, false
		}
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		if uint64(v) > maxInt {
			return 0, false
		}
		return int(v), true
	case uint64:
		if v > maxInt {
			return 0, false
		}
		return int(v), true
	default:
		return 0, false
	}
}
