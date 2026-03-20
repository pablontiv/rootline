# Design: Graph HTML Viewer, Remote Templates, Structural Inference

**Date**: 2026-03-20
**Status**: Approved
**Scope**: 3 independent features that advance rootline's visual tooling, adoption, and inference completeness

---

## Context

Rootline's engine and MCP server are complete (363/369 roadmap items done). The feasibility assessment of `docs/research/` identified three natural next steps from `roi-growth-strategy.md` and `inference-engine-architecture.md`:

1. **Graph HTML Viewer** — `rootline graph --open` for interactive visualization (Phase 2: Visual Impact)
2. **Remote Templates** — `rootline init --template <repo>` for schema distribution (Phase 2: Adoption + Schema Registry precursor)
3. **Structural Inference** — category 4 detector completing the inference engine (12/13 → 13/13)

All three are independent and can be implemented in parallel.

---

## Feature 1: `rootline graph --open`

### What It Does

Generates a temporary HTML file with an embedded Mermaid diagram and opens it in the default browser.

### Interface

```bash
rootline graph docs/epics/ --open                     # opens Mermaid diagram in browser
rootline graph docs/epics/ --open --where "tipo=='feature'"  # filtered graph
```

### Behavior

- `--open` flag (BoolVar) on `graph` command
- When `--open`: generates Mermaid text → embeds in HTML template → writes to `os.CreateTemp("", "rootline-graph-*.html")` → opens with platform browser command
- Prints path to stderr: `Opened: /tmp/rootline-graph-XXXX.html`
- `--open` implies `--format mermaid` (ignores `--format dot`)
- `--open` + `--check` is an error (no diagram to show)
- Temporary file is NOT cleaned up — user can find it later

### Implementation

**New files:**
- `internal/graph/templates/graph.html` — HTML template with Mermaid.js CDN embed
- `internal/graph/html.go` — `RenderHTML(mermaidContent string) (string, error)` + `OpenBrowser(path string) error`

**Modified files:**
- `cmd/rootline/graph.go` — add `--open` flag, branch logic
- `cmd/rootline/graph_test.go` — test HTML generation (without browser open)

### HTML Template

```html
<!DOCTYPE html>
<html><head>
  <meta charset="utf-8">
  <title>Rootline Graph</title>
  <script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
</head><body>
  <pre class="mermaid">{{.Content}}</pre>
  <script>mermaid.initialize({startOnLoad: true, theme: 'default'});</script>
</body></html>
```

Template embedded via `go:embed`.

### Browser Opening

```go
func OpenBrowser(url string) error {
    switch runtime.GOOS {
    case "linux":
        return exec.Command("xdg-open", url).Start()
    case "darwin":
        return exec.Command("open", url).Start()
    case "windows":
        return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
    default:
        return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
    }
}
```

### Tests

- HTML output contains valid Mermaid content
- `--open` + `--check` produces error
- `--open` overrides `--format dot` to mermaid
- OpenBrowser function is tested via interface (injectable opener for unit tests)

### Estimated Scope

~100 lines new code + ~50 lines tests + HTML template

---

## Feature 2: `rootline init --template <repo>`

### What It Does

Downloads `.stem` files from a remote GitHub repository and installs them in the target directory.

### Interface

```bash
rootline init --template pablontiv/epic-tracking          # shorthand → github.com
rootline init --template github.com/org/repo              # full URL
rootline init --template pablontiv/epic-tracking@v2       # specific tag/branch
rootline init --template pablontiv/epic-tracking --dry-run  # preview without writing
rootline init --template pablontiv/epic-tracking --force    # overwrite existing .stem
```

### Behavior

- `--template <ref>` flag (StringVar) on `init` command
- When `--template` is set: skips inference entirely, downloads and copies `.stem` files
- Mutually exclusive with inference mode (if `--template` set, no scan/analyze)
- Existing `--force` and `--dry-run` flags apply

### Fetch Protocol

1. Parse ref: `owner/repo[@tag]` or `github.com/owner/repo[@tag]`
2. Verify `git` is in PATH (error if not: `"git is required for --template"`)
3. `git clone --depth 1 [--branch <tag>] https://github.com/<owner>/<repo>.git <tmpdir>`
   - If no tag specified: default branch (no `--branch` flag)
4. Find all `.stem` files in the cloned repo root (recursive)
5. Validate each `.stem` is valid YAML via `yaml.Unmarshal`
6. Copy `.stem` files to target directory, preserving relative paths
7. Clean up temp clone directory

### Error Cases

| Condition | Error message |
|-----------|---------------|
| `git` not in PATH | `git is required for --template` |
| Repo doesn't exist | Git clone error propagated |
| No `.stem` files found | `no .stem files found in <repo>` |
| Invalid YAML in `.stem` | `invalid .stem file: <name>: <parse error>` |
| Target `.stem` exists (no `--force`) | `<name> already exists (use --force to overwrite)` |
| Network offline | Git clone error propagated |

### Implementation

**New files:**
- `internal/templates/fetch.go` — `FetchTemplate(ref, dest string, force, dryRun bool) ([]string, error)`
- `internal/templates/fetch_test.go` — tests with temp git repos as fixtures

**Modified files:**
- `cmd/rootline/init.go` — add `--template` flag, branch to fetch logic when set

### No Cache

First version always clones fresh. Cache is a future optimization (content-addressable by commit hash).

### Estimated Scope

~120 lines new code + ~80 lines tests

---

## Feature 3: Structural Inference (Category 4)

### What It Does

New detector that analyzes directory structure and infers `structural.subdirs` rules for `.stem` files.

### Inference Heuristics

| Rule | Signal | Threshold |
|------|--------|-----------|
| `require_index` | % of subdirs with README.md | ≥90% presence → infer |
| `min_children` | Minimum observed subdirectory count per parent | `min(counts)` across parents at same level |
| `max_children` | Maximum observed subdirectory count per parent | `max(counts)` across parents at same level |

### Interface

Results appear in `rootline analyze` report as new inference type:

```json
{
  "category": "structural",
  "type": "add_structural_rule",
  "rule": "require_index",
  "value": "README.md",
  "evidence": {
    "directories_with_index": 58,
    "directories_total": 60,
    "presence": 0.97
  },
  "requires_agent": false
}
```

### Implementation

**Detection algorithm:**
1. Walk `scanRoot` with `os.ReadDir()`
2. For each directory with subdirectories:
   - Count child directories (ignore files)
   - Check for `README.md` (or most-frequent index filename)
   - Accumulate stats per hierarchy level (if hierarchy detected)
3. Compute aggregates:
   - Index presence ratio → `require_index` if ≥0.90
   - Min/max child counts → `min_children`/`max_children`
4. Return `[]ReportInference` compatible with AnalyzeReport

**Integration points:**
- `cmd/rootline/analyze.go` — invoke as detector #13 alongside existing 12
- `cmd/rootline/init.go` — generate `structural:` section in `.stem` output
- `internal/infer/apply.go` — extend to write `structural:` section to `.stem`

**New files:**
- `internal/infer/structural.go` — `DetectStructural(scanRoot string, records []*extract.Record) []ReportInference`
- `internal/infer/structural_test.go` — tests with temporary directory trees

**Modified files:**
- `cmd/rootline/analyze.go` — integrate detector
- `cmd/rootline/init.go` — generate structural section
- `internal/infer/apply.go` — apply structural rules to `.stem`

### Tests

- Directory with 100% README.md → infers `require_index`
- Directory with 80% README.md → does NOT infer (below 90%)
- Empty directory → no inference
- Consistent child counts → infers min/max_children
- Varied child counts → uses actual min/max observed

### Estimated Scope

~120 lines new code + ~100 lines tests

---

## Verification Plan

### Per-Feature

1. **Graph --open**: `go test ./cmd/rootline/ -run TestGraph` + `go test ./internal/graph/ -run TestHTML` + manual `rootline graph docs/epics/ --open`
2. **Init --template**: `go test ./internal/templates/` + `go test ./cmd/rootline/ -run TestInit` + manual `rootline init --template pablontiv/epic-tracking --dry-run`
3. **Structural inference**: `go test ./internal/infer/ -run TestStructural` + `rootline analyze docs/epics/ --output json | jq '.categories[] | select(.id=="structural")'`

### Integration

```bash
just test                    # all tests pass
just check                   # build clean (golangci-lint in CI)
rootline validate --all docs/epics/  # existing validation unaffected
```

### Coverage Gate

```bash
go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1
# Must stay ≥85%
```
