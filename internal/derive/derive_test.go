package derive

import (
	"testing"
)

func TestCompileValid(t *testing.T) {
	ev := NewEvaluator()
	compiled, err := ev.Compile("1 + 2")
	if err != nil {
		t.Fatalf("Compile valid expression: %v", err)
	}
	if compiled == nil {
		t.Fatal("expected non-nil CompiledExpr")
	}
	if compiled.Expression != "1 + 2" {
		t.Errorf("Expression = %q, want %q", compiled.Expression, "1 + 2")
	}
}

func TestCompileInvalid(t *testing.T) {
	ev := NewEvaluator()
	_, err := ev.Compile("invalid !!!")
	if err == nil {
		t.Fatal("expected error for invalid expression")
	}
	// Error should mention the expression.
	if got := err.Error(); !contains(got, "invalid !!!") {
		t.Errorf("error = %q, want it to contain the expression", got)
	}
}

func TestEvalArithmetic(t *testing.T) {
	ev := NewEvaluator()
	compiled, err := ev.Compile("a + b")
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	result, err := ev.Eval(compiled, map[string]any{"a": 3, "b": 4})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if result != 7 {
		t.Errorf("result = %v, want 7", result)
	}
}

func TestEvalWithEnv(t *testing.T) {
	ev := NewEvaluator()
	compiled, err := ev.Compile("titulo")
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	result, err := ev.Eval(compiled, map[string]any{"titulo": "Hello"})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if result != "Hello" {
		t.Errorf("result = %v, want %q", result, "Hello")
	}
}

func TestEvalStringConcatenation(t *testing.T) {
	ev := NewEvaluator()
	compiled, err := ev.Compile(`a + "-" + b`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	result, err := ev.Eval(compiled, map[string]any{"a": "hello", "b": "world"})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if result != "hello-world" {
		t.Errorf("result = %q, want %q", result, "hello-world")
	}
}

func TestBuildEnv(t *testing.T) {
	fm := map[string]any{
		"titulo": "Test",
		"estado": "Pending",
		"count":  42,
	}
	env := BuildEnv(fm)
	if len(env) != 3 {
		t.Errorf("len(env) = %d, want 3", len(env))
	}
	if env["titulo"] != "Test" {
		t.Errorf("env[titulo] = %v, want %q", env["titulo"], "Test")
	}
	if env["estado"] != "Pending" {
		t.Errorf("env[estado] = %v, want %q", env["estado"], "Pending")
	}
}

func TestBuildEnvEmpty(t *testing.T) {
	env := BuildEnv(map[string]any{})
	if len(env) != 0 {
		t.Errorf("len(env) = %d, want 0", len(env))
	}
}

func TestEvalUndefinedVariable(t *testing.T) {
	ev := NewEvaluator()
	compiled, err := ev.Compile("missing_field")
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	// With AllowUndefinedVariables, accessing a missing field returns nil, not error.
	result, err := ev.Eval(compiled, map[string]any{})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
}

func TestEvalBooleanExpression(t *testing.T) {
	ev := NewEvaluator()
	compiled, err := ev.Compile("estado == 'Pending'")
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	result, err := ev.Eval(compiled, map[string]any{"estado": "Pending"})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if result != true {
		t.Errorf("result = %v, want true", result)
	}
}

func TestCompileEmptyExpression(t *testing.T) {
	ev := NewEvaluator()
	_, err := ev.Compile("")
	if err == nil {
		t.Fatal("expected error for empty expression")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
