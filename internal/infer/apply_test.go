package infer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
	"gopkg.in/yaml.v3"
)

func TestApplySchemaInferences_ExtendEnum(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  tipo:\n    type: enum\n    values: [a, b]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "enum_values", Field: "tipo", Value: "[a b c]"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	// Verify the stem was updated.
	data, _ := os.ReadFile(stemPath)
	stem, _ := rules.ParseStem(stemPath, data)

	sf := stem.Schema["tipo"]
	found := false
	for _, v := range sf.Values {
		if v == "c" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'c' in enum values, got %v", sf.Values)
	}
}

func TestApplySchemaInferences_RequiresAgentSkipped(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "required_field", Field: "estado", RequiresAgent: true, Message: "needs agent"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied for agent-required, got %d", len(result.Applied))
	}
	if len(result.Skipped) != 1 {
		t.Errorf("expected 1 skipped, got %d", len(result.Skipped))
	}
}

func TestApplySchemaInferences_AddRequired(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "required_field", Field: "estado"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(stemPath)
	stem, _ := rules.ParseStem(stemPath, data)
	if !stem.Schema["estado"].Required {
		t.Error("expected estado to be required")
	}
}

func TestApplySchemaInferences_NoModifications(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	original := []byte("version: 2\nschema:\n  estado:\n    type: string\n")
	if err := os.WriteFile(stemPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	// Empty inferences list → no modifications.
	result, err := ApplySchemaInferences(stemPath, nil, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied, got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_AddDefault(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "constant_field", Field: "estado", Value: "Pending"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(stemPath)
	if !strings.Contains(string(data), "default: Pending") {
		t.Errorf("expected 'default: Pending' in stem, got:\n%s", data)
	}
}

func TestApplySchemaInferences_SetType(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  count:\n    required: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "field_type", Field: "count", Value: "integer"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(stemPath)
	if !strings.Contains(string(data), "type: integer") {
		t.Errorf("expected 'type: integer' in stem, got:\n%s", data)
	}
}

func TestApplySchemaInferences_EnumNoValuesKey(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	// Field "tipo" exists but has no "values" key in YAML.
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  tipo:\n    type: enum\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "enum_values", Field: "tipo", Value: "[a b]"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	// The YAML node has "tipo" with "type: enum" but no "values" key.
	// The fix now correctly creates the values sequence when missing.
	if len(result.Applied) != 1 {
		t.Errorf("expected 1 applied (values sequence created), got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(stemPath)
	stem, err := rules.ParseStem(stemPath, data)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	sf := stem.Schema["tipo"]
	if len(sf.Values) != 2 {
		t.Errorf("expected 2 enum values, got %v", sf.Values)
	}
}

func TestApplySchemaInferences_UntypedField(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  mystery:\n    required: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ApplySchemaInferences(stemPath, []ReportInference{
		{Type: "untyped_field", Field: "mystery", Value: "string"},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Applied) == 0 {
		t.Fatal("expected applied changes")
	}

	data, _ := os.ReadFile(stemPath)
	if !strings.Contains(string(data), "type: string") {
		t.Errorf("expected type: string in stem, got:\n%s", data)
	}
}

func TestApplySchemaInferences_SequenceIncomplete(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  id:\n    type: sequence\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ApplySchemaInferences(stemPath, []ReportInference{
		{Type: "sequence_incomplete", Field: "id", Value: "T:3"},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Applied) == 0 {
		t.Fatal("expected applied changes")
	}

	data, _ := os.ReadFile(stemPath)
	content := string(data)
	if !strings.Contains(content, "prefix:") || !strings.Contains(content, "digits:") {
		t.Errorf("expected prefix and digits in stem, got:\n%s", content)
	}
}

func TestApplySchemaInferences_EnumNoField(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// enum_values for a field not in the schema → field is CREATED as enum.
	inferences := []ReportInference{
		{Type: "enum_values", Field: "prioridad", Value: "[alta media baja]"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied (enum field created), got %d: %v", len(result.Applied), result.Applied)
	}
	if !strings.Contains(result.Applied[0], "add_field: prioridad") {
		t.Errorf("expected add_field message, got %q", result.Applied[0])
	}

	data, _ := os.ReadFile(stemPath)
	stem, err := rules.ParseStem(stemPath, data)
	if err != nil {
		t.Fatalf("re-parse after grow: %v", err)
	}
	sf := stem.Schema["prioridad"]
	if sf.Type != "enum" {
		t.Errorf("expected created field type 'enum', got %q", sf.Type)
	}
	if len(sf.Values) != 3 {
		t.Errorf("expected 3 enum values, got %v", sf.Values)
	}
}

func TestApplySchemaInferences_EnumEmptyValue(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  tipo:\n    type: enum\n    values: [a]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Empty value list → no changes.
	inferences := []ReportInference{
		{Type: "enum_values", Field: "tipo", Value: "[]"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied for empty value list, got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_EnumAllExisting(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  tipo:\n    type: enum\n    values: [a, b]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// All values already exist → no changes.
	inferences := []ReportInference{
		{Type: "enum_values", Field: "tipo", Value: "[a b]"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied when all values exist, got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_RequiredAlreadySet(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n    required: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Field already required → no change.
	inferences := []ReportInference{
		{Type: "required_field", Field: "estado"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied (already required), got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_DefaultAlreadySet(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n    default: Done\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Field already has default → no change.
	inferences := []ReportInference{
		{Type: "constant_field", Field: "estado", Value: "Pending"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied (default exists), got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_TypeAlreadySet(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  count:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Field already has type → no change.
	inferences := []ReportInference{
		{Type: "field_type", Field: "count", Value: "integer"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied (type already set), got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_RequiredFieldNotInSchema(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// required_field for a field not in schema → field is CREATED with required: true.
	inferences := []ReportInference{
		{Type: "required_field", Field: "owner"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied (field created), got %d: %v", len(result.Applied), result.Applied)
	}
	if !strings.Contains(result.Applied[0], "add_field: owner") {
		t.Errorf("expected add_field message, got %q", result.Applied[0])
	}

	data, _ := os.ReadFile(stemPath)
	stem, err := rules.ParseStem(stemPath, data)
	if err != nil {
		t.Fatalf("re-parse after grow: %v", err)
	}
	if !stem.Schema["owner"].Required {
		t.Errorf("expected created field to be required, got %+v", stem.Schema["owner"])
	}
}

func TestApplySchemaInferences_UpdateExistingProperty(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	// Field has required: false initially, we update to true.
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n    required: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "required_field", Field: "estado"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(stemPath)
	if !strings.Contains(string(data), "required: \"true\"") && !strings.Contains(string(data), "required: 'true'") && !strings.Contains(string(data), "required: true") {
		t.Errorf("expected 'required: true' in stem, got:\n%s", data)
	}
}

func TestFindSchemaFieldNode_NoSchemaKey(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("version: 2\nother: value\n"), &doc); err != nil {
		t.Fatal(err)
	}
	node := findSchemaFieldNode(&doc, "estado")
	if node != nil {
		t.Error("expected nil for document without schema key")
	}
}

func TestFindSchemaFieldNode_SchemaNotMapping(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("version: 2\nschema: not_a_mapping\n"), &doc); err != nil {
		t.Fatal(err)
	}
	node := findSchemaFieldNode(&doc, "estado")
	if node != nil {
		t.Error("expected nil for schema that is not a mapping")
	}
}

func TestFindSchemaFieldNode_FieldNotInSchema(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("version: 2\nschema:\n  tipo:\n    type: string\n"), &doc); err != nil {
		t.Fatal(err)
	}
	node := findSchemaFieldNode(&doc, "nonexistent")
	if node != nil {
		t.Error("expected nil for field not in schema")
	}
}

func TestSetFieldProperty_NonMappingNode(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}
	// Should not panic, just return.
	setFieldProperty(node, "key", "value")
	if node.Value != "test" {
		t.Error("non-mapping node should not be modified")
	}
}

func TestSetFieldProperty_UpdateExisting(t *testing.T) {
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "type"},
			{Kind: yaml.ScalarNode, Value: "string"},
		},
	}
	setFieldProperty(node, "type", "integer")
	if node.Content[1].Value != "integer" {
		t.Errorf("expected updated value 'integer', got %q", node.Content[1].Value)
	}
}

func TestApplySchemaInferences_DefaultNotInSchema(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  tipo:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// constant_field for a field NOT in schema → field is CREATED with default.
	inferences := []ReportInference{
		{Type: "constant_field", Field: "estado", Value: "Pending"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied (field created), got %d: %v", len(result.Applied), result.Applied)
	}
	if !strings.Contains(result.Applied[0], "add_field: estado") {
		t.Errorf("expected add_field message, got %q", result.Applied[0])
	}

	data, _ := os.ReadFile(stemPath)
	stem, err := rules.ParseStem(stemPath, data)
	if err != nil {
		t.Fatalf("re-parse after grow: %v", err)
	}
	if stem.Schema["estado"].Default != "Pending" {
		t.Errorf("expected created field default 'Pending', got %q", stem.Schema["estado"].Default)
	}
}

func TestApplySchemaInferences_FieldTypeNotInSchema(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  tipo:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// field_type for a field NOT in schema → field is now CREATED.
	inferences := []ReportInference{
		{Type: "field_type", Field: "nuevo", Value: "integer"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied (field created), got %d: %v", len(result.Applied), result.Applied)
	}
	if !strings.Contains(result.Applied[0], "add_field: nuevo") {
		t.Errorf("expected add_field message, got %q", result.Applied[0])
	}

	data, _ := os.ReadFile(stemPath)
	stem, err := rules.ParseStem(stemPath, data)
	if err != nil {
		t.Fatalf("re-parse after grow: %v", err)
	}
	if stem.Schema["nuevo"].Type != "integer" {
		t.Errorf("expected created field type 'integer', got %q", stem.Schema["nuevo"].Type)
	}
}

func TestFindSchemaFieldNode_RootNotMapping(t *testing.T) {
	// A YAML document with a scalar value (not a mapping).
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("just a string"), &doc); err != nil {
		t.Fatal(err)
	}
	node := findSchemaFieldNode(&doc, "anything")
	if node != nil {
		t.Error("expected nil for non-mapping root")
	}
}

func TestEnsureSchemaFieldNode_ExistingField(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("version: 2\nschema:\n  tipo:\n    type: string\n"), &doc); err != nil {
		t.Fatal(err)
	}
	node, created := ensureSchemaFieldNode(&doc, "tipo")
	if node == nil {
		t.Fatal("expected non-nil node for existing field")
	}
	if created {
		t.Error("expected created=false for existing field")
	}
}

func TestEnsureSchemaFieldNode_CreatesField(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("version: 2\nschema:\n  tipo:\n    type: string\n"), &doc); err != nil {
		t.Fatal(err)
	}
	node, created := ensureSchemaFieldNode(&doc, "nuevo")
	if node == nil || !created {
		t.Fatalf("expected created node, got node=%v created=%v", node, created)
	}
	if node.Kind != yaml.MappingNode {
		t.Errorf("expected empty mapping node, got kind %v", node.Kind)
	}
	// The new field must be reachable via findSchemaFieldNode now.
	if findSchemaFieldNode(&doc, "nuevo") == nil {
		t.Error("created field not found in schema mapping")
	}
}

func TestEnsureSchemaFieldNode_CreatesSchemaKey(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("version: 2\n"), &doc); err != nil {
		t.Fatal(err)
	}
	node, created := ensureSchemaFieldNode(&doc, "estado")
	if node == nil || !created {
		t.Fatalf("expected created node, got node=%v created=%v", node, created)
	}
	if findSchemaFieldNode(&doc, "estado") == nil {
		t.Error("field not reachable after creating schema key")
	}
}

func TestEnsureSchemaFieldNode_SchemaNotMapping(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("version: 2\nschema: not_a_mapping\n"), &doc); err != nil {
		t.Fatal(err)
	}
	node, created := ensureSchemaFieldNode(&doc, "estado")
	if node != nil || created {
		t.Errorf("expected (nil, false) for non-mapping schema, got node=%v created=%v", node, created)
	}
}

func TestEnsureSchemaFieldNode_RootNotMapping(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("just a string"), &doc); err != nil {
		t.Fatal(err)
	}
	node, created := ensureSchemaFieldNode(&doc, "anything")
	if node != nil || created {
		t.Errorf("expected (nil, false) for non-mapping root, got node=%v created=%v", node, created)
	}
}

func TestApplySchemaInferences_ReadError(t *testing.T) {
	_, err := ApplySchemaInferences("/nonexistent/.stem", nil, false)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestApplySchemaInferences_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte(":\n  bad:\n    yaml: [unclosed"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ApplySchemaInferences(stemPath, nil, false)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestScaffoldSchema(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "doc1.md"), []byte("---\ntitle: Hello\nestado: draft\n---\n# Doc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "doc2.md"), []byte("---\ntitle: World\n---\n# Doc 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := ScaffoldSchema(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	stemPath := filepath.Join(dir, ".stem")
	data, readErr := os.ReadFile(stemPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	content := string(data)
	if !strings.Contains(content, "version: 2") {
		t.Error("expected version: 2")
	}
	if !strings.Contains(content, "estado:") {
		t.Error("expected estado field")
	}
	if !strings.Contains(content, "title:") {
		t.Error("expected title field")
	}
}

func TestScaffoldSchema_NoMarkdown(t *testing.T) {
	dir := t.TempDir()
	err := ScaffoldSchema(dir, false)
	if err == nil {
		t.Error("expected error when no markdown files found")
	}
}

func TestApplySchemaInferences_GrowMultiplePropsSameNewField(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Same NEW field referenced by three inferences in one run.
	inferences := []ReportInference{
		{Type: "field_type", Field: "prioridad", Value: "string"},
		{Type: "required_field", Field: "prioridad"},
		{Type: "constant_field", Field: "prioridad", Value: "media"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 3 {
		t.Fatalf("expected 3 applied, got %d: %v", len(result.Applied), result.Applied)
	}

	data, _ := os.ReadFile(stemPath)
	stem, err := rules.ParseStem(stemPath, data)
	if err != nil {
		t.Fatalf("re-parse after grow: %v", err)
	}
	sf := stem.Schema["prioridad"]
	if sf.Type != "string" || !sf.Required || sf.Default != "media" {
		t.Errorf("expected one node with type+required+default, got %+v", sf)
	}
}

func TestApplySchemaInferences_GrowStemWithoutSchemaKey(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	// Minimal .stem with no schema: key at all.
	if err := os.WriteFile(stemPath, []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "field_type", Field: "estado", Value: "string"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d: %v", len(result.Applied), result.Applied)
	}

	data, _ := os.ReadFile(stemPath)
	stem, err := rules.ParseStem(stemPath, data)
	if err != nil {
		t.Fatalf("re-parse after creating schema key: %v", err)
	}
	if stem.Schema["estado"].Type != "string" {
		t.Errorf("expected created field under new schema key, got %+v", stem.Schema)
	}
}

func TestApplySchemaInferences_GrowDryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	original := "version: 2\nschema:\n  estado:\n    type: string\n"
	if err := os.WriteFile(stemPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "field_type", Field: "nuevo", Value: "integer"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, true)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 1 || !result.DryRun {
		t.Fatalf("expected 1 applied + DryRun, got applied=%v dryRun=%v", result.Applied, result.DryRun)
	}

	data, _ := os.ReadFile(stemPath)
	if string(data) != original {
		t.Errorf("dry-run must not modify the file, got:\n%s", data)
	}
}

func TestApplySchemaInferences_EnumThenFieldTypeSameNewField(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// enum_values BEFORE field_type for the same new field.
	inferences := []ReportInference{
		{Type: "enum_values", Field: "prioridad", Value: "[alta media baja]"},
		{Type: "field_type", Field: "prioridad", Value: "enum"},
	}
	if _, err := ApplySchemaInferences(stemPath, inferences, false); err != nil {
		t.Fatalf("apply error: %v", err)
	}
	data, _ := os.ReadFile(stemPath)
	stem, err := rules.ParseStem(stemPath, data)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	sf := stem.Schema["prioridad"]
	if sf.Type != "enum" {
		t.Errorf("expected type enum, got %q", sf.Type)
	}
	if len(sf.Values) != 3 {
		t.Errorf("expected 3 enum values, got %v", sf.Values)
	}
}

func TestApplySchemaInferences_FieldTypeThenEnumSameNewField(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// field_type BEFORE enum_values for the same new field — the dropped-values bug.
	inferences := []ReportInference{
		{Type: "field_type", Field: "prioridad", Value: "enum"},
		{Type: "enum_values", Field: "prioridad", Value: "[alta media baja]"},
	}
	if _, err := ApplySchemaInferences(stemPath, inferences, false); err != nil {
		t.Fatalf("apply error: %v", err)
	}
	data, _ := os.ReadFile(stemPath)
	stem, err := rules.ParseStem(stemPath, data)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	sf := stem.Schema["prioridad"]
	if sf.Type != "enum" {
		t.Errorf("expected type enum, got %q", sf.Type)
	}
	if len(sf.Values) != 3 {
		t.Errorf("expected 3 enum values, got %v", sf.Values)
	}
}

// TestApplySchemaInferences_UnknownInferenceType tests that unknown inference types are rejected.
func TestApplySchemaInferences_UnknownInferenceType(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Unknown inference type → should be rejected
	inferences := []ReportInference{
		{Type: "unknown_type", Field: "estado", Value: "test"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Rejected) == 0 {
		t.Error("expected rejected[] to contain unknown inference type")
	}
	if len(result.Applied) > 0 {
		t.Errorf("expected no applied for unknown type, got %d", len(result.Applied))
	}

	// Verify the rejection message mentions the unknown type
	found := false
	for _, msg := range result.Rejected {
		if strings.Contains(msg, "unknown_type") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("rejection message should mention 'unknown_type', got: %v", result.Rejected)
	}
}

// TestApplySchemaInferences_MixedKnownAndUnknown tests that known types are applied and unknown types are rejected in one run.
func TestApplySchemaInferences_MixedKnownAndUnknown(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Mix of known and unknown types
	inferences := []ReportInference{
		{Type: "required_field", Field: "estado"},
		{Type: "unknown_op", Field: "estado", Value: "test"},
		{Type: "field_type", Field: "nuevo", Value: "integer"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	// Should have some applied (known types) and some rejected (unknown types)
	if len(result.Applied) == 0 {
		t.Error("expected some applied for known types")
	}
	if len(result.Rejected) == 0 {
		t.Error("expected some rejected for unknown types")
	}

	// Verify the file was modified for the known types
	data, _ := os.ReadFile(stemPath)
	stem, err := rules.ParseStem(stemPath, data)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if !stem.Schema["estado"].Required {
		t.Error("expected estado to be required (from known inference)")
	}
	if stem.Schema["nuevo"].Type != "integer" {
		t.Error("expected nuevo field with type integer (from known inference)")
	}
}
