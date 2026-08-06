package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// --field is registered as a StringSliceVar and documented "(repeatable)", but
// only fieldPath[0] was ever read, so the result depended purely on argument
// order. Both paths must come back, in flag order.
func TestField_RepeatableExtractsEveryPath(t *testing.T) {
	docs := setupFormatProject(t)

	out, err := runCmd(t, "query", docs, "--count", "--field", "meta.count", "--field", "kind")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("expected a JSON array for two --field flags, got %s", out)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 extracted values, got %v", got)
	}
	if got[0] != float64(2) {
		t.Errorf("first value = %v, want the count 2", got[0])
	}
	if got[1] != "rootline/count" {
		t.Errorf("second value = %v, want the kind", got[1])
	}
}

// Order follows the flags, not the envelope.
func TestField_RepeatableFollowsFlagOrder(t *testing.T) {
	docs := setupFormatProject(t)

	out, err := runCmd(t, "query", docs, "--count", "--field", "kind", "--field", "meta.count")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("expected a JSON array, got %s", out)
	}
	if got[0] != "rootline/count" || got[1] != float64(2) {
		t.Errorf("got %v, want the kind first and the count second", got)
	}
}

// A single --field keeps emitting the bare value, so no existing caller moves.
func TestField_SinglePathStaysScalar(t *testing.T) {
	docs := setupFormatProject(t)

	out, err := runCmd(t, "query", docs, "--count", "--field", "kind")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != `"rootline/count"` {
		t.Errorf("got %s, want the bare value", out)
	}
}

// --field reads the JSON envelope. Table, CSV, JSONL and the diagram writers
// never saw it, so the flag stopped working the moment a caller added -o table
// — with no diagnostic.
func TestField_RejectedOnNonJSONOutput(t *testing.T) {
	docs := setupFormatProject(t)

	cases := []struct {
		name string
		args []string
	}{
		{"stats table", []string{"stats", docs, "-o", "table", "--field", "kind"}},
		{"graph table", []string{"graph", docs, "-o", "table", "--field", "kind"}},
		{"tree table", []string{"tree", docs, "-o", "table", "--field", "kind"}},
		{"query csv", []string{"query", docs, "--select", "path,estado", "-o", "csv", "--field", "kind"}},
		{"query jsonl", []string{"query", docs, "--select", "path,estado", "-o", "jsonl", "--field", "kind"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runCmd(t, tc.args...)
			if err == nil {
				t.Fatalf("expected an error, got none; output: %s", out)
			}
			if !strings.Contains(err.Error(), "--field") || !strings.Contains(err.Error(), "--output json") {
				t.Errorf("error = %v, want it to name both flags", err)
			}
		})
	}
}

// A command that emits no envelope at all cannot honour --field either.
func TestField_RejectedOnFormatAgnosticCommand(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"hooks", "status"})
	if err != nil {
		t.Fatalf("locating hooks status: %v", err)
	}

	t.Cleanup(func() { fieldPath = nil })
	fieldPath = []string{"kind"}

	if err := validateFieldFlag(cmd); err == nil {
		t.Fatal("expected an error for --field on a format-agnostic command")
	} else if !strings.Contains(err.Error(), "--field") {
		t.Errorf("error = %v, want it to name the flag", err)
	}
}

// The empty values cobra can leave behind on re-execution are not a request.
func TestField_EmptyValuesAreNotARequest(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"query"})
	if err != nil {
		t.Fatalf("locating query: %v", err)
	}

	t.Cleanup(func() { fieldPath = nil; outputFormat = "json" })
	fieldPath = []string{"", ""}
	outputFormat = "table"

	if err := validateFieldFlag(cmd); err != nil {
		t.Errorf("empty --field values must not trigger the check: %v", err)
	}
}

// The help text must not promise more than the flag delivers.
func TestField_HelpScopesTheFlagToJSON(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("field")
	if f == nil {
		t.Fatal("--field is not registered")
	}
	if !strings.Contains(f.Usage, "repeatable") {
		t.Errorf("usage = %q, want it to keep the repeatable claim now that it is true", f.Usage)
	}
	if !strings.Contains(f.Usage, "json") {
		t.Errorf("usage = %q, want it to scope the flag to --output json", f.Usage)
	}
}
