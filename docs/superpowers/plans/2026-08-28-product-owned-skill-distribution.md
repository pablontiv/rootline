---
estado: Specified
---

# Product-Owned Skill Distribution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an explicit, reversible `rootline skill` lifecycle that installs the repository-owned Rootline skill as symlinks for Claude and the shared Agents directory without hooks, mutable copies, or legacy compatibility code.

**Architecture:** A focused `internal/skilldist` package resolves a primary checkout, hashes the canonical tree, inventories exactly two supported destinations, creates a digest-bound plan, and applies approved actions with backups and append-only receipts. Cobra remains a thin JSON-only adapter. Installation is two-phase, idempotent, and fail-closed on drift; uninstall and restore use the same plan/approval boundary.

**Tech Stack:** Go 1.26+, Cobra, standard `os`, `io/fs`, `path/filepath`, `crypto/sha256`, `encoding/json`, Rootline's existing JSON output/`--field` machinery, standard `testing`.

**Spec:** `docs/superpowers/specs/2026-08-28-product-owned-skill-distribution-design.md`

## Global Constraints

- Work only on branch `fix/issue-210-skill-distribution`; never commit directly to `master`.
- Canonical source is exactly `.claude/skills/rootline` under the selected primary checkout; require `SKILL.md`, a `.git` directory at the checkout root, and no staged, unstaged, or untracked changes inside the canonical skill tree.
- Reject linked worktrees whose checkout-root `.git` is a file. The implementation worktree must never become an installed symlink target.
- Manage exactly `~/.claude/skills/rootline` and `~/.agents/skills/rootline`; OpenCode and Pi share the Agents destination.
- Production code must not name, inspect, migrate, or mutate `~/.config/opencode/skills/rootline` or any other obsolete runtime path.
- Breaking remediation lives in `CHANGELOG.md` and agent guidance, not in compatibility code.
- Mutating commands always use two invocations: plan first, then `--approve PLAN_DIGEST`, where `PLAN_DIGEST` is the exact emitted value.
- Do not add prompts, `--force`, copied active installations, junction fallback, shell delegation, or a non-JSON output.
- Back up every existing supported destination before replacement. Backups are recovery artifacts; active installations are symlinks only.
- Recompute the complete plan before applying. Any source or preimage drift invalidates approval before mutation.
- Execute destinations in stable `claude`, then `agents` order. Roll back the destination currently failing; retain earlier verified actions and seal the receipt incomplete.
- Receipts are JSON Lines, append-only, synced before returning, and never rewritten.
- Tests inject home/state/time/receipt IDs and use temporary paths. No test may read or mutate the real user home.
- Preserve one JSON envelope per verb and emit it before non-zero exit. Plan-only runs exit zero with `complete: false`; errors exit non-zero.
- Use TDD: observe each focused test fail for the intended reason before production edits.
- Use Conventional Commits with neutral English identifiers, comments, documentation, and commit messages.
- The delivery PR references #210 but does not auto-close it. Operational migration and closure happen only after merge from the primary checkout.

## File Structure

### New focused files

- `internal/skilldist/model.go` — public service model, destination/action enums, stable error codes, options, results, receipts, and plans.
- `internal/skilldist/digest.go` — deterministic tree and plan hashing.
- `internal/skilldist/digest_test.go` — deterministic hashing, content drift, and source-symlink refusal.
- `internal/skilldist/source.go` — primary-checkout source resolution and commit capture.
- `internal/skilldist/source_test.go` — primary checkout, missing skill, and linked-worktree refusal.
- `internal/skilldist/inventory.go` — supported destination definitions and `Lstat` classification.
- `internal/skilldist/inventory_test.go` — absent, directory, correct/divergent symlink, unsupported type, and legacy sentinel coverage.
- `internal/skilldist/plan.go` — deterministic install/status/uninstall/restore planning and plan digest.
- `internal/skilldist/plan_test.go` — idempotency and approval sensitivity.
- `internal/skilldist/store.go` — native state-root selection, append-only receipts, backups, and backup verification.
- `internal/skilldist/store_test.go` — append, sync-visible records, exclusive backups, and exact restoration.
- `internal/skilldist/service.go` — plan/apply orchestration, symlink publication, verification, rollback, status, uninstall, and restore.
- `internal/skilldist/service_test.go` — two-phase lifecycle, drift refusal, best-effort failure, uninstall, and restore.
- `cmd/rootline/skill.go` — Cobra command tree, flags, JSON envelopes, and service adapter.
- `cmd/rootline/skill_test.go` — command contracts, `--field`, exit status, and preflight exemption.
- `cmd/rootline/skill_e2e_test.go` — complete temporary-home lifecycle with real Git and filesystem symlinks.
- `cmd/rootline/skill_documentation_test.go` — breaking-change and agent-remediation documentation contract.
- `docs/skill.md` — user and agent lifecycle guide.

### Existing files modified

- `cmd/rootline/output.go` — declare the skill command group and four verbs JSON-only.
- `cmd/rootline/output_test.go` — assert JSON-only rejection behavior.
- `cmd/rootline/preflight.go` — exempt schema-independent `skill` commands.
- `cmd/rootline/commands_test.go` — reset new Cobra flags between tests.
- `.githooks/pre-push` — remove all skill installation mutation.
- `cmd/rootline/hooks_test.go` — replace copy expectation with a no-home-mutation contract.
- `CHANGELOG.md` — document the breaking destination contract and exact agent remediation.
- `README.md` — link the explicit skill lifecycle.
- `.claude/skills/rootline/SKILL.md` — require agents to read the breaking changelog entry before remediation.
- `CLAUDE.md` — describe the new CLI command and ownership boundary.

---

### Task 1: Resolve and hash the canonical primary-checkout source

**Files:**
- Create: `internal/skilldist/model.go`
- Create: `internal/skilldist/digest.go`
- Create: `internal/skilldist/digest_test.go`
- Create: `internal/skilldist/source.go`
- Create: `internal/skilldist/source_test.go`

**Interfaces:**
- Consumes: a requested repository path or current working directory; standard Git executable; canonical `.claude/skills/rootline` tree.
- Produces: `type Digest string`; `type Source struct {RepoRoot, SkillPath, Commit string; Digest Digest}`; `ResolveSource(ctx context.Context, requested string) (Source, error)`; `DigestTree(root string) (Digest, error)`; `type OperationError struct {Code ErrorCode; Path, Destination string; Err error}`.

- [ ] **Step 1: Write failing source and digest tests**

Create `internal/skilldist/digest_test.go` with deterministic and drift assertions:

```go
package skilldist

import (
    "os"
    "path/filepath"
    "testing"
)

func TestDigestTreeIsDeterministicAndContentSensitive(t *testing.T) {
    root := t.TempDir()
    mustWriteSkillFile(t, root, "z.md", "z")
    mustWriteSkillFile(t, root, "nested/a.md", "a")

    first, err := DigestTree(root)
    if err != nil {
        t.Fatalf("DigestTree: %v", err)
    }
    second, err := DigestTree(root)
    if err != nil {
        t.Fatalf("DigestTree second call: %v", err)
    }
    if first != second {
        t.Fatalf("digest changed without content change: %q != %q", first, second)
    }

    if err := os.WriteFile(filepath.Join(root, "nested", "a.md"), []byte("changed"), 0o644); err != nil {
        t.Fatal(err)
    }
    changed, err := DigestTree(root)
    if err != nil {
        t.Fatalf("DigestTree changed tree: %v", err)
    }
    if changed == first {
        t.Fatal("content change did not change digest")
    }
}

func TestDigestTreeHashesSymlinkTargetWithoutFollowingIt(t *testing.T) {
    root := t.TempDir()
    outside := filepath.Join(t.TempDir(), "outside.md")
    if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
        t.Fatal(err)
    }
    link := filepath.Join(root, "linked.md")
    if err := os.Symlink(outside, link); err != nil {
        t.Skipf("symlinks unavailable: %v", err)
    }
    first, err := DigestTree(root)
    if err != nil { t.Fatal(err) }
    if err := os.WriteFile(outside, []byte("changed outside"), 0o644); err != nil { t.Fatal(err) }
    second, err := DigestTree(root)
    if err != nil { t.Fatal(err) }
    if first != second {
        t.Fatal("DigestTree followed a symlink instead of hashing its lexical target")
    }
}
```

Create `internal/skilldist/source_test.go` with a real Git fixture:

```go
func TestResolveSourceRequiresPrimaryCheckoutAndCanonicalSkill(t *testing.T) {
    repo := initSkillRepository(t)
    source, err := ResolveSource(context.Background(), repo)
    if err != nil {
        t.Fatalf("ResolveSource: %v", err)
    }
    if source.RepoRoot != repo {
        t.Fatalf("RepoRoot = %q, want %q", source.RepoRoot, repo)
    }
    if source.SkillPath != filepath.Join(repo, ".claude", "skills", "rootline") {
        t.Fatalf("SkillPath = %q", source.SkillPath)
    }
    if source.Commit == "" || source.Digest == "" {
        t.Fatalf("incomplete source evidence: %#v", source)
    }
}

func TestResolveSourceRejectsLinkedWorktreeMarker(t *testing.T) {
    repo := t.TempDir()
    mustWriteSkillFile(t, filepath.Join(repo, ".claude", "skills", "rootline"), "SKILL.md", "---\nname: rootline\n---\n")
    if err := os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: /tmp/common/worktrees/probe\n"), 0o644); err != nil {
        t.Fatal(err)
    }
    _, err := ResolveSource(context.Background(), repo)
    assertOperationErrorCode(t, err, ErrLinkedWorktreeRefused)
}

func TestResolveSourceRejectsSymlinkInsideCanonicalTree(t *testing.T) {
    repo := initSkillRepository(t)
    outside := filepath.Join(t.TempDir(), "outside.md")
    if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil { t.Fatal(err) }
    if err := os.Symlink(outside, filepath.Join(repo, ".claude", "skills", "rootline", "escape.md")); err != nil {
        t.Skipf("symlinks unavailable: %v", err)
    }
    commitSkillRepository(t, repo, "add canonical symlink")
    _, err := ResolveSource(context.Background(), repo)
    assertOperationErrorCode(t, err, ErrSourceNotCanonical)
}

func TestResolveSourceRejectsDirtyCanonicalTree(t *testing.T) {
    repo := initSkillRepository(t)
    mustWriteSkillFile(t, filepath.Join(repo, ".claude", "skills", "rootline"), "untracked.md", "dirty")
    _, err := ResolveSource(context.Background(), repo)
    assertOperationErrorCode(t, err, ErrSourceNotCanonical)
}
```

The test helpers create parent directories, run `git init`, configure local author identity, add the skill tree, and commit it. `commitSkillRepository(t, repo, message)` runs `git add --all` and `git commit -m message`. Use `github.com/pablontiv/rootline/internal/gitenv.ClearedEnv()` on fixture Git commands so inherited hook Git variables cannot redirect them.

- [ ] **Step 2: Run the focused tests and verify the missing API failure**

Run:

```bash
go test ./internal/skilldist -run 'DigestTree|ResolveSource' -v
```

Expected: FAIL because the package and interfaces do not exist.

- [ ] **Step 3: Add the stable model and error codes**

Create `internal/skilldist/model.go` with these exact foundational declarations:

```go
package skilldist

import "fmt"

type Digest string

type ErrorCode string

const (
    ErrSourceNotCanonical    ErrorCode = "source_not_canonical"
    ErrLinkedWorktreeRefused ErrorCode = "linked_worktree_refused"
    ErrSourceDigestChanged   ErrorCode = "source_digest_changed"
    ErrPreimageDigestChanged ErrorCode = "preimage_digest_changed"
    ErrUnsupportedFileType   ErrorCode = "unsupported_file_type"
    ErrSymlinkPermission     ErrorCode = "symlink_permission_denied"
    ErrBackupFailed          ErrorCode = "backup_failed"
    ErrVerificationFailed    ErrorCode = "verification_failed"
    ErrRestoreConflict       ErrorCode = "restore_conflict"
)

type OperationError struct {
    Code        ErrorCode `json:"code"`
    Path        string    `json:"path,omitempty"`
    Destination string    `json:"destination,omitempty"`
    Message     string    `json:"message"`
    Err         error     `json:"-"`
}

func (e *OperationError) Error() string {
    if e.Message != "" {
        return e.Message
    }
    return fmt.Sprintf("%s: %v", e.Code, e.Err)
}

func operationError(code ErrorCode, path, destination string, err error) *OperationError {
    return &OperationError{Code: code, Path: path, Destination: destination, Message: err.Error(), Err: err}
}

type Source struct {
    RepoRoot string `json:"repo_root"`
    SkillPath string `json:"path"`
    Commit string `json:"commit"`
    Digest Digest `json:"digest"`
}
```

Keep `OperationError.Message` free of wrapped absolute-path duplication: `Path` carries the machine path separately.

- [ ] **Step 4: Implement deterministic tree hashing**

Create `internal/skilldist/digest.go`. Use `filepath.WalkDir`, which supplies lexical sibling order. `DigestTree` hashes symlink target text without following it so backups can preserve exact preimages; package-private `digestCanonicalTree` sets `rejectSymlinks=true` for source resolution:

```go
func DigestTree(root string) (Digest, error) { return digestTree(root, false) }
func digestCanonicalTree(root string) (Digest, error) { return digestTree(root, true) }

func digestTree(root string, rejectSymlinks bool) (Digest, error) {
    h := sha256.New()
    err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
        if walkErr != nil {
            return walkErr
        }
        rel, err := filepath.Rel(root, path)
        if err != nil {
            return err
        }
        if rel == "." {
            return nil
        }
        info, err := entry.Info()
        if err != nil {
            return err
        }
        kind := "file"
        if info.Mode()&os.ModeSymlink != 0 {
            if rejectSymlinks {
                return operationError(ErrSourceNotCanonical, path, "", fmt.Errorf("canonical skill contains a symlink"))
            }
            kind = "symlink"
        } else if entry.IsDir() {
            kind = "dir"
        } else if !info.Mode().IsRegular() {
            return fmt.Errorf("tree contains unsupported file type %s at %s", info.Mode().Type(), path)
        }
        normalized := filepath.ToSlash(rel)
        if _, err := fmt.Fprintf(h, "%d:%s%d:%s:%04o:%d", len(normalized), normalized, len(kind), kind, info.Mode().Perm(), info.Size()); err != nil {
            return err
        }
        if entry.IsDir() {
            return nil
        }
        if kind == "symlink" {
            target, err := os.Readlink(path)
            if err != nil { return err }
            _, err = fmt.Fprintf(h, "%d:%s", len(target), target)
            return err
        }
        file, err := os.Open(path)
        if err != nil {
            return err
        }
        _, copyErr := io.Copy(h, file)
        closeErr := file.Close()
        return errors.Join(copyErr, closeErr)
    })
    if err != nil {
        return "", err
    }
    return Digest("sha256:" + hex.EncodeToString(h.Sum(nil))), nil
}
```

- [ ] **Step 5: Implement primary-checkout resolution**

Create `internal/skilldist/source.go`. Canonicalize the requested path, execute `git -C requestedPath rev-parse --show-toplevel` and `git -C repoRoot rev-parse HEAD` with `gitenv.ClearedEnv()`, require `os.Stat(filepath.Join(repoRoot, ".git")).IsDir()`, then require `filepath.Join(repoRoot, ".claude", "skills", "rootline", "SKILL.md")` to be a regular file. Execute `git -C repoRoot status --porcelain -- .claude/skills/rootline` and reject any output so `Commit` and `Digest` describe one committed source. Return `ErrLinkedWorktreeRefused` for a `.git` non-directory and `ErrSourceNotCanonical` for every other source refusal.

The default requested path is resolved by the CLI, not by this function; `ResolveSource` always receives a non-empty path.

- [ ] **Step 6: Run focused and package tests**

Run:

```bash
go test ./internal/skilldist -run 'DigestTree|ResolveSource' -v
go test ./internal/gitenv ./internal/skilldist -race
```

Expected: PASS.

- [ ] **Step 7: Commit the canonical-source foundation**

```bash
git add internal/skilldist/model.go internal/skilldist/digest.go internal/skilldist/digest_test.go internal/skilldist/source.go internal/skilldist/source_test.go
git commit -m "feat(skill): resolve canonical skill source"
```

---

### Task 2: Inventory supported destinations and build immutable plans

**Files:**
- Modify: `internal/skilldist/model.go`
- Create: `internal/skilldist/inventory.go`
- Create: `internal/skilldist/inventory_test.go`
- Create: `internal/skilldist/plan.go`
- Create: `internal/skilldist/plan_test.go`

**Interfaces:**
- Consumes: `Source`, injected home path, supported operation, and optional receipt evidence.
- Produces: `SupportedDestinations(home string) []Destination`; `InventoryDestinations(home string, source Source) ([]DestinationState, error)`; `BuildPlan(operation Operation, source *Source, states []DestinationState, receipt *Receipt) (Plan, error)`; `Plan.Digest` bound to all observed evidence.

- [ ] **Step 1: Write failing inventory tests**

Create `internal/skilldist/inventory_test.go`:

```go
func TestInventoryDestinationsClassifiesOnlyClaudeAndAgents(t *testing.T) {
    home := t.TempDir()
    sourceRoot := t.TempDir()
    mustWriteSkillFile(t, sourceRoot, "SKILL.md", "canonical")
    sourceDigest, err := DigestTree(sourceRoot)
    if err != nil {
        t.Fatal(err)
    }
    source := Source{SkillPath: sourceRoot, Digest: sourceDigest}

    claude := filepath.Join(home, ".claude", "skills", "rootline")
    mustWriteSkillFile(t, claude, "SKILL.md", "copy")
    agents := filepath.Join(home, ".agents", "skills", "rootline")
    if err := os.MkdirAll(filepath.Dir(agents), 0o755); err != nil {
        t.Fatal(err)
    }
    if err := os.Symlink(sourceRoot, agents); err != nil {
        t.Skipf("symlinks unavailable: %v", err)
    }

    legacy := filepath.Join(home, ".config", "opencode", "skills", "rootline", "sentinel")
    mustWriteSkillFile(t, filepath.Dir(legacy), filepath.Base(legacy), "untouched")

    states, err := InventoryDestinations(home, source)
    if err != nil {
        t.Fatalf("InventoryDestinations: %v", err)
    }
    if len(states) != 2 || states[0].ID != DestinationClaude || states[1].ID != DestinationAgents {
        t.Fatalf("states = %#v", states)
    }
    if states[0].Kind != KindDirectory || states[1].Kind != KindCorrectSymlink {
        t.Fatalf("unexpected classifications: %#v", states)
    }
    data, err := os.ReadFile(legacy)
    if err != nil || string(data) != "untouched" {
        t.Fatalf("legacy sentinel changed: data=%q err=%v", data, err)
    }
}
```

Add table tests for absent, divergent symlink, regular file/FIFO where supported, and a correct symlink whose lexical target differs but canonical target matches. The latter is `KindDivergentSymlink`: installation identity requires both lexical and canonical agreement.

- [ ] **Step 2: Write failing plan tests**

Create `internal/skilldist/plan_test.go`:

```go
func TestInstallPlanIsStableIdempotentAndPreimageBound(t *testing.T) {
    source := Source{RepoRoot: "/repo", SkillPath: "/repo/.claude/skills/rootline", Commit: "abc", Digest: "sha256:source"}
    states := []DestinationState{
        {ID: DestinationClaude, Path: "/home/.claude/skills/rootline", Kind: KindDirectory, Digest: "sha256:old"},
        {ID: DestinationAgents, Path: "/home/.agents/skills/rootline", Kind: KindAbsent},
    }
    plan, err := BuildPlan(OperationInstall, &source, states, nil)
    if err != nil {
        t.Fatal(err)
    }
    if plan.Actions[0].Kind != ActionReplaceWithSymlink || plan.Actions[1].Kind != ActionCreateSymlink {
        t.Fatalf("actions = %#v", plan.Actions)
    }
    repeated, err := BuildPlan(OperationInstall, &source, states, nil)
    if err != nil {
        t.Fatal(err)
    }
    if plan.Digest != repeated.Digest {
        t.Fatalf("plan digest is unstable: %q != %q", plan.Digest, repeated.Digest)
    }
    states[0].Digest = "sha256:changed"
    changed, err := BuildPlan(OperationInstall, &source, states, nil)
    if err != nil {
        t.Fatal(err)
    }
    if changed.Digest == plan.Digest {
        t.Fatal("preimage change did not invalidate approval")
    }
}
```

Add a no-op test where both states are exact symlinks, and an unsupported-type test that returns `ErrUnsupportedFileType` before a plan is approved.

- [ ] **Step 3: Add destination and plan model types**

Extend `model.go` with exact stable enums and JSON fields:

```go
type DestinationID string
const (
    DestinationClaude DestinationID = "claude"
    DestinationAgents DestinationID = "agents"
)

type EntryKind string
const (
    KindAbsent EntryKind = "absent"
    KindDirectory EntryKind = "directory"
    KindCorrectSymlink EntryKind = "correct_symlink"
    KindDivergentSymlink EntryKind = "divergent_symlink"
    KindUnsupported EntryKind = "unsupported"
)

type Operation string
const (
    OperationInstall Operation = "install"
    OperationStatus Operation = "status"
    OperationUninstall Operation = "uninstall"
    OperationRestore Operation = "restore"
)

type ActionKind string
const (
    ActionCreateSymlink ActionKind = "create_symlink"
    ActionReplaceWithSymlink ActionKind = "replace_with_symlink"
    ActionRemoveManagedSymlink ActionKind = "remove_managed_symlink"
    ActionRestorePreimage ActionKind = "restore_preimage"
    ActionNoOp ActionKind = "no_op"
)

type Destination struct { ID DestinationID `json:"id"`; Path string `json:"path"` }
type DestinationState struct {
    ID DestinationID `json:"id"`
    Path string `json:"path"`
    Kind EntryKind `json:"kind"`
    Digest Digest `json:"digest,omitempty"`
    LexicalTarget string `json:"lexical_target,omitempty"`
    CanonicalTarget string `json:"canonical_target,omitempty"`
}
type Action struct { Kind ActionKind `json:"kind"`; Destination DestinationState `json:"destination"` }
type Plan struct {
    Operation Operation `json:"operation"`
    Source *Source `json:"source,omitempty"`
    ReceiptID string `json:"receipt_id,omitempty"`
    Actions []Action `json:"actions"`
    Digest Digest `json:"digest"`
}
```

- [ ] **Step 4: Implement inventory without legacy knowledge**

Create `inventory.go`. `SupportedDestinations` returns exactly two entries in fixed order. Use `os.Lstat`; hash directories with `DigestTree`; read symlinks with `os.Readlink`; resolve their canonical paths with `filepath.EvalSymlinks`; classify as correct only when `filepath.Clean(lexicalTarget) == filepath.Clean(source.SkillPath)`, canonical target equals the evaluated canonical source, and its tree digest equals `source.Digest`.

A regular file, socket, FIFO, device, or other non-directory/non-symlink returns a state with `KindUnsupported`; planning converts it to `ErrUnsupportedFileType`.

- [ ] **Step 5: Implement canonical plan hashing**

Create `plan.go`. Build actions in the input state order, reject any ID outside Claude/Agents, and hash a projection that excludes `Plan.Digest` itself:

```go
func digestPlan(plan Plan) (Digest, error) {
    canonical := struct {
        Operation Operation `json:"operation"`
        Source *Source `json:"source,omitempty"`
        ReceiptID string `json:"receipt_id,omitempty"`
        Actions []Action `json:"actions"`
    }{plan.Operation, plan.Source, plan.ReceiptID, plan.Actions}
    data, err := json.Marshal(canonical)
    if err != nil {
        return "", err
    }
    sum := sha256.Sum256(data)
    return Digest("sha256:" + hex.EncodeToString(sum[:])), nil
}
```

For install: absent → create; exact symlink → no-op; directory/divergent symlink → replace. Status produces no actions. Uninstall/restore reject until receipt interfaces from Task 3 are available; add their branches in Task 5 rather than inventing temporary behavior.

- [ ] **Step 6: Run focused tests**

```bash
go test ./internal/skilldist -run 'Inventory|InstallPlan' -v
go test ./internal/skilldist -race
```

Expected: PASS.

- [ ] **Step 7: Commit inventory and planning**

```bash
git add internal/skilldist/model.go internal/skilldist/inventory.go internal/skilldist/inventory_test.go internal/skilldist/plan.go internal/skilldist/plan_test.go
git commit -m "feat(skill): plan supported destinations"
```

---

### Task 3: Persist append-only receipts and exact backups

**Files:**
- Modify: `internal/skilldist/model.go`
- Create: `internal/skilldist/store.go`
- Create: `internal/skilldist/store_test.go`

**Interfaces:**
- Consumes: injected state root, receipt ID, destination state, and current filesystem preimage.
- Produces: `NewStore(stateRoot string) *Store`; `Store.Reserve(receiptID string) error`; `Store.Append(Receipt) error`; `Store.Load(id string) (Receipt, error)`; `Store.Latest() (Receipt, bool, error)`; `Store.Backup(receiptID string, state DestinationState) (Backup, error)`; `Store.VerifyBackup(Backup) error`; `Store.RestoreBackup(Backup, destination string) error`.

- [ ] **Step 1: Write failing receipt and backup tests**

Create `internal/skilldist/store_test.go`:

```go
func TestStoreAppendsReceiptsWithoutRewritingHistory(t *testing.T) {
    store := NewStore(t.TempDir())
    first := Receipt{Version: 1, Kind: "rootline/skill-receipt", ID: "r1", Operation: OperationInstall, Complete: true}
    second := Receipt{Version: 1, Kind: "rootline/skill-receipt", ID: "r2", Operation: OperationUninstall, Complete: false}
    if err := store.Append(first); err != nil { t.Fatal(err) }
    before, err := os.ReadFile(store.receiptsPath())
    if err != nil { t.Fatal(err) }
    if err := store.Append(second); err != nil { t.Fatal(err) }
    after, err := os.ReadFile(store.receiptsPath())
    if err != nil { t.Fatal(err) }
    if !bytes.HasPrefix(after, before) {
        t.Fatal("second append rewrote receipt history")
    }
    loaded, err := store.Load("r2")
    if err != nil || loaded.ID != "r2" {
        t.Fatalf("Load r2 = %#v, %v", loaded, err)
    }
}

func TestStoreBackupAndRestoreDirectoryExactly(t *testing.T) {
    stateRoot := t.TempDir()
    original := filepath.Join(t.TempDir(), "rootline")
    mustWriteSkillFile(t, original, "SKILL.md", "preimage")
    digest, err := DigestTree(original)
    if err != nil { t.Fatal(err) }
    store := NewStore(stateRoot)
    if err := store.Reserve("r1"); err != nil { t.Fatal(err) }
    backup, err := store.Backup("r1", DestinationState{ID: DestinationClaude, Path: original, Kind: KindDirectory, Digest: digest})
    if err != nil { t.Fatal(err) }
    if err := os.RemoveAll(original); err != nil { t.Fatal(err) }
    if err := store.RestoreBackup(backup, original); err != nil { t.Fatal(err) }
    restored, err := DigestTree(original)
    if err != nil || restored != digest {
        t.Fatalf("restored digest = %q, err=%v, want %q", restored, err, digest)
    }
}
```

Add tests proving `Reserve("r1")` fails on a duplicate receipt ID, two different destination backups succeed under one reserved receipt, each destination backup path is exclusive, symlink metadata restores exactly, malformed JSONL fails, and `Latest` returns `(Receipt{}, false, nil)` without creating the state directory.

- [ ] **Step 2: Run tests and verify missing store failures**

```bash
go test ./internal/skilldist -run 'Store' -v
```

Expected: FAIL because receipt/store interfaces do not exist.

- [ ] **Step 3: Add receipt and backup types**

Extend `model.go`:

```go
type Backup struct {
    Destination DestinationID `json:"destination"`
    OriginalPath string `json:"original_path"`
    StoredPath string `json:"stored_path,omitempty"`
    Kind EntryKind `json:"kind"`
    Digest Digest `json:"digest,omitempty"`
    LinkTarget string `json:"link_target,omitempty"`
}
type ActionResult struct {
    Destination DestinationID `json:"destination"`
    Action ActionKind `json:"action"`
    Before DestinationState `json:"before"`
    After DestinationState `json:"after"`
    Complete bool `json:"complete"`
    RolledBack bool `json:"rolled_back,omitempty"`
    Error *OperationError `json:"error,omitempty"`
}
type Receipt struct {
    Version int `json:"version"`
    Kind string `json:"kind"`
    ID string `json:"id"`
    Timestamp time.Time `json:"timestamp"`
    Operation Operation `json:"operation"`
    Complete bool `json:"complete"`
    Source *Source `json:"source,omitempty"`
    PlanDigest Digest `json:"plan_digest"`
    Actions []ActionResult `json:"actions"`
    Backups []Backup `json:"backups"`
    Errors []OperationError `json:"errors"`
}
```

Initialize all slices as non-nil so receipts remain stable JSON contracts. `Before` records absence as well as existing preimages; restore uses it to distinguish “remove the installed link” from “restore a backed-up directory or symlink.”

- [ ] **Step 4: Implement state-root and append semantics**

Create `store.go`. `NewStore` receives the already-selected state root and appends `rootline/skill` internally. `Append` creates the directory at `0o700`, opens `receipts.jsonl` with `os.O_CREATE|os.O_WRONLY|os.O_APPEND` and `0o600`, writes exactly one compact JSON object plus newline, calls `Sync`, then closes with `errors.Join`.

`Load` and `Latest` scan line-by-line, reject malformed records, and never rewrite. Duplicate receipt IDs are errors.

- [ ] **Step 5: Implement backup and restoration**

For a directory, recursively copy regular files/directories, recreate internal symlinks from their lexical targets without following them, preserve file permission bits, and verify the copied tree digest equals the observed preimage digest. For a destination that is itself a symlink, record `LinkTarget` and create no active copied tree. For absent, return no backup.

`Reserve` creates the receipt-ID backup directory with `os.Mkdir`, so an existing receipt ID fails exclusively. `Backup` requires that reservation and creates one exclusive child named by destination ID; this permits Claude and Agents backups in the same receipt without weakening collision protection. `RestoreBackup` refuses if the destination exists, recreates a directory from the stored tree or recreates the exact symlink target, then verifies digest/type.

- [ ] **Step 6: Run store and package tests**

```bash
go test ./internal/skilldist -run 'Store|Backup' -v
go test ./internal/skilldist -race
```

Expected: PASS.

- [ ] **Step 7: Commit receipt storage**

```bash
git add internal/skilldist/model.go internal/skilldist/store.go internal/skilldist/store_test.go
git commit -m "feat(skill): persist backups and receipts"
```

---

### Task 4: Apply install plans with verification and rollback

**Files:**
- Modify: `internal/skilldist/model.go`
- Create: `internal/skilldist/service.go`
- Create: `internal/skilldist/service_test.go`

**Interfaces:**
- Consumes: Tasks 1–3 interfaces and injected `Options`.
- Produces: `New(Options) (*Service, error)`; `Service.Install(ctx context.Context, sourcePath string, approval Digest) Result`; `Service.Status(ctx context.Context, sourcePath string) Result`; `Result.Failed() bool`.

- [ ] **Step 1: Write failing two-phase install tests**

Create `internal/skilldist/service_test.go`:

```go
func TestInstallRequiresExactPlanThenConvergesIdempotently(t *testing.T) {
    fixture := newServiceFixture(t)
    mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "old")

    planned := fixture.service.Install(context.Background(), fixture.repo, "")
    if planned.Attempted || planned.Complete || planned.Plan == nil || planned.Plan.Digest == "" {
        t.Fatalf("unexpected plan result: %#v", planned)
    }
    if _, err := os.Lstat(fixture.claudePath()); err != nil {
        t.Fatalf("plan mutated preimage: %v", err)
    }

    applied := fixture.service.Install(context.Background(), fixture.repo, planned.Plan.Digest)
    if applied.Failed() || !applied.Attempted || !applied.Complete || applied.Receipt == nil {
        t.Fatalf("apply result: %#v", applied)
    }
    assertSymlinkTo(t, fixture.claudePath(), fixture.skillPath())
    assertSymlinkTo(t, fixture.agentsPath(), fixture.skillPath())

    repeated := fixture.service.Install(context.Background(), fixture.repo, "")
    for _, action := range repeated.Plan.Actions {
        if action.Kind != ActionNoOp {
            t.Fatalf("repeat action = %#v, want no-op", action)
        }
    }
}

func TestInstallRejectsStalePreimageApprovalBeforeMutation(t *testing.T) {
    fixture := newServiceFixture(t)
    mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "first")
    planned := fixture.service.Install(context.Background(), fixture.repo, "")
    mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "changed")

    result := fixture.service.Install(context.Background(), fixture.repo, planned.Plan.Digest)
    if !result.Failed() || result.Attempted {
        t.Fatalf("stale approval result = %#v", result)
    }
    assertResultErrorCode(t, result, ErrPreimageDigestChanged)
    data, err := os.ReadFile(filepath.Join(fixture.claudePath(), "SKILL.md"))
    if err != nil || string(data) != "changed" {
        t.Fatalf("stale approval mutated preimage: data=%q err=%v", data, err)
    }
}
```

Add a source-content drift test, a verifier test for lexical target mismatch, and a best-effort test that installs Claude, triggers an internal package test failpoint before Agents publication, rolls Agents back, retains Claude, and writes an incomplete receipt. The failpoint belongs to the unexported executor struct, not public `Options`. Add a test that recreates the final path after its preimage is staged: direct `os.Symlink` must fail without overwriting the recreated entry, rollback must report incomplete, and the independent backup must remain valid. Add a table test proving `os.ErrPermission`, `syscall.EPERM`, and `syscall.EACCES` from symlink creation map to `ErrSymlinkPermission`, while verification failures retain `ErrVerificationFailed`.

- [ ] **Step 2: Run service tests and verify missing service failure**

```bash
go test ./internal/skilldist -run 'Install|Status' -v
```

Expected: FAIL because `Service`, `Options`, and `Result` do not exist.

- [ ] **Step 3: Add service options and result model**

Extend `model.go`:

```go
type Options struct {
    HomeDir string
    StateDir string
    Now func() time.Time
    NewReceiptID func() string
}
type Result struct {
    Operation Operation `json:"operation"`
    Attempted bool `json:"attempted"`
    Complete bool `json:"complete"`
    Source *Source `json:"source,omitempty"`
    Plan *Plan `json:"plan,omitempty"`
    Destinations []DestinationState `json:"destinations"`
    Backups []Backup `json:"backups"`
    Receipt *Receipt `json:"receipt,omitempty"`
    ReceiptDrift bool `json:"receipt_drift"`
    Errors []OperationError `json:"errors"`
}
func (r Result) Failed() bool { return len(r.Errors) > 0 }
```

`New` requires non-empty home/state paths after applying production defaults, supplies `time.Now` and a cryptographically random 128-bit hex receipt ID when omitted, and returns a service with fixed supported destination order.

- [ ] **Step 4: Implement plan-only and stale-approval behavior**

`Install` resolves the source, inventories destinations, builds the plan, and returns immediately when approval is empty. When approval differs from the recomputed plan digest, return `ErrPreimageDigestChanged` before mutation. The stateless CLI receives only the old digest, not the old plan body, so it cannot truthfully attribute that mismatch to one field. `ErrSourceDigestChanged` is reserved for a source change detected after approval validation but before or during publication.

Do not add a plan cache or persistent pending plans. The approval digest is authorization, not stored state.

- [ ] **Step 5: Implement backup, publication, and verification**

Reserve the receipt ID once before iterating. For each non-no-op action:

```go
if err := s.store.Reserve(receipt.ID); err != nil {
    return failedResult(ErrBackupFailed, err)
}
for _, action := range plan.Actions {
    if action.Kind == ActionNoOp {
        recordActionSuccess(action)
        continue
    }
    backup, err := s.store.Backup(receipt.ID, action.Destination)
    if err != nil {
        recordFailure(ErrBackupFailed, action.Destination, err)
        break
    }
    if backup.Kind != KindAbsent {
        receipt.Backups = append(receipt.Backups, backup)
    }
    if err := executor.publishSymlink(action.Destination, source); err != nil {
        rolledBack := executor.rollback(action.Destination, backup)
        recordActionFailure(action, err, rolledBack)
        break
    }
    recordActionSuccess(action)
}
```

`publishSymlink` moves an existing destination to a unique sibling staging name, re-inventories that moved preimage and compares it to the approved state, then calls `os.Symlink(source.SkillPath, destinationPath)` directly. Direct creation is atomic and fails rather than overwriting a path recreated concurrently. Remove the staged preimage only after verifier success. On failure, restore it only if the final path remains absent; otherwise preserve the external entry, remove staging only after confirming the independent backup, and report rollback incomplete. Permission errors from `os.Symlink` map to `ErrSymlinkPermission`; all lexical/canonical/digest mismatches map to `ErrVerificationFailed`.

Seal the receipt once, append it even when incomplete, and set `Result.Complete` exactly to `Receipt.Complete`. A plan containing only no-ops still produces a complete receipt when explicitly approved.

- [ ] **Step 6: Implement read-only status**

`Status` resolves source and inventories both destinations. It reads `Store.Latest()` without creating state. `Complete` is true only when the source is valid and both destinations are exact symlinks; `Attempted` remains false. Set `ReceiptDrift` when the current source digest or any destination evidence differs from the latest receipt, while keeping convergence with the current source visible through `Complete`. Receipt drift is informational, not a mutating-operation error. Add a test that commits a canonical skill update after installation: status remains complete because both symlinks resolve to the current source and reports `receipt_drift:true` because the receipt names the prior digest.

- [ ] **Step 7: Run service tests and race detector**

```bash
go test ./internal/skilldist -run 'Install|Status|Verification|Rollback' -v
go test ./internal/skilldist -race
```

Expected: PASS.

- [ ] **Step 8: Commit install lifecycle**

```bash
git add internal/skilldist/model.go internal/skilldist/service.go internal/skilldist/service_test.go
git commit -m "feat(skill): apply verified skill installs"
```

---

### Task 5: Add approved uninstall and restore flows

**Files:**
- Modify: `internal/skilldist/plan.go`
- Modify: `internal/skilldist/plan_test.go`
- Modify: `internal/skilldist/service.go`
- Modify: `internal/skilldist/service_test.go`

**Interfaces:**
- Consumes: latest or selected `Receipt`, current destination inventory, verified backups.
- Produces: `Service.Uninstall(ctx context.Context, approval Digest) Result`; `Service.Restore(ctx context.Context, receiptID string, approval Digest) Result`.

- [ ] **Step 1: Write failing uninstall and restore tests**

Append to `service_test.go`:

```go
func TestUninstallRemovesOnlyIntactReceiptedSymlinks(t *testing.T) {
    fixture := newServiceFixture(t)
    plan := fixture.service.Install(context.Background(), fixture.repo, "")
    installed := fixture.service.Install(context.Background(), fixture.repo, plan.Plan.Digest)
    if installed.Failed() { t.Fatalf("install: %#v", installed) }

    uninstallPlan := fixture.service.Uninstall(context.Background(), "")
    if uninstallPlan.Attempted || uninstallPlan.Plan == nil {
        t.Fatalf("uninstall plan = %#v", uninstallPlan)
    }
    removed := fixture.service.Uninstall(context.Background(), uninstallPlan.Plan.Digest)
    if removed.Failed() || !removed.Complete { t.Fatalf("uninstall = %#v", removed) }
    for _, path := range []string{fixture.claudePath(), fixture.agentsPath()} {
        if _, err := os.Lstat(path); !os.IsNotExist(err) {
            t.Fatalf("%s still exists: %v", path, err)
        }
    }
}

func TestRestoreRecreatesRecordedDirectoryPreimage(t *testing.T) {
    fixture := newServiceFixture(t)
    mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "original")
    plan := fixture.service.Install(context.Background(), fixture.repo, "")
    installed := fixture.service.Install(context.Background(), fixture.repo, plan.Plan.Digest)
    restorePlan := fixture.service.Restore(context.Background(), installed.Receipt.ID, "")
    restored := fixture.service.Restore(context.Background(), installed.Receipt.ID, restorePlan.Plan.Digest)
    if restored.Failed() || !restored.Complete { t.Fatalf("restore = %#v", restored) }
    data, err := os.ReadFile(filepath.Join(fixture.claudePath(), "SKILL.md"))
    if err != nil || string(data) != "original" {
        t.Fatalf("restored preimage data=%q err=%v", data, err)
    }
}
```

Add tests that uninstall refuses an unreceipted or retargeted symlink, restore refuses a missing/corrupt backup, restore approval changes when current state drifts, and uninstall does not restore old directories automatically.

- [ ] **Step 2: Run focused tests and observe missing methods**

```bash
go test ./internal/skilldist -run 'Uninstall|Restore' -v
```

Expected: FAIL because the methods and plan branches do not exist.

- [ ] **Step 3: Implement uninstall planning and execution**

Load the latest complete install receipt. For each supported destination, require current state `KindCorrectSymlink` and receipt evidence naming that destination/source. Build `ActionRemoveManagedSymlink`; absent or changed state returns `ErrRestoreConflict` rather than silently skipping. Approval recomputes the same plan, removes each intact symlink, verifies absence, records a complete/incomplete uninstall receipt, and never invokes `RestoreBackup`.

- [ ] **Step 4: Implement restore planning and execution**

Load the requested receipt and read each action's `Before` state. Verify every backup required by a non-absent `Before` state, then pair evidence with fixed supported destinations. An absent preimage yields removal of the currently receipted link; a directory or symlink preimage yields `ActionRestorePreimage`. Include current destination evidence and receipt ID in the plan digest.

On apply, re-plan, back up any current state that the restore itself will replace into the new restore receipt, remove the current receipted installation, call `Store.RestoreBackup`, verify exact kind/digest/target, and use the same per-destination rollback rule on failure.

- [ ] **Step 5: Run all package tests**

```bash
go test ./internal/skilldist -run 'Uninstall|Restore' -v
go test ./internal/skilldist -race
```

Expected: PASS.

- [ ] **Step 6: Commit recovery lifecycle**

```bash
git add internal/skilldist/plan.go internal/skilldist/plan_test.go internal/skilldist/service.go internal/skilldist/service_test.go
git commit -m "feat(skill): add uninstall and restore"
```

---

### Task 6: Expose the JSON-only `rootline skill` command group

**Files:**
- Create: `cmd/rootline/skill.go`
- Create: `cmd/rootline/skill_test.go`
- Modify: `cmd/rootline/output.go`
- Modify: `cmd/rootline/output_test.go`
- Modify: `cmd/rootline/preflight.go`
- Modify: `cmd/rootline/commands_test.go`

**Interfaces:**
- Consumes: `skilldist.New`, four service methods, `outputJSON`, global `--output` and `--field`.
- Produces: `rootline/skill-install`, `rootline/skill-status`, `rootline/skill-uninstall`, and `rootline/skill-restore` version-1 envelopes.

- [ ] **Step 1: Write failing command contract tests**

Create `cmd/rootline/skill_test.go`:

```go
func TestSkillInstallPlanApplyAndFieldExtraction(t *testing.T) {
    fixture := newSkillCommandFixture(t)
    out, err := runCmd(t, "skill", "install", "--source", fixture.repo, "--field", "plan_digest")
    if err != nil { t.Fatalf("plan: %v\n%s", err, out) }
    var digest string
    if err := json.Unmarshal([]byte(out), &digest); err != nil || digest == "" {
        t.Fatalf("plan_digest output=%q err=%v", out, err)
    }
    out, err = runCmd(t, "skill", "install", "--source", fixture.repo, "--approve", digest)
    if err != nil { t.Fatalf("apply: %v\n%s", err, out) }
    var result SkillEnvelope
    if err := json.Unmarshal([]byte(out), &result); err != nil { t.Fatal(err) }
    if result.Kind != "rootline/skill-install" || !result.Complete || result.Receipt == nil {
        t.Fatalf("result = %#v", result)
    }
}

func TestSkillCommandsRejectNonJSONOutput(t *testing.T) {
    fixture := newSkillCommandFixture(t)
    _, err := runCmd(t, "skill", "status", "--source", fixture.repo, "-o", "table")
    if err == nil || !strings.Contains(err.Error(), "does not support output format") {
        t.Fatalf("error = %v", err)
    }
}

func TestSkillCommandIsBoundaryPreflightExempt(t *testing.T) {
    fixture := newSkillCommandFixture(t)
    out, err := runCmd(t, "skill", "status", "--source", fixture.repo)
    if err != nil || strings.Contains(out, "declared governance boundary") {
        t.Fatalf("status was governance-gated: out=%s err=%v", out, err)
    }
}
```

Add tests for plan-only exit zero with `complete:false`, stale approval payload before non-zero exit, required `--receipt`, all four kinds, and `--field receipt.id`.

- [ ] **Step 2: Run focused command tests**

```bash
go test ./cmd/rootline -run 'Skill' -v
```

Expected: FAIL because the command and envelope do not exist.

- [ ] **Step 3: Add the Cobra command tree and envelope**

Create `cmd/rootline/skill.go` with package variables resettable by tests:

```go
var skillSource string
var skillApproval string
var skillReceipt string

type SkillEnvelope struct {
    Version int `json:"version"`
    Kind string `json:"kind"`
    Complete bool `json:"complete"`
    Source *skilldist.Source `json:"source,omitempty"`
    PlanDigest skilldist.Digest `json:"plan_digest,omitempty"`
    Destinations []skilldist.DestinationState `json:"destinations"`
    Backups []skilldist.Backup `json:"backups"`
    Receipt *skilldist.Receipt `json:"receipt,omitempty"`
    ReceiptDrift bool `json:"receipt_drift"`
    Errors []skilldist.OperationError `json:"errors"`
}
```

Define `skill`, `install`, `status`, `uninstall`, and `restore` Cobra commands. `install/status` accept `--source`; mutating verbs accept `--approve`; restore requires `--receipt`. Resolve empty source with `os.Getwd()` before constructing the service.

Add `newSkillService()` that uses `os.UserHomeDir`, selects state root as `XDG_STATE_HOME` on Unix or `$HOME/.local/state`, and uses `os.UserConfigDir` on Windows. Keep a package-level `skillServiceFactory` only if command tests cannot reliably inject environment paths; reset it after every test with `t.Cleanup`.

Map `skilldist.Result` into `SkillEnvelope`; call `outputJSON(cmd, envelope, result.Failed())`. Plan digest comes from `result.Plan.Digest` when a plan exists. Initialize all envelope slices to empty arrays.

- [ ] **Step 4: Register JSON formats and preflight exemption**

Add to `commandOutputFormats`:

```go
"rootline skill":           {"json"},
"rootline skill install":   {"json"},
"rootline skill status":    {"json"},
"rootline skill uninstall": {"json"},
"rootline skill restore":   {"json"},
```

Add `"skill": true` to `commandsExemptFromBoundaryPreflight` and update its schema-independent comment. Extend `output_test.go` with a skill table-format rejection case.

- [ ] **Step 5: Reset flags and verify envelope/exit binding**

Extend `resetFlags()` to set `source`, `approve`, and `receipt` flags to empty and clear their `Changed` bits on the relevant skill subcommands.

Add a table test asserting `result.Failed()` exactly matches `err != nil` while plan-only remains exit zero despite `complete:false`. This preserves the intentional distinction between “not attempted” and “attempted incompletely.”

- [ ] **Step 6: Run command and output tests**

```bash
go test ./cmd/rootline -run 'Skill|CommandOutputFormats|Output_RejectsUnsupported' -v
go test ./cmd/rootline -race
```

Expected: PASS.

- [ ] **Step 7: Commit the CLI contract**

```bash
git add cmd/rootline/skill.go cmd/rootline/skill_test.go cmd/rootline/output.go cmd/rootline/output_test.go cmd/rootline/preflight.go cmd/rootline/commands_test.go
git commit -m "feat(skill): expose managed skill lifecycle"
```

---

### Task 7: Remove hook mutation and publish the breaking agent contract

**Files:**
- Modify: `.githooks/pre-push`
- Modify: `cmd/rootline/hooks_test.go`
- Create: `cmd/rootline/skill_documentation_test.go`
- Create: `docs/skill.md`
- Modify: `CHANGELOG.md`
- Modify: `README.md`
- Modify: `.claude/skills/rootline/SKILL.md`
- Modify: `CLAUDE.md`

**Interfaces:**
- Consumes: final CLI names and JSON fields from Task 6.
- Produces: non-mutating hook contract and exact install/update/verify/uninstall/restore instructions.

- [ ] **Step 1: Replace the hook test with a failing negative contract**

Rename `TestPrePushSyncsSkillWithoutInstallingBinary` to `TestPrePushDoesNotMutateHome` and use a sentinel tree:

```go
func TestPrePushDoesNotMutateHome(t *testing.T) {
    if runtime.GOOS == "windows" { t.Skip("pre-push is a Bash hook") }
    repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
    if err != nil { t.Fatal(err) }
    testHome := t.TempDir()
    sentinel := filepath.Join(testHome, ".claude", "skills", "rootline", "SKILL.md")
    mustWriteFile(t, sentinel, []byte("do not replace\n"), 0o600)

    cmd := exec.Command(filepath.Join(repoRoot, ".githooks", "pre-push"), "origin", "unused")
    cmd.Dir = repoRoot
    cmd.Stdin = strings.NewReader("")
    cmd.Env = append(os.Environ(), "HOME="+testHome)
    output, err := cmd.CombinedOutput()
    if err != nil { t.Fatalf("pre-push failed: %v\n%s", err, output) }
    data, err := os.ReadFile(sentinel)
    if err != nil || string(data) != "do not replace\n" {
        t.Fatalf("pre-push mutated home: data=%q err=%v", data, err)
    }
}
```

Run:

```bash
go test ./cmd/rootline -run TestPrePushDoesNotMutateHome -v
```

Expected: FAIL because the current hook replaces the sentinel.

- [ ] **Step 2: Remove the pre-push installation block**

Delete everything from `# Install the project Rootline skill to the Claude user scope.` through its final `echo`. Do not add a warning, status probe, or replacement hook behavior.

Run the focused hook test again; expect PASS.

- [ ] **Step 3: Write failing documentation contract tests**

Create `cmd/rootline/skill_documentation_test.go`:

```go
func TestSkillDistributionDocumentationContract(t *testing.T) {
    repo := documentationContractRepoRoot(t)
    required := map[string][]string{
        "CHANGELOG.md": {
            "**BREAKING**", "~/.config/opencode/skills/rootline", "~/.agents/skills/rootline", "rootline skill install", "rootline skill status",
        },
        "docs/skill.md": {
            "rootline skill install", "--approve", "rootline skill uninstall", "rootline skill restore", "receipts.jsonl",
        },
        ".claude/skills/rootline/SKILL.md": {
            "CHANGELOG.md", "rootline skill status", "~/.agents/skills/rootline",
        },
        "README.md": {"docs/skill.md", "rootline skill install"},
        "CLAUDE.md": {"rootline skill", "internal/skilldist"},
    }
    for path, needles := range required {
        text := string(documentationContractRead(t, repo, path))
        for _, needle := range needles {
            if !strings.Contains(text, needle) {
                t.Errorf("%s missing %q", path, needle)
            }
        }
    }
}
```

Run:

```bash
go test ./cmd/rootline -run SkillDistributionDocumentationContract -v
```

Expected: FAIL naming missing documentation strings.

- [ ] **Step 4: Add exact breaking guidance to `CHANGELOG.md`**

Under `[Unreleased]` → `Changed`, add:

```markdown
- **BREAKING**: Rootline no longer installs or supports the former OpenCode-specific `~/.config/opencode/skills/rootline` destination. OpenCode and Pi consume the shared `~/.agents/skills/rootline` symlink; Claude consumes `~/.claude/skills/rootline`. Agents upgrading an older environment must inspect and preserve the former destination themselves, retire it deliberately, run `rootline skill install` from the primary Rootline checkout, apply the emitted `plan_digest` with `--approve`, and verify with `rootline skill status`. Rootline does not discover or mutate obsolete runtime paths.
```

Do not put the obsolete path literal in production Go or the canonical skill; the changelog is the sole migration history.

- [ ] **Step 5: Write `docs/skill.md` and update active guidance**

Document these exact workflows with JSON examples:

```bash
plan=$(rootline skill install --source /stable/rootline --field plan_digest | jq -r .)
receipt_id=$(rootline skill install --source /stable/rootline --approve "$plan" --field receipt.id | jq -r .)
rootline skill status --source /stable/rootline
uninstall_plan=$(rootline skill uninstall --field plan_digest | jq -r .)
rootline skill uninstall --approve "$uninstall_plan"
restore_plan=$(rootline skill restore --receipt "$receipt_id" --field plan_digest | jq -r .)
rootline skill restore --receipt "$receipt_id" --approve "$restore_plan"
```

Explain that `receipt_id` is the exact value emitted by `--field receipt.id`. Also explain primary-checkout refusal, plan drift, state-root locations, backup ownership, best-effort receipts, Windows symlink permission, and issue #210's post-merge operational gate.

Update README with a short “Agent skill distribution” paragraph linking the guide. Update the canonical skill with a concise rule: before correcting an installation created under an older Rootline release, read the relevant `CHANGELOG.md` breaking entry; use only the currently documented destinations. Explain that `receipt_drift:true` after a committed source update is resolved by approving an idempotent install plan before uninstall. Update CLAUDE.md package layout and CLI command summary.

- [ ] **Step 6: Run documentation and hook contracts**

```bash
go test ./cmd/rootline -run 'PrePushDoesNotMutateHome|SkillDistributionDocumentationContract' -v
rootline validate --all docs/adr/
```

Expected: tests PASS; ADR validation reports all records valid.

- [ ] **Step 7: Commit hook and documentation boundary**

```bash
git add .githooks/pre-push cmd/rootline/hooks_test.go cmd/rootline/skill_documentation_test.go docs/skill.md CHANGELOG.md README.md .claude/skills/rootline/SKILL.md CLAUDE.md
git commit -m "docs(skill): publish explicit distribution workflow"
```

---

### Task 8: Prove the complete lifecycle and repository quality gates

**Files:**
- Create: `cmd/rootline/skill_e2e_test.go`
- Modify only if a verified defect is found: files introduced in Tasks 1–7.

**Interfaces:**
- Consumes: compiled Cobra command tree and all lifecycle contracts.
- Produces: one acceptance test proving plan → install → status → uninstall → restore while preserving an obsolete-path sentinel.

- [ ] **Step 1: Write the failing end-to-end lifecycle test**

Create `cmd/rootline/skill_e2e_test.go`:

```go
func TestSkillLifecycleEndToEnd(t *testing.T) {
    fixture := newSkillCommandFixture(t)
    oldClaude := filepath.Join(fixture.home, ".claude", "skills", "rootline", "SKILL.md")
    mustWriteFile(t, oldClaude, []byte("old claude\n"), 0o600)
    legacy := filepath.Join(fixture.home, ".config", "opencode", "skills", "rootline", "sentinel")
    mustWriteFile(t, legacy, []byte("legacy untouched\n"), 0o600)

    installDigest := runSkillField(t, "plan_digest", "skill", "install", "--source", fixture.repo)
    installOut, installErr := runCmd(t, "skill", "install", "--source", fixture.repo, "--approve", installDigest)
    if installErr != nil {
        if runtime.GOOS != "windows" || !skillEnvelopeHasCode(t, installOut, "symlink_permission_denied") {
            t.Fatalf("install: %v\n%s", installErr, installOut)
        }
        assertNoActiveCopiedInstallation(t, fixture)
        legacyData, err := os.ReadFile(legacy)
        if err != nil || string(legacyData) != "legacy untouched\n" {
            t.Fatalf("obsolete-path sentinel changed: data=%q err=%v", legacyData, err)
        }
        return
    }
    receiptID := decodeSkillReceiptID(t, installOut)

    statusOut := runSkillSuccess(t, "skill", "status", "--source", fixture.repo)
    assertSkillStatusConverged(t, statusOut)

    uninstallDigest := runSkillField(t, "plan_digest", "skill", "uninstall")
    runSkillSuccess(t, "skill", "uninstall", "--approve", uninstallDigest)

    restoreDigest := runSkillField(t, "plan_digest", "skill", "restore", "--receipt", receiptID)
    runSkillSuccess(t, "skill", "restore", "--receipt", receiptID, "--approve", restoreDigest)

    data, err := os.ReadFile(oldClaude)
    if err != nil || string(data) != "old claude\n" {
        t.Fatalf("Claude preimage not restored: data=%q err=%v", data, err)
    }
    legacyData, err := os.ReadFile(legacy)
    if err != nil || string(legacyData) != "legacy untouched\n" {
        t.Fatalf("obsolete-path sentinel changed: data=%q err=%v", legacyData, err)
    }
}
```

The fixture injects `HOME` and `XDG_STATE_HOME`, initializes and commits a primary Git repository, and registers cleanup that restores environment and Cobra flags. It never points at the implementation worktree.

- [ ] **Step 2: Run the E2E test and fix only contract defects**

```bash
go test ./cmd/rootline -run TestSkillLifecycleEndToEnd -v
```

Expected: PASS once Tasks 1–7 are complete. If it fails, preserve the public interfaces and correct the smallest responsible package; add a regression assertion before production edits.

- [ ] **Step 3: Run proactive diagnostics**

Run LSP diagnostics on:

```text
internal/skilldist/
cmd/rootline/skill.go
cmd/rootline/skill_test.go
cmd/rootline/skill_e2e_test.go
cmd/rootline/hooks_test.go
```

Expected: no errors. Treat Spanish-only spelling hints in historical ADRs as false positives; do not rewrite accepted historical prose for an unrelated implementation.

- [ ] **Step 4: Run focused package and race tests**

```bash
go test ./internal/skilldist -race
go test ./cmd/rootline -run 'Skill|PrePushDoesNotMutateHome|CommandOutputFormats' -race
```

Expected: PASS.

- [ ] **Step 5: Run full repository verification**

```bash
just test
just check
just coverage-check
rootline validate --all docs/adr/
```

Expected: all commands exit zero; every package remains at or above its configured 85% floor; ADR records are valid.

- [ ] **Step 6: Verify no legacy production knowledge or copied-install fallback exists**

Run:

```bash
if rg -n '\.config/opencode/skills|cp -r .*skills|CopyFile.*skill' internal/skilldist cmd/rootline/skill.go .githooks/pre-push; then
  echo "unexpected legacy or copy-based production path" >&2
  exit 1
fi
```

Expected: no matches and exit zero. Documentation may describe the breaking history without teaching production code the obsolete path.

- [ ] **Step 7: Commit E2E acceptance**

```bash
git add cmd/rootline/skill_e2e_test.go
git commit -m "test(skill): cover managed lifecycle"
```

- [ ] **Step 8: Prepare PR evidence without mutating the real home**

Capture:

```bash
git status --short
git log --oneline origin/master..HEAD
git diff --stat origin/master...HEAD
gh issue view 210 --repo pablontiv/rootline --json number,state,title,url
```

Expected: clean worktree; the commit range contains the design plus implementation commits; issue #210 remains open. The PR body must say `Refs #210`, list the temporary-home acceptance evidence, and state that real-home migration/closure remains a post-merge gate.
