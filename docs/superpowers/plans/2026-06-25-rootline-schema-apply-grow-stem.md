# Schema Apply Grow-Stem Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `schema apply` grow a `.stem` by creating schema field nodes for newly-observed fields instead of silently dropping inferences that target absent fields.

**Architecture:** All changes are concentrated in `internal/infer/apply.go`. A new `ensureSchemaFieldNode` helper creates the `schema:` mapping and/or the field node when absent, returning a `created` flag. The four property handlers (`field_type`/`untyped_field`, `enum_values`, `required_field`, `constant_field`) switch from "find-or-give-up" to "ensure-and-populate" and return `(applied, created)` so the caller can distinguish `add_field` (new) from refinement messages. `sequence_incomplete` is intentionally unchanged.

**Tech Stack:** Go 1.25+, `gopkg.in/yaml.v3` (yaml.Node manipulation for format-preserving writes), standard `testing` package.

## Global Constraints

- Go 1.25+; no new third-party dependencies.
- `just check` (gofmt + golangci-lint + build) must pass; `golangci-lint`'s `unused` check counts test usage, so every new unexported function must be referenced by a test or production code in the same task.
- `just test` runs with `-race`; 0 failures.
- Coverage floor for `internal/infer/` ≥ 85% (`.coverage-floors.toml`).
- Conventional Commits enforced by `.githooks/commit-msg`.
- `.stem` files are v2-only; tests use `version: 2` headers.
- Do NOT touch `.stem` resolution, the `Record` type, or governance detectors (Claim 2 is explicitly out of scope).

---

## File Structure

- **Modify:** `internal/infer/apply.go` — add `ensureSchemaFieldNode`; change 4 handler signatures to `(bool, bool)`; update the dispatch `switch` in `ApplySchemaInferences` for observability.
- **Modify:** `internal/infer/apply_test.go` — flip 4 contract tests; add growth + helper + integration tests.
- **Modify (docs, Task 6):** `CLAUDE.md` — note that `schema apply` now creates newly-observed fields.

---

## Task 1: `ensureSchemaFieldNode` helper

**Files:**
- Modify: `internal/infer/apply.go` (add helper after `findSchemaFieldNode`, ~line 139)
- Test: `internal/infer/apply_test.go` (add helper tests near the existing `TestFindSchemaFieldNode_*` block)

**Interfaces:**
- Produces: `ensureSchemaFieldNode(doc *yaml.Node, fieldName string) (*yaml.Node, bool)` — returns the field's mapping node and a `created` flag. Creates the `schema:` mapping if absent and the field mapping if absent. Returns `(nil, false)` only when the root or an existing `schema:` value is not a mapping.

- [ ] **Step 1: Write the failing tests**

Add to `internal/infer/apply_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infer/ -run TestEnsureSchemaFieldNode -v`
Expected: FAIL — `undefined: ensureSchemaFieldNode`.

- [ ] **Step 3: Implement the helper**

Add to `internal/infer/apply.go` immediately after `findSchemaFieldNode` (after line 139):

```go
// ensureSchemaFieldNode navigates doc → schema → fieldName, creating the
// schema mapping and/or the field's mapping node if absent. The returned bool
// is true when the field node was newly created. Returns (nil, false) only when
// the root or an existing schema value is not a mapping.
func ensureSchemaFieldNode(doc *yaml.Node, fieldName string) (*yaml.Node, bool) {
	root := doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return nil, false
	}

	// Find or create the "schema" key.
	var schemaNode *yaml.Node
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "schema" {
			schemaNode = root.Content[i+1]
			break
		}
	}
	if schemaNode == nil {
		schemaNode = &yaml.Node{Kind: yaml.MappingNode}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "schema"},
			schemaNode,
		)
	}
	if schemaNode.Kind != yaml.MappingNode {
		return nil, false
	}

	// Find an existing field node.
	for i := 0; i < len(schemaNode.Content)-1; i += 2 {
		if schemaNode.Content[i].Value == fieldName {
			return schemaNode.Content[i+1], false
		}
	}

	// Create a new empty field mapping.
	fieldNode := &yaml.Node{Kind: yaml.MappingNode}
	schemaNode.Content = append(schemaNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: fieldName},
		fieldNode,
	)
	return fieldNode, true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infer/ -run TestEnsureSchemaFieldNode -v`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/infer/apply.go internal/infer/apply_test.go
git commit -m "feat(infer): add ensureSchemaFieldNode helper for schema growth"
```

---

## Task 2: `field_type` / `untyped_field` create absent fields

**Files:**
- Modify: `internal/infer/apply.go` (`applyFieldTypeNode` ~line 237; dispatch `switch` cases for `field_type` and `untyped_field` ~lines 70-80)
- Test: `internal/infer/apply_test.go` (flip `TestApplySchemaInferences_FieldTypeNotInSchema` ~line 492; add growth test)

**Interfaces:**
- Consumes: `ensureSchemaFieldNode` (Task 1).
- Produces: `applyFieldTypeNode(doc *yaml.Node, stem *rules.StemFile, inf ReportInference) (bool, bool)` — `(applied, created)`.

- [ ] **Step 1: Flip the contract test and add a growth test**

In `internal/infer/apply_test.go`, REPLACE the body of `TestApplySchemaInferences_FieldTypeNotInSchema` (currently asserts 0 applied) with:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infer/ -run TestApplySchemaInferences_FieldTypeNotInSchema -v`
Expected: FAIL — got 0 applied (current code drops the inference).

- [ ] **Step 3: Implement growth + merge the two switch cases**

In `internal/infer/apply.go`, REPLACE `applyFieldTypeNode` (lines 237-250) with:

```go
func applyFieldTypeNode(doc *yaml.Node, stem *rules.StemFile, inf ReportInference) (bool, bool) {
	sf, ok := stem.Schema[inf.Field]
	if ok && sf.Type != "" {
		return false, false
	}

	fieldNode, created := ensureSchemaFieldNode(doc, inf.Field)
	if fieldNode == nil {
		return false, false
	}

	setFieldProperty(fieldNode, "type", inf.Value)
	return true, created
}
```

REPLACE the two separate `case "field_type":` and `case "untyped_field":` blocks (lines 70-80) with a single merged case:

```go
		case "field_type", "untyped_field":
			if applied, created := applyFieldTypeNode(&doc, stem, inf); applied {
				if created {
					result.Applied = append(result.Applied, fmt.Sprintf("add_field: %s (type=%s)", inf.Field, inf.Value))
				} else {
					result.Applied = append(result.Applied, fmt.Sprintf("set_type: %s=%s", inf.Field, inf.Value))
				}
				modified = true
			}
```

- [ ] **Step 4: Run the focused and package tests**

Run: `go test ./internal/infer/ -run TestApplySchemaInferences_FieldType -v && go test ./internal/infer/ -run 'TestApplySchemaInferences_SetType|TestApplySchemaInferences_UntypedField|TestApplySchemaInferences_TypeAlreadySet' -v`
Expected: PASS — growth test passes; existing refine/already-set tests still pass.

- [ ] **Step 5: Commit**

```bash
git add internal/infer/apply.go internal/infer/apply_test.go
git commit -m "feat(schema): apply field_type creates absent .stem fields"
```

---

## Task 3: `enum_values` creates absent enum fields

**Files:**
- Modify: `internal/infer/apply.go` (`applyEnumExtensionNode` ~line 159; dispatch `case "enum_values":` ~line 52)
- Test: `internal/infer/apply_test.go` (flip `TestApplySchemaInferences_EnumNoField` ~line 241; add growth test)

**Interfaces:**
- Consumes: `ensureSchemaFieldNode` (Task 1), `parseValueList` (existing).
- Produces: `applyEnumExtensionNode(doc *yaml.Node, stem *rules.StemFile, inf ReportInference) (bool, bool)`.

- [ ] **Step 1: Flip the contract test and add a growth test**

In `internal/infer/apply_test.go`, REPLACE the body of `TestApplySchemaInferences_EnumNoField` with:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infer/ -run TestApplySchemaInferences_EnumNoField -v`
Expected: FAIL — got 0 applied.

- [ ] **Step 3: Implement growth**

In `internal/infer/apply.go`, REPLACE `applyEnumExtensionNode` (lines 159-205) with:

```go
func applyEnumExtensionNode(doc *yaml.Node, stem *rules.StemFile, inf ReportInference) (bool, bool) {
	inferredValues := parseValueList(inf.Value)
	if len(inferredValues) == 0 {
		return false, false
	}

	// Zero value when the field is absent → empty existing set → all inferred are new.
	sf := stem.Schema[inf.Field]
	existing := make(map[string]bool)
	for _, v := range sf.Values {
		existing[v] = true
	}

	var newValues []string
	for _, v := range inferredValues {
		if !existing[v] {
			newValues = append(newValues, v)
		}
	}
	if len(newValues) == 0 {
		return false, false
	}

	fieldNode, created := ensureSchemaFieldNode(doc, inf.Field)
	if fieldNode == nil {
		return false, false
	}

	if created {
		// New field: declare it as an enum with a fresh values sequence.
		setFieldProperty(fieldNode, "type", "enum")
		valuesNode := &yaml.Node{Kind: yaml.SequenceNode}
		for _, v := range newValues {
			valuesNode.Content = append(valuesNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: v},
			)
		}
		fieldNode.Content = append(fieldNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "values"},
			valuesNode,
		)
		return true, true
	}

	// Existing field: append to its values sequence.
	for i := 0; i < len(fieldNode.Content)-1; i += 2 {
		if fieldNode.Content[i].Value == "values" {
			valuesNode := fieldNode.Content[i+1]
			if valuesNode.Kind == yaml.SequenceNode {
				for _, v := range newValues {
					valuesNode.Content = append(valuesNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: v},
					)
				}
				return true, false
			}
		}
	}
	return false, false
}
```

REPLACE the dispatch `case "enum_values":` block (lines 52-56) with:

```go
		case "enum_values":
			if applied, created := applyEnumExtensionNode(&doc, stem, inf); applied {
				if created {
					result.Applied = append(result.Applied, fmt.Sprintf("add_field: %s (enum)", inf.Field))
				} else {
					result.Applied = append(result.Applied, fmt.Sprintf("extend_enum: %s", inf.Field))
				}
				modified = true
			}
```

- [ ] **Step 4: Run focused + regression tests**

Run: `go test ./internal/infer/ -run 'TestApplySchemaInferences_Enum|TestApplySchemaInferences_ExtendEnum' -v`
Expected: PASS — `_EnumNoField` (grows), `_ExtendEnum` (refines existing), `_EnumEmptyValue` (0 applied), `_EnumAllExisting` (0 applied) all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/infer/apply.go internal/infer/apply_test.go
git commit -m "feat(schema): apply enum_values creates absent enum fields"
```

---

## Task 4: `required_field` creates absent fields

**Files:**
- Modify: `internal/infer/apply.go` (`applyRequiredFieldNode` ~line 207; dispatch `case "required_field":` ~line 58)
- Test: `internal/infer/apply_test.go` (flip `TestApplySchemaInferences_RequiredFieldNotInSchema` ~line 367; add growth test)

**Interfaces:**
- Consumes: `ensureSchemaFieldNode` (Task 1).
- Produces: `applyRequiredFieldNode(doc *yaml.Node, stem *rules.StemFile, inf ReportInference) (bool, bool)`.

- [ ] **Step 1: Flip the contract test and add a growth test**

In `internal/infer/apply_test.go`, REPLACE the body of `TestApplySchemaInferences_RequiredFieldNotInSchema` with:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infer/ -run TestApplySchemaInferences_RequiredFieldNotInSchema -v`
Expected: FAIL — got 0 applied.

- [ ] **Step 3: Implement growth**

In `internal/infer/apply.go`, REPLACE `applyRequiredFieldNode` (lines 207-220) with:

```go
func applyRequiredFieldNode(doc *yaml.Node, stem *rules.StemFile, inf ReportInference) (bool, bool) {
	sf, ok := stem.Schema[inf.Field]
	if ok && sf.Required {
		return false, false
	}

	fieldNode, created := ensureSchemaFieldNode(doc, inf.Field)
	if fieldNode == nil {
		return false, false
	}

	setFieldProperty(fieldNode, "required", "true")
	return true, created
}
```

REPLACE the dispatch `case "required_field":` block (lines 58-62) with:

```go
		case "required_field":
			if applied, created := applyRequiredFieldNode(&doc, stem, inf); applied {
				if created {
					result.Applied = append(result.Applied, fmt.Sprintf("add_field: %s (required)", inf.Field))
				} else {
					result.Applied = append(result.Applied, fmt.Sprintf("add_required: %s", inf.Field))
				}
				modified = true
			}
```

- [ ] **Step 4: Run focused + regression tests**

Run: `go test ./internal/infer/ -run 'TestApplySchemaInferences_RequiredField|TestApplySchemaInferences_AddRequired|TestApplySchemaInferences_RequiredAlreadySet|TestApplySchemaInferences_UpdateExistingProperty' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/infer/apply.go internal/infer/apply_test.go
git commit -m "feat(schema): apply required_field creates absent fields"
```

---

## Task 5: `constant_field` default creates absent fields

**Files:**
- Modify: `internal/infer/apply.go` (`applyDefaultValueNode` ~line 222; dispatch `case "constant_field":` ~line 64)
- Test: `internal/infer/apply_test.go` (flip `TestApplySchemaInferences_DefaultNotInSchema` ~line 471; add growth test)

**Interfaces:**
- Consumes: `ensureSchemaFieldNode` (Task 1).
- Produces: `applyDefaultValueNode(doc *yaml.Node, stem *rules.StemFile, inf ReportInference) (bool, bool)`.

- [ ] **Step 1: Flip the contract test and add a growth test**

In `internal/infer/apply_test.go`, REPLACE the body of `TestApplySchemaInferences_DefaultNotInSchema` with:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infer/ -run TestApplySchemaInferences_DefaultNotInSchema -v`
Expected: FAIL — got 0 applied.

- [ ] **Step 3: Implement growth**

In `internal/infer/apply.go`, REPLACE `applyDefaultValueNode` (lines 222-235) with:

```go
func applyDefaultValueNode(doc *yaml.Node, stem *rules.StemFile, inf ReportInference) (bool, bool) {
	sf, ok := stem.Schema[inf.Field]
	if ok && sf.Default != "" {
		return false, false
	}

	fieldNode, created := ensureSchemaFieldNode(doc, inf.Field)
	if fieldNode == nil {
		return false, false
	}

	setFieldProperty(fieldNode, "default", inf.Value)
	return true, created
}
```

REPLACE the dispatch `case "constant_field":` block (lines 64-68) with:

```go
		case "constant_field":
			if applied, created := applyDefaultValueNode(&doc, stem, inf); applied {
				if created {
					result.Applied = append(result.Applied, fmt.Sprintf("add_field: %s (default=%s)", inf.Field, inf.Value))
				} else {
					result.Applied = append(result.Applied, fmt.Sprintf("add_default: %s=%s", inf.Field, inf.Value))
				}
				modified = true
			}
```

- [ ] **Step 4: Run focused + full package tests**

Run: `go test ./internal/infer/ -run 'TestApplySchemaInferences_Default|TestApplySchemaInferences_AddDefault' -v && go test ./internal/infer/ -count=1`
Expected: PASS — focused tests pass and the whole `internal/infer` package is green.

- [ ] **Step 5: Commit**

```bash
git add internal/infer/apply.go internal/infer/apply_test.go
git commit -m "feat(schema): apply constant_field default creates absent fields"
```

---

## Task 6: Integration (multi-inference, no-schema-key, validate) + docs

**Files:**
- Test: `internal/infer/apply_test.go` (add 3 integration tests)
- Modify: `CLAUDE.md` (schema apply description)

**Interfaces:**
- Consumes: `ApplySchemaInferences` (now growing), `rules.ParseStem`.

- [ ] **Step 1: Write the integration tests**

Add to `internal/infer/apply_test.go`:

```go
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
```

- [ ] **Step 2: Run the integration tests**

Run: `go test ./internal/infer/ -run 'TestApplySchemaInferences_Grow' -v`
Expected: PASS (3 tests).

- [ ] **Step 3: Update CLAUDE.md schema apply description**

In `CLAUDE.md`, find the sentence describing `schema apply` (search for `applies only schema-surface proposals (create_stem, update_stem) to`). Append a clause so it reads:

```
... applies only schema-surface proposals (create_stem, update_stem) to `.stem` files; update_stem now grows the `.stem` by creating field nodes for newly-observed fields (field_type/enum_values/required_field/constant_field), closing the `analyze --incremental` → `schema apply` loop; auto-detects report kind; ...
```

(Keep the rest of the sentence intact — only insert the `update_stem now grows...` clause after `to `.stem` files;`.)

- [ ] **Step 4: Run full suite + check + coverage**

Run: `just test && just check`
Expected: all packages PASS with `-race`; gofmt clean; lint clean; build OK.

Run: `go test ./internal/infer/ -coverprofile=/tmp/infer.cov && go tool cover -func=/tmp/infer.cov | tail -1`
Expected: `internal/infer` total coverage ≥ 85%.

- [ ] **Step 5: Commit**

```bash
git add internal/infer/apply_test.go CLAUDE.md
git commit -m "feat(schema): close analyze --incremental to schema apply loop"
```

---

## Self-Review (completed by plan author)

**Spec coverage:**
- `ensureSchemaFieldNode` helper (spec §"Helper nuevo") → Task 1. ✓
- `field_type`/`untyped_field` grow (spec table) → Task 2. ✓
- `enum_values` grow + `type: enum` (spec "enum sobre campo nuevo DEBE setear type:enum") → Task 3. ✓
- `required_field` grow → Task 4. ✓
- `constant_field` grow → Task 5. ✓
- `sequence_incomplete` unchanged (spec table) → not modified; no task touches it. ✓
- Ordering / stale `stem.Schema` (spec §"Ordenamiento") → Task 6 multi-prop test. ✓
- Observability `add_field` vs refine (spec §"Observabilidad") → message assertions in Tasks 2-5. ✓
- `.stem` without `schema:` key (spec §"ensureSchemaFieldNode") → Task 6 test. ✓
- Dry-run preserved (spec §"Dry-run y round-trip") → Task 6 test. ✓
- Round-trip re-parse (spec §"Dry-run y round-trip") → every growth test calls `rules.ParseStem`. ✓
- 4 flipped tests (spec §"Tests a flipear") → Tasks 2-5. ✓
- Coverage ≥85% + just check/test (spec §"Verificación") → Task 6. ✓
- Out of scope: Claim 2 governance — no task touches `analyze.go` or governance detectors. ✓

**Placeholder scan:** none — every step has full code or exact commands.

**Type consistency:** all four handlers use the uniform signature `(doc *yaml.Node, stem *rules.StemFile, inf ReportInference) (bool, bool)`; the caller consumes `(applied, created)` consistently; `ensureSchemaFieldNode` returns `(*yaml.Node, bool)` everywhere it is referenced.
