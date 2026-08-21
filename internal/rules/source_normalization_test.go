package rules

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestNormalizeValidationSources_GovernanceRelative(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "docs", ".stem")
	got, err := NormalizeValidationSources([]ValidationError{{Source: source}}, root)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Source != "docs/.stem" {
		t.Fatalf("got %q", got[0].Source)
	}
}

func TestNormalizeValidationSources_PreservesExactSymbolicAndParsedBodySources(t *testing.T) {
	for _, source := range []string{"", "schema", "scope", "links.checks", "body.h1", `body.section["## Notes"]`} {
		got, err := NormalizeValidationSources([]ValidationError{{Source: source}}, t.TempDir())
		if err != nil || got[0].Source != source {
			t.Fatalf("source=%q got=%+v err=%v", source, got, err)
		}
	}
}

func TestNormalizeValidationSources_RejectsUnknownAndInvalidBodyTokens(t *testing.T) {
	for _, source := range []string{"frontmatter.title", "contract.stem", ".", "body.section[## Notes]", `body.section["Notes"]`} {
		_, err := NormalizeValidationSources([]ValidationError{{Source: source}}, t.TempDir())
		if err == nil {
			t.Fatalf("source %q: expected unsupported source rejection", source)
		}
	}
}

func TestNormalizeValidationSources_RejectsInvalidGovernanceRoot(t *testing.T) {
	for _, root := range []string{"", ".", "relative/root", `C:relative`} {
		_, err := NormalizeValidationSources([]ValidationError{{Source: "schema"}}, root)
		if err == nil {
			t.Fatalf("governance root %q: expected rejection", root)
		}
	}
}

func TestNormalizeValidationSources_ClonesInput(t *testing.T) {
	root := t.TempDir()
	original := []ValidationError{
		{Rule: "required", Field: "title", Message: "missing", Source: filepath.Join(root, ".stem"), Severity: "error"},
		{Rule: "link_resolve", Field: "links", Message: "bad", Source: "links.checks", Severity: "warn"},
	}
	before := append([]ValidationError(nil), original...)

	got, err := NormalizeValidationSources(original, root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, before) {
		t.Fatalf("NormalizeValidationSources mutated input: got %#v want %#v", original, before)
	}
	if got[0].Source != ".stem" || got[1].Source != "links.checks" {
		t.Fatalf("normalized sources = %#v", got)
	}
}

func TestNormalizeValidationSources_AlreadyRelativeFilesystemPath(t *testing.T) {
	got, err := NormalizeValidationSources([]ValidationError{{Source: `docs\\nested\\.stem`}}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Source != "docs/nested/.stem" {
		t.Fatalf("got %q", got[0].Source)
	}
}

func TestNormalizeValidationSources_RejectsOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "outside", ".stem")
	_, err := NormalizeValidationSources([]ValidationError{{Source: outside}}, root)
	if err == nil {
		t.Fatal("expected outside-root error")
	}
}

func TestNormalizeValidationSources_RejectsSiblingPrefixAndEscapes(t *testing.T) {
	root := t.TempDir()
	for _, source := range []string{
		filepath.Join(root+"-sibling", ".stem"),
		filepath.Join("..", ".stem"),
		`..\\.stem`,
		`C:record.md`,
		`C:\\outside\\record.md`,
		`\\\\server\\share\\record.md`,
	} {
		_, err := NormalizeValidationSources([]ValidationError{{Source: source}}, root)
		if err == nil {
			t.Fatalf("source %q: expected escape rejection", source)
		}
	}
}

func TestNormalizeValidationSources_DoesNotMutateInputOnFailure(t *testing.T) {
	root := t.TempDir()
	original := []ValidationError{{Source: filepath.Join(root, ".stem")}, {Source: filepath.Join("..", ".stem")}}
	before := append([]ValidationError(nil), original...)

	_, err := NormalizeValidationSources(original, root)
	if err == nil {
		t.Fatal("expected normalization failure")
	}
	if !reflect.DeepEqual(original, before) {
		t.Fatalf("NormalizeValidationSources mutated input on failure: got %#v want %#v", original, before)
	}
}

func TestNormalizeValidationSources_RejectsCrossVolumeOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("cross-volume filepath.Rel is Windows-specific")
	}
	_, err := NormalizeValidationSources([]ValidationError{{Source: `D:\\outside\\record.md`}}, `C:\\root`)
	if err == nil {
		t.Fatal("expected cross-volume rejection")
	}
}

func TestNormalizeValidationSources_DeterministicOutput(t *testing.T) {
	root := t.TempDir()
	errs := []ValidationError{{Source: filepath.Join(root, "b", ".stem")}, {Source: filepath.Join(root, "a", ".stem")}}
	first, err := NormalizeValidationSources(errs, root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NormalizeValidationSources(errs, root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("first=%#v second=%#v", first, second)
	}
}

func TestNormalizePublicPath_WindowsSeparators(t *testing.T) {
	if got := normalizePublicPath(`docs\\nested\\.stem`); got != "docs/nested/.stem" {
		t.Fatalf("got %q", got)
	}
	if got := normalizePublicPath(filepath.Join("docs", "nested", ".stem")); strings.Contains(got, `\\`) {
		t.Fatalf("got backslash in %q", got)
	}
}
