package infer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestPlanSchemaInferencesReturnsCandidateWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	original := []byte("version: 2\nschema:\n  status:\n    type: enum\n    values: [Pending]\n")
	if err := os.WriteFile(stemPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(stemPath, 0o600); err != nil {
		t.Fatal(err)
	}
	infoBefore, err := os.Stat(stemPath)
	if err != nil {
		t.Fatal(err)
	}

	plan, err := PlanSchemaInferences(stemPath, []ReportInference{
		{Type: "enum_values", Field: "status", Value: "[Pending Done]"},
	})
	if err != nil {
		t.Fatalf("PlanSchemaInferences: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if plan.Target != stemPath {
		t.Fatalf("target = %q, want %q", plan.Target, stemPath)
	}
	if !plan.Modified {
		t.Fatal("expected Modified=true")
	}
	if len(plan.Result.Applied) != 1 || !strings.Contains(plan.Result.Applied[0], "extend_enum: status") {
		t.Fatalf("applied actions = %v", plan.Result.Applied)
	}

	after, err := os.ReadFile(stemPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(original) {
		t.Fatalf("planning wrote to target:\nbefore:\n%s\nafter:\n%s", original, after)
	}
	infoAfter, err := os.Stat(stemPath)
	if err != nil {
		t.Fatal(err)
	}
	if infoAfter.Mode().Perm() != infoBefore.Mode().Perm() {
		t.Fatalf("planning changed mode: got %v want %v", infoAfter.Mode().Perm(), infoBefore.Mode().Perm())
	}

	stem, err := rules.ParseStem(stemPath, plan.Content)
	if err != nil {
		t.Fatalf("candidate parse: %v", err)
	}
	values := stem.Schema["status"].Values
	if !containsString(values, "Done") {
		t.Fatalf("candidate values = %v, want Done", values)
	}
}

func TestPlanSchemaInferencesNoModificationReturnsOriginalTargetAndModifiedFalse(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	original := []byte("version: 2\nschema:\n  status:\n    type: enum\n    values: [Pending, Done]\n")
	if err := os.WriteFile(stemPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanSchemaInferences(stemPath, []ReportInference{
		{Type: "enum_values", Field: "status", Value: "[Pending Done]"},
	})
	if err != nil {
		t.Fatalf("PlanSchemaInferences: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if plan.Target != stemPath {
		t.Fatalf("target = %q, want %q", plan.Target, stemPath)
	}
	if plan.Modified {
		t.Fatal("expected Modified=false")
	}
	if string(plan.Content) != string(original) {
		t.Fatalf("content =\n%s\nwant original:\n%s", plan.Content, original)
	}
	if len(plan.Result.Applied) != 0 {
		t.Fatalf("applied = %v, want none", plan.Result.Applied)
	}
}

func TestPlanSchemaInferencesPreservesCommentsAndNativeScalars(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	original := []byte("# governed schema\nversion: 2\nschema:\n  status:\n    # current lifecycle\n    type: string\n  count:\n    type: integer\n    required: true\n")
	if err := os.WriteFile(stemPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanSchemaInferences(stemPath, []ReportInference{
		{Type: "constant_field", Field: "status", Value: "Pending"},
	})
	if err != nil {
		t.Fatalf("PlanSchemaInferences: %v", err)
	}
	if !plan.Modified {
		t.Fatal("expected Modified=true")
	}
	candidate := string(plan.Content)
	for _, want := range []string{"# governed schema", "# current lifecycle", "required: true", "default: Pending"} {
		if !strings.Contains(candidate, want) {
			t.Fatalf("candidate does not contain %q:\n%s", want, candidate)
		}
	}
	if strings.Contains(candidate, `required: "true"`) || strings.Contains(candidate, `required: 'true'`) {
		t.Fatalf("candidate quoted native boolean scalar:\n%s", candidate)
	}

	after, err := os.ReadFile(stemPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(original) {
		t.Fatalf("planning wrote to target:\n%s", after)
	}
}

func TestApplySchemaInferencePlanWritesAtomicallyAndPreservesExistingMode(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	original := []byte("version: 2\nschema:\n  status:\n    type: enum\n    values: [Pending]\n")
	if err := os.WriteFile(stemPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(stemPath, 0o600); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanSchemaInferences(stemPath, []ReportInference{
		{Type: "enum_values", Field: "status", Value: "[Pending Done]"},
	})
	if err != nil {
		t.Fatalf("PlanSchemaInferences: %v", err)
	}
	if err := ApplySchemaInferencePlan(plan); err != nil {
		t.Fatalf("ApplySchemaInferencePlan: %v", err)
	}

	after, err := os.ReadFile(stemPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) == string(original) {
		t.Fatal("expected persisted content to change")
	}
	stem, err := rules.ParseStem(stemPath, after)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(stem.Schema["status"].Values, "Done") {
		t.Fatalf("persisted values = %v, want Done", stem.Schema["status"].Values)
	}
	info, err := os.Stat(stemPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %v, want 0600", got)
	}
}

func TestApplySchemaInferencePlanFailsWhenParentDirectoryIsMissing(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  status:\n    type: enum\n    values: [Pending]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanSchemaInferences(stemPath, []ReportInference{
		{Type: "enum_values", Field: "status", Value: "[Pending Done]"},
	})
	if err != nil {
		t.Fatalf("PlanSchemaInferences: %v", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}

	if err := ApplySchemaInferencePlan(plan); err == nil {
		t.Fatal("expected error for missing parent directory")
	}
}

func TestPlanSchemaInferencesNormalizesOrderBeforeMutation(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	original := []byte("version: 2\nschema:\n  title:\n    type: string\n")
	if err := os.WriteFile(stemPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	ascending := []ReportInference{
		{Type: "field_type", Field: "priority", Value: "enum"},
		{Type: "enum_values", Field: "priority", Value: "[Low High]"},
		{Type: "field_type", Field: "status", Value: "string"},
		{Type: "required_field", Field: "status"},
		{Type: "constant_field", Field: "status", Value: "Pending"},
	}
	reversed := append([]ReportInference(nil), ascending...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}

	planA, err := PlanSchemaInferences(stemPath, ascending)
	if err != nil {
		t.Fatalf("PlanSchemaInferences ascending: %v", err)
	}
	planB, err := PlanSchemaInferences(stemPath, reversed)
	if err != nil {
		t.Fatalf("PlanSchemaInferences reversed: %v", err)
	}
	if string(planA.Content) != string(planB.Content) {
		t.Fatalf("candidate bytes differ by inference order:\nA:\n%s\nB:\n%s", planA.Content, planB.Content)
	}
	if strings.Join(planA.Result.Applied, "\n") != strings.Join(planB.Result.Applied, "\n") {
		t.Fatalf("applied actions differ by inference order:\nA=%#v\nB=%#v", planA.Result.Applied, planB.Result.Applied)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
