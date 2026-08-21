package infer

import (
	"fmt"
	"os"
	"sort"

	"github.com/pablontiv/rootline/internal/fsx"
	"github.com/pablontiv/rootline/internal/rules"
	"gopkg.in/yaml.v3"
)

// SchemaInferencePlan describes the candidate bytes produced by schema
// inference planning. Planning reads and validates the prospective .stem state
// but does not persist it.
type SchemaInferencePlan struct {
	Target   string
	Content  []byte
	Result   ApplyResult
	Modified bool
}

// PlanSchemaInferences plans schema-modifying inferences for a .stem file
// without writing to disk. It performs the same read, parse, YAML-node mutation,
// marshal, and local declaration validation work used by ApplySchemaInferences.
func PlanSchemaInferences(stemPath string, inferences []ReportInference) (*SchemaInferencePlan, error) {
	data, err := os.ReadFile(stemPath)
	if err != nil {
		return nil, fmt.Errorf("reading stem: %w", err)
	}

	// Parse into struct for reading current state.
	stem, err := rules.ParseStem(stemPath, data)
	if err != nil {
		return nil, fmt.Errorf("parsing stem: %w", err)
	}

	// Parse into yaml.Node for preserving-format writes.
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing yaml node: %w", err)
	}

	inferences = normalizeSchemaInferences(inferences)

	sectionPlan, err := buildSectionApplyPlan(stem, inferences)
	if err != nil {
		return nil, err
	}

	result := ApplyResult{}
	modified := false
	changedFields := map[string]bool{}

	for _, inf := range inferences {
		if inf.RequiresAgent {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s: %s (requires agent)", inf.Type, inf.Message))
			continue
		}

		switch inf.Type {
		case "enum_values":
			if applied, created := applyEnumExtensionNode(&doc, stem, inf); applied {
				if created {
					result.Applied = append(result.Applied, fmt.Sprintf("add_field: %s (enum)", inf.Field))
				} else {
					result.Applied = append(result.Applied, fmt.Sprintf("extend_enum: %s", inf.Field))
				}
				modified = true
				changedFields[inf.Field] = true
			}

		case "required_field":
			if applied, created := applyRequiredFieldNode(&doc, stem, inf); applied {
				if created {
					result.Applied = append(result.Applied, fmt.Sprintf("add_field: %s (required)", inf.Field))
				} else {
					result.Applied = append(result.Applied, fmt.Sprintf("add_required: %s", inf.Field))
				}
				modified = true
				changedFields[inf.Field] = true
			}

		case "constant_field":
			if applied, created := applyDefaultValueNode(&doc, stem, inf); applied {
				if created {
					result.Applied = append(result.Applied, fmt.Sprintf("add_field: %s (default=%s)", inf.Field, inf.Value))
				} else {
					result.Applied = append(result.Applied, fmt.Sprintf("add_default: %s=%s", inf.Field, inf.Value))
				}
				modified = true
				changedFields[inf.Field] = true
			}

		case "field_type", "untyped_field":
			if applied, created := applyFieldTypeNode(&doc, stem, inf); applied {
				if created {
					result.Applied = append(result.Applied, fmt.Sprintf("add_field: %s (type=%s)", inf.Field, inf.Value))
				} else {
					result.Applied = append(result.Applied, fmt.Sprintf("set_type: %s=%s", inf.Field, inf.Value))
				}
				modified = true
				changedFields[inf.Field] = true
			}

		case "sequence_incomplete":
			if applySequenceCompleteNode(&doc, inf) {
				result.Applied = append(result.Applied, fmt.Sprintf("sequence: %s completed", inf.Field))
				modified = true
				changedFields[inf.Field] = true
			}

		case "required_section", "optional_section":
			continue

		default:
			// Drift guard: this fires only if schema.go's routing filter admits a type this switch doesn't handle (wiring bug).
			result.Rejected = append(result.Rejected, fmt.Sprintf("%s: unknown inference type", inf.Type))
		}
	}

	for _, field := range sortedSectionPlanFields(sectionPlan) {
		if applied, created := applySectionFieldNode(&doc, sectionPlan[field]); applied {
			if created {
				result.Applied = append(result.Applied, fmt.Sprintf("add_field: %s (section)", field))
			} else {
				result.Applied = append(result.Applied, fmt.Sprintf("merge_section: %s", field))
			}
			modified = true
			changedFields[field] = true
		}
	}

	if !modified {
		return &SchemaInferencePlan{
			Target:   stemPath,
			Content:  append([]byte(nil), data...),
			Result:   copyApplyResult(result),
			Modified: false,
		}, nil
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return nil, fmt.Errorf("marshaling stem: %w", err)
	}
	if err := validateProspectiveChangedFields(stemPath, out, changedFields); err != nil {
		return nil, err
	}

	return &SchemaInferencePlan{
		Target:   stemPath,
		Content:  out,
		Result:   copyApplyResult(result),
		Modified: true,
	}, nil
}

// ApplySchemaInferencePlan persists a planned schema inference candidate. It is
// intentionally the only persistence boundary for inference plans.
func ApplySchemaInferencePlan(plan *SchemaInferencePlan) error {
	if plan == nil || !plan.Modified {
		return nil
	}
	if err := fsx.WriteFileAtomic(plan.Target, plan.Content, 0o644); err != nil {
		return fmt.Errorf("writing stem: %w", err)
	}
	return nil
}

func copyApplyResult(result ApplyResult) ApplyResult {
	result.Applied = append([]string(nil), result.Applied...)
	result.Skipped = append([]string(nil), result.Skipped...)
	result.Rejected = append([]string(nil), result.Rejected...)
	sort.Strings(result.Applied)
	sort.Strings(result.Skipped)
	sort.Strings(result.Rejected)
	return result
}

func normalizeSchemaInferences(inferences []ReportInference) []ReportInference {
	out := append([]ReportInference(nil), inferences...)
	sort.SliceStable(out, func(i, j int) bool {
		return compareSchemaInference(out[i], out[j]) < 0
	})
	return out
}

func compareSchemaInference(a, b ReportInference) int {
	keysA := []string{a.Field, fmt.Sprintf("%02d", schemaInferenceTypeRank(a.Type)), a.Type, a.Value, a.Source, a.SourceDirective, a.Message, fmt.Sprint(a.RequiresAgent)}
	keysB := []string{b.Field, fmt.Sprintf("%02d", schemaInferenceTypeRank(b.Type)), b.Type, b.Value, b.Source, b.SourceDirective, b.Message, fmt.Sprint(b.RequiresAgent)}
	for i := range keysA {
		if keysA[i] < keysB[i] {
			return -1
		}
		if keysA[i] > keysB[i] {
			return 1
		}
	}
	return 0
}

func schemaInferenceTypeRank(inferenceType string) int {
	switch inferenceType {
	case "field_type", "untyped_field":
		return 10
	case "enum_values":
		return 20
	case "sequence_incomplete":
		return 30
	case "constant_field":
		return 40
	case "required_field":
		return 50
	case "required_section", "optional_section":
		return 60
	default:
		return 90
	}
}

func validateProspectiveChangedFields(stemPath string, content []byte, changedFields map[string]bool) error {
	stem, err := rules.ParseStem(stemPath, content)
	if err != nil {
		return fmt.Errorf("validating prospective .stem %s: %w", stemPath, err)
	}
	fields := make([]string, 0, len(changedFields))
	for field := range changedFields {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	for _, field := range fields {
		declaration, ok := stem.Schema[field]
		if !ok {
			return fmt.Errorf("validating prospective .stem %s: field %q missing after schema apply", stemPath, field)
		}
		if issues := rules.ValidateFieldDeclaration(field, declaration); len(issues) > 0 {
			return fmt.Errorf("validating prospective .stem %s: field %q: %s", stemPath, field, issues[0].Message)
		}
	}
	return nil
}
