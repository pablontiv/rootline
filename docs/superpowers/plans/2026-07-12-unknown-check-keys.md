# Unknown `links.checks` Keys Warning Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Capture unknown keys under `links.checks` at `.stem` parse time and surface them as stem health warning #11 (`unknown-check-keys`) with a fuzzy "did you mean" suggestion.

**Architecture:** `LinkSchema.UnmarshalYAML` already special-cases the `checks` key; after the existing `Decode`, iterate the mapping's key nodes and stash keys outside {resolve, anchors, encoding, cycles} into a new diagnostic field `UnknownCheckKeys []string` (excluded from JSON). `ValidateStemHealth` (same package) reads that field per parsed `.stem` file and emits `warn` checks with suggestions from `github.com/pablontiv/picokit/fuzzy` (already a dependency, used the same way in `validate.go:63`). No changes to merge, `IsEmpty()`, or any JSON contract.

**Tech Stack:** Go 1.25+, gopkg.in/yaml.v3, `github.com/pablontiv/picokit/fuzzy`, standard `testing` package.

**Spec:** `docs/superpowers/specs/2026-07-12-unknown-check-keys-design.md`

## Global Constraints

- Work from repo root `/Users/Shared/harness/rootline`.
- `just check` and `just test` (`go test ./... -race`) green after every task; coverage floor 85% per package (`just coverage-check`, enforced by pre-push hook).
- Conventional commits; no Co-Authored-By or AI attribution.
- All strings, comments, identifiers in English.
- The new field must never serialize: JSON outputs of `describe`, `graph`, `query` are unchanged.
- Stem health `warn` status must be used (not `fail`): exit code of `validate --all` changes only under the existing `--strict` flag.

---

### Task 1: Capture unknown check keys + stem health check #11 + docs

**Files:**
- Modify: `internal/rules/rules.go` (LinkSchema struct ~line 54; `UnmarshalYAML` `checks` branch ~line 113)
- Modify: `internal/rules/stemhealth.go` (append check inside `ValidateStemHealth`, after the last existing check over `parsedStems`)
- Modify: `CLAUDE.md:33` ("10 checks" list in the `internal/rules/` bullet)
- Test: `internal/rules/rules_test.go`, `internal/rules/stemhealth_test.go`

**Interfaces:**
- Consumes: `fuzzy.Match(input string, candidates []string) string` from `github.com/pablontiv/picokit/fuzzy` (returns `""` on no match — same usage as `internal/rules/validate.go:63`); `parsedStems map[string]*StemFile` already built in `ValidateStemHealth`; `StemFile.Links` of type `LinkSchema`.
- Produces: `rules.LinkSchema.UnknownCheckKeys []string` (document order, `json:"-"`); package-level `knownCheckKeys []string`; stem health check name `"unknown-check-keys"`.

- [ ] **Step 1: Write the failing rules tests**

Append to `internal/rules/rules_test.go`:

```go
func TestLinkSchema_UnknownCheckKeysCaptured(t *testing.T) {
	src := `
links:
  checks:
    resolve: true
    cicles: true
    ancors: false
`
	var stem StemFile
	if err := yaml.Unmarshal([]byte(src), &stem); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := stem.Links.UnknownCheckKeys
	if len(got) != 2 || got[0] != "cicles" || got[1] != "ancors" {
		t.Errorf("UnknownCheckKeys = %v, want [cicles ancors]", got)
	}
}

func TestLinkSchema_KnownCheckKeysNotCaptured(t *testing.T) {
	src := `
links:
  checks:
    resolve: true
    anchors: true
    encoding: true
    cycles: true
`
	var stem StemFile
	if err := yaml.Unmarshal([]byte(src), &stem); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(stem.Links.UnknownCheckKeys) != 0 {
		t.Errorf("UnknownCheckKeys = %v, want empty", stem.Links.UnknownCheckKeys)
	}
}
```

- [ ] **Step 2: Write the failing stemhealth tests**

Append to `internal/rules/stemhealth_test.go` (add `"strings"` to its imports):

```go
func TestValidateStemHealth_UnknownCheckKeys(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
links:
  checks:
    cicles: true
`))
	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var found *StemHealthCheck
	for i, c := range result.Checks {
		if c.Name == "unknown-check-keys" {
			found = &result.Checks[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected unknown-check-keys check")
	}
	if found.Status != "warn" {
		t.Errorf("status = %q, want warn", found.Status)
	}
	if found.Field != "cicles" {
		t.Errorf("field = %q, want cicles", found.Field)
	}
	if !strings.Contains(found.Message, `did you mean "cycles"?`) {
		t.Errorf("message = %q, want cycles suggestion", found.Message)
	}
}

func TestValidateStemHealth_KnownCheckKeysNoWarn(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
links:
  checks:
    resolve: true
    cycles: true
`))
	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "unknown-check-keys" {
			t.Errorf("unexpected unknown-check-keys check: %+v", c)
		}
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/rules/ -run 'UnknownCheckKeys|KnownCheckKeys' -v`
Expected: FAIL to compile — `stem.Links.UnknownCheckKeys undefined`.

- [ ] **Step 4: Implement the capture in `rules.go`**

4a. Add the field to `LinkSchema` (struct currently at ~line 54):

```go
type LinkSchema struct {
	Allowed []string            `json:"allowed,omitempty"`
	Styles  []string            `json:"styles,omitempty"`
	Checks  *LinkChecks         `json:"checks,omitempty"`
	Rules   map[string]LinkRule `json:"rules,omitempty"`

	// UnknownCheckKeys lists keys found under links.checks that no check
	// consumes. Diagnostic only — surfaced by stem health, never serialized.
	UnknownCheckKeys []string `json:"-"`
}
```

4b. Add the known-key list next to `LinkChecks` (shared with stemhealth — same package):

```go
// knownCheckKeys lists the keys LinkChecks consumes; keep in sync with its
// struct fields.
var knownCheckKeys = []string{"resolve", "anchors", "encoding", "cycles"}
```

4c. In `UnmarshalYAML`, extend the `checks` branch (add `"slices"` to the file's imports):

```go
		if key == "checks" {
			var checks LinkChecks
			if err := val.Decode(&checks); err != nil {
				return fmt.Errorf("links.checks: %w", err)
			}
			ls.Checks = &checks
			for j := 0; j+1 < len(val.Content); j += 2 {
				k := val.Content[j].Value
				if !slices.Contains(knownCheckKeys, k) {
					ls.UnknownCheckKeys = append(ls.UnknownCheckKeys, k)
				}
			}
			continue
		}
```

(If `val` is not a mapping, `Decode` already errors before the loop; the loop preserves document order.)

- [ ] **Step 5: Implement stem health check #11 in `stemhealth.go`**

Append inside `ValidateStemHealth`, after the last existing check loop and before the final `return`. Add `"github.com/pablontiv/picokit/fuzzy"` to the file's imports:

```go
	// Check 11: unknown keys under links.checks (silently inert otherwise)
	for sf, stem := range parsedStems {
		relPath, _ := filepath.Rel(absRoot, sf)
		for _, key := range stem.Links.UnknownCheckKeys {
			msg := fmt.Sprintf("unknown key %q in links.checks", key)
			if match := fuzzy.Match(key, knownCheckKeys); match != "" {
				msg += fmt.Sprintf(" (did you mean %q?)", match)
			}
			checks = append(checks, StemHealthCheck{
				Name:    "unknown-check-keys",
				Status:  "warn",
				Message: msg,
				Path:    relPath,
				Field:   key,
			})
		}
	}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/rules/ -run 'UnknownCheckKeys|KnownCheckKeys|StemHealth' -v`
Expected: PASS (all, including pre-existing StemHealth tests).

- [ ] **Step 7: Update CLAUDE.md**

In the `internal/rules/` bullet (line 33), change:

```
stem health diagnostics (10 checks: yaml-valid, scope-match, type-consistency, enum-values, rule-field-exists, field-override, aggregated-required, aggregate-formula-coverage, stem-files-exist, monotonic-violations — called by `validate --all` as pre-phase)
```

to:

```
stem health diagnostics (11 checks: yaml-valid, scope-match, type-consistency, enum-values, rule-field-exists, field-override, aggregated-required, aggregate-formula-coverage, stem-files-exist, monotonic-violations, unknown-check-keys — called by `validate --all` as pre-phase)
```

- [ ] **Step 8: Full suite and commit**

Run: `just check && just test`
Expected: both green.

```bash
git add internal/rules/rules.go internal/rules/rules_test.go internal/rules/stemhealth.go internal/rules/stemhealth_test.go CLAUDE.md
git commit -m "feat(rules): warn on unknown links.checks keys in stem health"
```

---

### Task 2: Verification, acceptance, and Definition of Done

**Files:** none created (backlog dir `/opt/factory/docs/backlog/` known absent on this machine — check, record, skip if still missing).

**Interfaces:**
- Consumes: Task 1 pushed to `master`; installed `rootline` with picokit autoupdate.

- [ ] **Step 1: Local gates**

Run: `just check && just test && just coverage-check`
Expected: all green, no package below 85%.

- [ ] **Step 2: Acceptance with dev build (scratch dir, NOT the dsprima wiki)**

```bash
go build -o /tmp/rl-dev ./cmd/rootline
mkdir -p /tmp/ucktest && cd /tmp/ucktest
printf 'version: 2\nlinks:\n  checks:\n    cicles: true\n' > .stem
printf -- '---\n---\n# Doc\n' > a.md
/tmp/rl-dev validate --all . 2>&1 | rg "unknown-check-keys|cicles"; echo "exit=$?"
/tmp/rl-dev validate --all . > /dev/null 2>&1; echo "validate exit=$?"
```

Expected: output names `cicles` with suggestion `cycles`; the warn does NOT change the exit code (0 if the docs otherwise validate). Then fix the typo to `cycles: true` and re-run — no `unknown-check-keys` output. Clean up `/tmp/ucktest` and `/tmp/rl-dev`.

- [ ] **Step 3: Push and release**

```bash
cd /Users/Shared/harness/rootline && git push
```

CI auto-tags a **minor** bump (post-1.0 `feat` → v2.1.0) and publishes. Watch with `gh run list --limit 3` until green and `gh release view --json tagName` shows the new tag.

- [ ] **Step 4: Install verify (DoD)**

```bash
which rootline
rootline --version   # first run fetches update in background
rootline --version   # second run applies staged update — must show v2.1.0
```

- [ ] **Step 5: Re-run acceptance with the installed binary**

Repeat Step 2's scratch-dir check using `rootline` instead of `/tmp/rl-dev`; expect identical results. Also run `rootline validate --all .` on `/Users/Shared/dsprima/docs` (read-only, no canary): expect no `unknown-check-keys` output (its `.stem` uses only known keys) and unchanged results versus v2.0.0.

Done when: gates green, pushed, release v2.1.0 live, installed CLI shows v2.1.0, both acceptance runs pass, backlog status recorded.
