package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSchemaErrorsHaveStableSerializedOrder(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":   "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  status:\n    type: enum\n    required: true\n    values: [Pending, Completed]\n  priority:\n    type: enum\n    required: true\n    values: [Low, High]\n",
		"task.md": "---\ntitle: Example\n---\n\n# Example\n",
	})
	mustChdir(t, root)

	tests := []struct {
		name   string
		args   []string
		format string
	}{
		{name: "single JSON", args: []string{filepath.Join(root, "task.md"), "-o", "json"}, format: "json"},
		{name: "all JSON", args: []string{"--all", ".", "-o", "json"}, format: "json"},
		{name: "single table", args: []string{filepath.Join(root, "task.md"), "-o", "table"}, format: "table"},
		{name: "all table", args: []string{"--all", ".", "-o", "table"}, format: "table"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var first string
			for run := 0; run < 64; run++ {
				stdout, err := executeValidate(t, tt.args...)
				if err != ErrValidationFailed {
					t.Fatalf("run %d: err = %v, want ErrValidationFailed\nstdout=%s", run, err, stdout)
				}
				if run == 0 {
					first = stdout
				} else if stdout != first {
					t.Fatalf("run %d: output changed\nfirst:\n%s\ncurrent:\n%s", run, first, stdout)
				}

				if tt.format == "json" {
					env := decodeEnvelope(t, stdout)
					if env["version"] != float64(2) || env["kind"] != "rootline/validate-batch" {
						t.Fatalf("run %d: unexpected envelope contract: version=%v kind=%v", run, env["version"], env["kind"])
					}
					errs := firstResult(t, stdout)["errors"].([]any)
					if len(errs) != 2 || errs[0].(map[string]any)["field"] != "priority" || errs[1].(map[string]any)["field"] != "status" {
						t.Fatalf("run %d: error order = %#v, want priority then status", run, errs)
					}
					continue
				}

				priority := strings.Index(stdout, `required field "priority" is missing`)
				status := strings.Index(stdout, `required field "status" is missing`)
				if priority < 0 || status < 0 || priority >= status {
					t.Fatalf("run %d: table errors not ordered priority then status:\n%s", run, stdout)
				}
			}
		})
	}
}

func TestValidateAggregateErrorsHaveStableSerializedOrder(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":     "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  total:\n    type: integer\n  completed:\n    type: integer\naggregate:\n  total: \"len(descendants)\"\n  completed: \"len(filter(descendants, .status == 'Completed'))\"\n",
		"README.md": "---\ntotal: 99\ncompleted: 99\n---\n\n# Aggregate index\n",
		"task.md":   "---\nstatus: Completed\n---\n\n# Task\n",
	})
	mustChdir(t, root)

	tests := []struct {
		name   string
		format string
	}{
		{name: "JSON", format: "json"},
		{name: "table", format: "table"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var first string
			for run := 0; run < 64; run++ {
				stdout, err := executeValidate(t, "--all", ".", "-o", tt.format)
				if err != ErrValidationFailed {
					t.Fatalf("run %d: err = %v, want ErrValidationFailed\nstdout=%s", run, err, stdout)
				}
				if run == 0 {
					first = stdout
				} else if stdout != first {
					t.Fatalf("run %d: output changed\nfirst:\n%s\ncurrent:\n%s", run, first, stdout)
				}

				if tt.format == "json" {
					env := decodeEnvelope(t, stdout)
					if env["version"] != float64(2) || env["kind"] != "rootline/validate-batch" {
						t.Fatalf("run %d: unexpected envelope contract: version=%v kind=%v", run, env["version"], env["kind"])
					}
					errs := firstResult(t, stdout)["errors"].([]any)
					if len(errs) != 2 || errs[0].(map[string]any)["field"] != "completed" || errs[1].(map[string]any)["field"] != "total" {
						t.Fatalf("run %d: error order = %#v, want completed then total", run, errs)
					}
					continue
				}

				completed := strings.Index(stdout, `field "completed" is "99" but aggregate computes "1"`)
				total := strings.Index(stdout, `field "total" is "99" but aggregate computes "1"`)
				if completed < 0 || total < 0 || completed >= total {
					t.Fatalf("run %d: table errors not ordered completed then total:\n%s", run, stdout)
				}
			}
		})
	}
}
