# Opt-in Cycle Failure in `graph --check` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make cycle detection in `rootline graph --check` opt-in via `links.checks.cycles: true` in `.stem` (absent = informational), with a `--fail-cycles` CLI override.

**Architecture:** Add a plain `Cycles bool` field to the existing `LinkChecks` struct (symmetric with `resolve`/`anchors`/`encoding`). `cmd/rootline/graph.go` already loads the merged stem; derive `failCycles` from it, let an explicitly-set `--fail-cycles` flag override it, and exclude cycles from `hasProblems` unless failing. This is a **breaking change** (`feat!`): today cycles always fail `--check`; after this, they fail only when opted in.

**Tech Stack:** Go 1.25+, cobra, gopkg.in/yaml.v3, standard `testing` package (no external frameworks).

**Spec:** `docs/superpowers/specs/2026-07-12-graph-check-cycle-severity-design.md`

## Global Constraints

- Go 1.25+; run everything from repo root `/Users/Shared/harness/rootline`.
- `just check` (gofmt + golangci-lint + build) and `just test` (`go test ./... -race`) must pass after every task.
- Coverage floor: 85% per package (`.coverage-floors.toml`); pre-push hook runs `just coverage-check`.
- Conventional commits enforced by `.githooks/commit-msg`. No Co-Authored-By or AI attribution.
- All user-facing strings, comments, and identifiers in English.
- JSON output contract unchanged: `cycles` array always present and fully populated.
- The spec places behavioral tests in `internal/e2e`; this plan puts them in `cmd/rootline/graph_test.go` instead because the exit-code/flag logic under test lives only in the cobra layer (`runCmd` + `ErrValidationFailed`), which `internal/e2e` (library-level) cannot exercise. Deliberate, approved deviation of location, not coverage.

---

### Task 1: `Cycles` field on `LinkChecks`

**Files:**
- Modify: `internal/rules/rules.go:62-66` (`LinkChecks` struct)
- Test: `internal/rules/rules_test.go` (append after `TestLinkSchema_UnmarshalStylesAndChecks`, ~line 668)

**Interfaces:**
- Produces: `rules.LinkChecks.Cycles bool` — decoded from `.stem` key `links.checks.cycles`; zero value `false` means "cycles are informational". Task 2 reads this field.

- [ ] **Step 1: Write the failing test**

Append to `internal/rules/rules_test.go`:

```go
func TestLinkChecks_CyclesDecode(t *testing.T) {
	src := `
links:
  checks:
    cycles: true
`
	var stem StemFile
	if err := yaml.Unmarshal([]byte(src), &stem); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if stem.Links.Checks == nil || !stem.Links.Checks.Cycles {
		t.Errorf("Checks = %+v, want Cycles true", stem.Links.Checks)
	}

	var absent StemFile
	if err := yaml.Unmarshal([]byte("links:\n  checks:\n    resolve: true\n"), &absent); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if absent.Links.Checks == nil || absent.Links.Checks.Cycles {
		t.Errorf("Checks = %+v, want Cycles false when key absent", absent.Links.Checks)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rules/ -run TestLinkChecks_CyclesDecode`
Expected: FAIL to compile — `stem.Links.Checks.Cycles undefined (type *LinkChecks has no field or method Cycles)`

- [ ] **Step 3: Add the field**

In `internal/rules/rules.go`, replace the `LinkChecks` struct:

```go
// LinkChecks enables filesystem-backed link checks (ADO code-wiki conventions).
// Cycles opts graph --check into treating link cycles as failures.
type LinkChecks struct {
	Resolve  bool `yaml:"resolve" json:"resolve,omitempty"`
	Anchors  bool `yaml:"anchors" json:"anchors,omitempty"`
	Encoding bool `yaml:"encoding" json:"encoding,omitempty"`
	Cycles   bool `yaml:"cycles" json:"cycles,omitempty"`
}
```

No changes to `UnmarshalYAML`, merge, or `IsEmpty()` — the `checks` key already decodes into this struct wholesale.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/rules/ -run TestLinkChecks_CyclesDecode`
Expected: PASS

- [ ] **Step 5: Run package tests and commit**

Run: `go test ./internal/rules/ -race`
Expected: PASS (all)

```bash
git add internal/rules/rules.go internal/rules/rules_test.go
git commit -m "feat(rules): add cycles key to links.checks schema"
```

---

### Task 2: Opt-in cycle failure + `--fail-cycles` flag in `graph --check`

**Files:**
- Modify: `cmd/rootline/graph.go` (flag vars ~line 16-21, `init` ~line 31-37, stem-loading block ~line 87-94, `--check` block ~line 100-128)
- Modify: `cmd/rootline/commands_test.go:70-73` (`resetFlags`)
- Test: `cmd/rootline/graph_test.go` (replace `TestGraphCheck_WithCycle` at line 76-89; add new tests after it)

**Interfaces:**
- Consumes: `rules.LinkChecks.Cycles bool` (Task 1); existing helpers `runCmd(t, args...) (string, error)`, `mustWriteFile`, `ErrValidationFailed` (all already in `cmd/rootline` test files).
- Produces: CLI flag `--fail-cycles` (bool, overrides `.stem` only when explicitly set, via `cmd.Flags().Changed("fail-cycles")`); output header `Cycles found (informational): N` when not failing.

- [ ] **Step 1: Write the failing tests**

In `cmd/rootline/graph_test.go`, **replace** `TestGraphCheck_WithCycle` (lines 76-89) with:

```go
func TestGraphCheck_WithCycle_InformationalByDefault(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	out, err := runCmd(t, "graph", "--check", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Cycles found (informational): 1") {
		t.Errorf("expected informational cycle report, got: %s", out)
	}
	if strings.Contains(out, "No cycles or broken links") {
		t.Errorf("clean message printed despite cycles present: %s", out)
	}
}
```

Then **add** these four tests after it:

```go
func TestGraphCheck_CyclesOptInFails(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nlinks:\n  checks:\n    cycles: true\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	out, err := runCmd(t, "graph", "--check", dir)
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Cycles found: 1") {
		t.Errorf("expected failing cycle report, got: %s", out)
	}
}

func TestGraphCheck_InformationalCyclesDontMaskBrokenLinks(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n[[missing.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	out, err := runCmd(t, "graph", "--check", dir)
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed for broken link, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Broken links: 1") {
		t.Errorf("expected broken link report, got: %s", out)
	}
	if !strings.Contains(out, "Cycles found (informational): 1") {
		t.Errorf("expected informational cycles alongside broken links, got: %s", out)
	}
}

func TestGraphCheck_FailCyclesFlagForcesFailure(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	out, err := runCmd(t, "graph", "--check", "--fail-cycles=true", dir)
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed with --fail-cycles=true, got: %v\noutput: %s", err, out)
	}
}

func TestGraphCheck_FailCyclesFlagForcesInformational(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nlinks:\n  checks:\n    cycles: true\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	out, err := runCmd(t, "graph", "--check", "--fail-cycles=false", dir)
	if err != nil {
		t.Fatalf("expected success with --fail-cycles=false, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Cycles found (informational): 1") {
		t.Errorf("expected informational cycle report, got: %s", out)
	}
}
```

- [ ] **Step 2: Update `resetFlags` (test hygiene for `Changed`)**

`runCmd` reuses the process-global `rootCmd`; cobra's `Changed` state persists across `Execute()` calls, so a test that sets `--fail-cycles` would poison every later test. In `cmd/rootline/commands_test.go`, inside `resetFlags()` right after the existing `graphOpen = false` (line 73), add:

```go
	graphFailCycles = false
	if f := graphCmd.Flags().Lookup("fail-cycles"); f != nil {
		f.Changed = false
	}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./cmd/rootline/ -run 'TestGraphCheck' -v`
Expected: FAIL to compile — `undefined: graphFailCycles` (from resetFlags). After the vars exist, the new tests fail with `unknown flag: --fail-cycles` and wrong exit behavior.

- [ ] **Step 4: Implement in `graph.go`**

4a. Flag variable (top var block, line 16-21):

```go
var (
	graphFormat     string
	graphCheck      bool
	graphWhere      []string
	graphOpen       bool
	graphFailCycles bool
)
```

4b. Register in `init()` (after the `--open` flag, line 35):

```go
	graphCmd.Flags().BoolVar(&graphFailCycles, "fail-cycles", false, "treat cycles as check failures (overrides .stem links.checks.cycles)")
```

4c. Replace the stem-loading block (lines 87-94) to also read the setting:

```go
	// Load .stem schema: filter links and read the cycle-failure opt-in.
	failCycles := false
	entries, err := rules.WalkUp(absRoot)
	if err == nil {
		stem := rules.MergeStemFiles(entries)
		if stem != nil {
			filterLinksBySchema(records, stem.Links)
			failCycles = stem.Links.Checks != nil && stem.Links.Checks.Cycles
		}
	}
	if cmd.Flags().Changed("fail-cycles") {
		failCycles = graphFailCycles
	}
```

4d. Replace the `--check` block (lines 100-128):

```go
	// --check mode: report issues and exit.
	if graphCheck {
		hasProblems := (failCycles && len(cycles) > 0) || len(broken) > 0
		if len(cycles) > 0 {
			header := "Cycles found"
			if !failCycles {
				header = "Cycles found (informational)"
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: %d\n", header, len(cycles))
			for i, c := range cycles {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %d: %s\n", i+1, strings.Join(c, " → "))
			}
		}
		if len(broken) > 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Broken links: %d\n", len(broken))
			for _, b := range broken {
				msg := fmt.Sprintf("  %s:%d → %s (%s)", b.Source, b.Line, b.Target, b.Type)
				if len(b.Suggestions) > 0 {
					msg += fmt.Sprintf(" — did you mean: %s?", strings.Join(b.Suggestions, ", "))
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), msg)
			}
		}
		if len(cycles) == 0 && len(broken) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No cycles or broken links found.")
		}
		if hasProblems {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return ErrValidationFailed
		}
		return nil
	}
```

Note the clean-state message is now gated on "no cycles AND no broken" (not `!hasProblems`) — informational cycles must not print "No cycles or broken links found."

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./cmd/rootline/ -race`
Expected: PASS — including all pre-existing graph tests. If any other test asserted the old cycle-failure default, fix the test expectation (the behavior change is the point of this task), but do NOT weaken broken-link assertions.

- [ ] **Step 6: Full suite and commit**

Run: `just check && just test`
Expected: both green.

```bash
git add cmd/rootline/graph.go cmd/rootline/graph_test.go cmd/rootline/commands_test.go
git commit -m "feat!: make graph --check cycle failure opt-in via checks.cycles

BREAKING CHANGE: graph --check no longer fails on link cycles by default.
Cycles are reported as informational; set links.checks.cycles: true in
.stem (or pass --fail-cycles) to restore failing behavior. Broken links
still fail unconditionally."
```

---

### Task 3: Documentation updates

**Files:**
- Modify: `CLAUDE.md:33` (the `internal/rules/` bullet)
- Modify: `.claude/skills/rootline/SKILL.md:87-91` (checks YAML example)
- Modify: `.claude/skills/rootline/ref-validate.md:55` (link-check rules paragraph)

**Interfaces:**
- Consumes: final behavior from Task 2 (key name `cycles`, flag `--fail-cycles`, informational default).
- Produces: nothing consumed by other tasks.

- [ ] **Step 1: Update `CLAUDE.md`**

In the `internal/rules/` bullet (line 33), change:

```
`links.checks` (`resolve`/`anchors`/`encoding`) enables ADO code-wiki checks via `CheckLinks`
```

to:

```
`links.checks` (`resolve`/`anchors`/`encoding`) enables ADO code-wiki checks via `CheckLinks`; `links.checks.cycles: true` opts `graph --check` into failing on link cycles (default: cycles are informational; `--fail-cycles` overrides)
```

- [ ] **Step 2: Update `SKILL.md` example**

In the YAML block at `.claude/skills/rootline/SKILL.md:87-91`, add one line after `encoding: true`:

```yaml
    cycles: true       # graph --check fails on link cycles (default: informational)
```

- [ ] **Step 3: Update `ref-validate.md`**

Append one sentence to the link-check paragraph at line 55:

```
`checks.cycles: true` additionally makes `graph --check` fail on link cycles; without it cycles are printed as informational and only broken links set the exit code (override per-run with `--fail-cycles`).
```

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md .claude/skills/rootline/SKILL.md .claude/skills/rootline/ref-validate.md
git commit -m "docs: document checks.cycles opt-in and --fail-cycles override"
```

---

### Task 4: Verification, acceptance, and Definition of Done

**Files:**
- Modify: backlog under `/opt/factory/docs/backlog/` (add/update the entry for this feature as done)

**Interfaces:**
- Consumes: everything above, pushed to `master`.

- [ ] **Step 1: Local gates**

Run: `just check && just test && just coverage-check`
Expected: all green, no package below 85%.

- [ ] **Step 2: Local acceptance against DS Prima wiki (dev build)**

```bash
go build -o /tmp/rootline-dev ./cmd/rootline
cd /Users/Shared/dsprima/docs
/tmp/rootline-dev graph --check .; echo "exit=$?"
```

Expected: `exit=0`, output contains `Cycles found (informational): 22` (count may drift with wiki edits — informational header + exit 0 are the assertions), zero broken links. The wiki `.stem` needs no change — that is the point of the new default.

Canary:

```bash
echo '[x](no-existe.md)' > canary-cycle-check.md
/tmp/rootline-dev graph --check .; echo "exit=$?"
rm canary-cycle-check.md
```

Expected: `exit=1`, `Broken links: 1` reported alongside informational cycles.

Override check:

```bash
/tmp/rootline-dev graph --check --fail-cycles=true .; echo "exit=$?"
```

Expected: `exit=1` (cycles fail when forced).

- [ ] **Step 3: Update backlog**

Add the completed item to the current backlog file in `/opt/factory/docs/backlog/` (follow the existing entry format in that directory): feature `graph --check` opt-in cycle failure via `checks.cycles` + `--fail-cycles`, status done, date 2026-07-12, breaking change noted.

- [ ] **Step 4: Push and release**

```bash
cd /Users/Shared/harness/rootline
git push
```

Pre-push hook runs `just coverage-check` (Go files changed). CI (`go-release` via crossbeam) auto-tags a **minor** bump (pre-1.0 `feat!` → v0.x+1.0) and publishes binaries. Watch: `gh run watch` or `gh run list --limit 3`.

- [ ] **Step 5: Install released CLI and verify (DoD)**

The previous manual build reports `version dev` (no ldflags) — install from the release, not from `go build`:

```bash
# after the release workflow completes:
gh release view --json tagName -q .tagName   # confirm new tag
# install per the project's release artifacts (brew/binary download), then:
which rootline
rootline --version   # must print the NEW tag, not "dev"
```

- [ ] **Step 6: Re-run acceptance with the installed binary**

```bash
cd /Users/Shared/dsprima/docs
rootline graph --check .; echo "exit=$?"   # expected: exit=0, informational cycles
```

Done when: backlog updated, `just check` + `just test` green, commits pushed, `rootline --version` shows the new release, and the DS Prima acceptance passes with the installed binary.
