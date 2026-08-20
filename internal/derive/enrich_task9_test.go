package derive

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestEnrichBuiltinsTask9_SourceResolutionMatrix(t *testing.T) {
	stem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"notes": {Type: "string", Extract: `body.section["## Notes"]`},
		"plain": {Type: "string"},
	}}
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }
	records := []*extract.Record{
		{Path: "body.md", Body: "# Body\n\n## Notes\n\nfrom body", Frontmatter: map[string]any{}, Derived: map[string]any{"plain": "kept"}},
		{Path: "empty-fm.md", Body: "# Body\n\n## Notes\n\nfrom body", Frontmatter: map[string]any{"notes": ""}},
		{Path: "nil-fm.md", Body: "# Body\n\n## Notes\n\nfrom body", Frontmatter: map[string]any{"notes": nil}},
		{Path: "empty-section.md", Body: "# Body\n\n## Notes\n\n## End\n\nend", Frontmatter: map[string]any{}},
	}

	if err := EnrichBuiltins(context.Background(), records, "/root", resolver); err != nil {
		t.Fatalf("EnrichBuiltins unexpected error: %v", err)
	}

	if got := records[0].Derived["notes"]; got != "from body" {
		t.Fatalf("body fallback notes = %#v, want body content", got)
	}
	if got := records[0].Derived["plain"]; got != "kept" {
		t.Fatalf("non-source derived value = %#v, want preserved", got)
	}
	if got := records[1].Derived["notes"]; got != "" {
		t.Fatalf("empty frontmatter override = %#v, want empty string", got)
	}
	if got, ok := records[2].Derived["notes"]; !ok || got != nil {
		t.Fatalf("nil frontmatter override = (%#v,%v), want present nil", got, ok)
	}
	if got, ok := records[3].Derived["notes"]; !ok || got != "" {
		t.Fatalf("empty source section = (%#v,%v), want present empty string", got, ok)
	}
}

func TestEnrichBuiltinsTask9_ResolverMatchScopeControlsSourceFields(t *testing.T) {
	sourceStem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"notes": {Type: "string", Extract: `body.section["## Notes"]`},
	}}
	plainStem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"notes": {Type: "string"},
	}}
	records := []*extract.Record{
		{Path: "scoped.md", Body: "# Scoped\n\n## Notes\n\nfrom body", Frontmatter: map[string]any{}, Derived: map[string]any{"plain": "kept"}},
		{Path: "plain.md", Body: "# Plain\n\n## Notes\n\nignored", Frontmatter: map[string]any{}, Derived: map[string]any{"plain": "kept"}},
	}
	resolver := func(dir, recordPath string) *rules.StemFile {
		if recordPath == "scoped.md" {
			return sourceStem
		}
		return plainStem
	}

	if err := EnrichBuiltins(context.Background(), records, "/root", resolver); err != nil {
		t.Fatalf("EnrichBuiltins unexpected error: %v", err)
	}
	if got := records[0].Derived["notes"]; got != "from body" {
		t.Fatalf("scoped notes = %#v, want body value", got)
	}
	if _, exists := records[1].Derived["notes"]; exists {
		t.Fatalf("plain scoped record got source-derived notes: %#v", records[1].Derived)
	}
	for _, rec := range records {
		if got := rec.Derived["plain"]; got != "kept" {
			t.Fatalf("%s non-source derived = %#v, want kept", rec.Path, got)
		}
	}
}

func TestEnrichBuiltinsTask9_DuplicateSourceReturnsError(t *testing.T) {
	stem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"notes": {Type: "string", Extract: `body.section["## Notes"]`},
	}}
	record := &extract.Record{
		Path:        "dup.md",
		Body:        "# Body\n\n## Notes\n\nfirst\n\n## Notes\n\nsecond",
		Frontmatter: map[string]any{},
	}

	err := EnrichBuiltins(context.Background(), []*extract.Record{record}, "/root", func(dir, recordPath string) *rules.StemFile { return stem })
	if err == nil || !strings.Contains(err.Error(), "ambiguous body section source") {
		t.Fatalf("EnrichBuiltins duplicate error = %v, want ambiguity", err)
	}
}

func TestEnrichBuiltinsTask9_LaterRecordFailureDoesNotMutateAnyRecord(t *testing.T) {
	stem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"notes": {Type: "string", Extract: `body.section["## Notes"]`},
	}}
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }
	records := []*extract.Record{
		{
			Path:        "good.md",
			Body:        "# Good\n\n## Notes\n\nfrom body",
			Frontmatter: map[string]any{},
			Derived:     map[string]any{"existing": "kept"},
			Errors:      []extract.ExtractionError{{Message: "kept error"}},
		},
		{
			Path:        "bad.md",
			Body:        "# Bad\n\n## Notes\n\nfirst\n\n## Notes\n\nsecond",
			Frontmatter: map[string]any{},
		},
	}
	beforeDerivedIdentity := []uintptr{mapIdentity(records[0].Derived), mapIdentity(records[1].Derived)}
	beforeErrorSlices := [][]extract.ExtractionError{records[0].Errors, records[1].Errors}
	beforeDerived := []map[string]any{cloneAnyMap(records[0].Derived), cloneAnyMap(records[1].Derived)}
	beforeErrors := [][]extract.ExtractionError{append([]extract.ExtractionError(nil), records[0].Errors...), append([]extract.ExtractionError(nil), records[1].Errors...)}

	err := EnrichBuiltins(context.Background(), records, "/root", resolver)
	if err == nil || !strings.Contains(err.Error(), `resolving source field "notes" for bad.md`) {
		t.Fatalf("EnrichBuiltins error = %v, want contextual bad.md notes failure", err)
	}
	for i, rec := range records {
		if mapIdentity(rec.Derived) != beforeDerivedIdentity[i] {
			t.Fatalf("record %d Derived map identity changed after failed batch", i)
		}
		if !sameExtractionErrorSlice(rec.Errors, beforeErrorSlices[i]) {
			t.Fatalf("record %d Errors slice identity changed after failed batch", i)
		}
		if !reflect.DeepEqual(rec.Derived, beforeDerived[i]) {
			t.Fatalf("record %d Derived mutated after failed batch: got %#v want %#v", i, rec.Derived, beforeDerived[i])
		}
		if !reflect.DeepEqual(rec.Errors, beforeErrors[i]) {
			t.Fatalf("record %d Errors mutated after failed batch: got %#v want %#v", i, rec.Errors, beforeErrors[i])
		}
	}
}

func TestEnrichBuiltinsTask9_CancellationAtFinalPublicationGatePreservesAllRecords(t *testing.T) {
	stem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"notes": {Type: "string", Extract: `body.section["## Notes"]`},
	}}
	records := []*extract.Record{
		{
			Path:        "first.md",
			Body:        "# First\n\n## Notes\n\nfirst body",
			Frontmatter: map[string]any{},
			Derived:     map[string]any{"existing": "first kept"},
			Errors:      []extract.ExtractionError{{Message: "first kept error"}},
		},
		{
			Path:        "second.md",
			Body:        "# Second\n\n## Notes\n\nsecond body",
			Frontmatter: map[string]any{},
			Derived:     map[string]any{"existing": "second kept"},
			Errors:      []extract.ExtractionError{{Message: "second kept error"}},
		},
	}
	beforeDerivedIdentity := []uintptr{mapIdentity(records[0].Derived), mapIdentity(records[1].Derived)}
	beforeErrorSlices := [][]extract.ExtractionError{records[0].Errors, records[1].Errors}
	beforeDerived := []map[string]any{cloneAnyMap(records[0].Derived), cloneAnyMap(records[1].Derived)}
	beforeErrors := [][]extract.ExtractionError{append([]extract.ExtractionError(nil), records[0].Errors...), append([]extract.ExtractionError(nil), records[1].Errors...)}
	ctx := &errAfterNContext{Context: context.Background(), cancelAt: 3}
	resolverCalls := 0
	resolver := func(dir, recordPath string) *rules.StemFile {
		resolverCalls++
		return stem
	}

	err := EnrichBuiltins(ctx, records, "/root", resolver)
	if err != context.Canceled {
		t.Fatalf("EnrichBuiltins cancellation error = %v, want context.Canceled", err)
	}
	if resolverCalls != 2 || ctx.errCalls != 3 {
		t.Fatalf("call flow = %d resolver calls, %d Err calls; want two records resolved and cancellation at final Err", resolverCalls, ctx.errCalls)
	}
	for i, rec := range records {
		if mapIdentity(rec.Derived) != beforeDerivedIdentity[i] {
			t.Fatalf("record %d Derived map identity changed after final-gate cancellation", i)
		}
		if !sameExtractionErrorSlice(rec.Errors, beforeErrorSlices[i]) {
			t.Fatalf("record %d Errors slice identity changed after final-gate cancellation", i)
		}
		if !reflect.DeepEqual(rec.Derived, beforeDerived[i]) || !reflect.DeepEqual(rec.Errors, beforeErrors[i]) {
			t.Fatalf("record %d mutated after final-gate cancellation: derived %#v errors %#v", i, rec.Derived, rec.Errors)
		}
	}
}

func TestEnrichBuiltinsTask9_MultipleAmbiguousFieldsReportSortedField(t *testing.T) {
	stem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"zeta":  {Type: "string", Extract: `body.section["## Notes"]`},
		"alpha": {Type: "string", Extract: `body.section["## Notes"]`},
	}}
	record := &extract.Record{
		Path:        "dup.md",
		Body:        "# Body\n\n## Notes\n\nfirst\n\n## Notes\n\nsecond",
		Frontmatter: map[string]any{},
	}

	for i := 0; i < 50; i++ {
		err := EnrichBuiltins(context.Background(), []*extract.Record{record}, "/root", func(dir, recordPath string) *rules.StemFile { return stem })
		if err == nil || !strings.Contains(err.Error(), `resolving source field "alpha" for dup.md`) {
			t.Fatalf("run %d EnrichBuiltins error = %v, want deterministic alpha failure", i, err)
		}
	}
}

type errAfterNContext struct {
	context.Context
	cancelAt int
	errCalls int
}

func (c *errAfterNContext) Err() error {
	c.errCalls++
	if c.errCalls >= c.cancelAt {
		return context.Canceled
	}
	return nil
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mapIdentity(in map[string]any) uintptr {
	if in == nil {
		return 0
	}
	return reflect.ValueOf(in).Pointer()
}

func sameExtractionErrorSlice(a, b []extract.ExtractionError) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return a == nil && b == nil
	}
	return &a[0] == &b[0]
}
