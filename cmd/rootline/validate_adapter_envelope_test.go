package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

func TestValidateNormalizationFailureAccumulatesAndEmitsEnvelope(t *testing.T) {
	governanceRoot := t.TempDir()
	effective, err := rules.ParseStem(filepath.Join(filepath.Dir(governanceRoot), "outside.stem"), []byte("version: 2\nschema:\n  title:\n    type: string\n    required: true\n"))
	if err != nil {
		t.Fatal(err)
	}
	record := &extract.Record{Path: "doc.md", Frontmatter: map[string]any{}}
	errs := rules.Validate(context.Background(), record, effective)
	if len(errs) != 1 || errs[0].Source != effective.Schema["title"].Source {
		t.Fatalf("rules.Validate errors = %#v, want one real field-source error from %q", errs, effective.Schema["title"].Source)
	}

	for _, format := range []string{"json", "table"} {
		t.Run(format, func(t *testing.T) {
			results := []*rules.ValidationResult{rules.NewValidationResult("prior.md", nil)}
			results, notices := appendNormalizedValidationResult(results, nil, rules.NewValidationResult(record.Path, errs), record.Path, filepath.Join(governanceRoot, record.Path), governanceRoot)

			var stdout bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&stdout)
			outputFormat = format
			validateStrict = false
			t.Cleanup(resetFlags)
			err := emitValidateEnvelope(cmd, rules.NewValidationEnvelope(rules.ValidationEnvelopeInput{Results: results, Notices: notices}))
			if err != ErrValidationFailed {
				t.Fatalf("emit err = %v, want ErrValidationFailed\nstdout=%s", err, stdout.String())
			}
			if strings.TrimSpace(stdout.String()) == "" {
				t.Fatal("emitter returned nonzero before writing stdout")
			}

			if format == "table" {
				for _, want := range []string{"prior.md", "doc.md", "schema_resolution_failed"} {
					if !strings.Contains(stdout.String(), want) {
						t.Fatalf("table output missing %q:\n%s", want, stdout.String())
					}
				}
				return
			}

			env := decodeEnvelope(t, stdout.String())
			assertJSONKeys(t, env, []string{"version", "kind", "results", "structural", "stem_health", "drift_warnings", "notices", "summary"})
			if got := envelopePaths(t, env); len(got) != 2 || got[0] != "prior.md" || got[1] != "doc.md" {
				t.Fatalf("result paths = %v, want prior result followed by skipped document", got)
			}
			for _, row := range env["results"].([]any) {
				assertJSONKeys(t, row.(map[string]any), []string{"version", "kind", "path", "valid", "errors", "warnings"})
			}
			assertSkippedResultPaths(t, map[string]any{"results": []any{env["results"].([]any)[1]}}, []string{"doc.md"})
			assertSummaryCounts(t, env, map[string]float64{"total": 2, "valid": 1, "invalid": 1, "errors_count": 1})
			assertNoticeCodes(t, env, []string{"schema_resolution_failed"})
			message := env["notices"].([]any)[0].(map[string]any)["message"].(string)
			if !strings.Contains(message, "outside governance root") {
				t.Fatalf("notice message = %q, want causal normalization failure", message)
			}
		})
	}
}
