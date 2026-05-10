I did **not** write `/home/shared/rootline/research/rootline-local-context.md` because the task also says “Do not edit files”; per higher-priority scouting rules, no-edit wins. Findings below are ready to paste.

# Rootline local context findings

## Key repo/package structure

- Module: `github.com/pablontiv/rootline`, Go `1.25.0` (`go.mod` lines 1-17).
- Single CLI binary entrypoint: `cmd/rootline/main.go` lines 1-5, root command/global flags in `cmd/rootline/root.go` lines 1-31.
- Packages from `go list ./...`:
  - `cmd/rootline`
  - `internal/{derive,e2e,extract,fix,fuzzy,graph,index,infer,migrate,proposal,query,rules,templates}`
- Important constraint: all reusable engine code is under `internal/`, so an external Pi package cannot import it directly. External integration should use CLI JSON unless the Pi extension lives inside this repo/module.

## CLI and JSON outputs

Commands confirmed via `./rootline --help`: `analyze`, `apply`, `completion`, `describe`, `explain`, `fix`, `graph`, `hooks`, `init`, `migrate`, `new`, `query`, `set`, `stats`, `trace`, `tree`, `validate`.

Global flags:
- `--output json|table`, default `json`
- `--field` dot-path extraction  
Source: `cmd/rootline/root.go` lines 25-27; implementation `cmd/rootline/validate.go` lines 351-375.

Stable JSON contracts:
- Query: `version`, `kind`, `meta`, `rows` (`internal/query/query.go` lines 37-77).
- Validate: single and batch results (`internal/rules/result.go` lines 1-85).
- Describe: effective schema (`internal/rules/describe.go` lines 10-65).
- Tree: `rootline/tree` (`cmd/rootline/tree.go` lines 32-44, 90-100).
- Stats: `rootline/stats` (`cmd/rootline/stats.go` lines 35-111).
- Graph: `rootline/graph` (`cmd/rootline/graph.go` lines 39-54, 141-166).
- Analyze report: `version:1`, `kind:"analyze"` (`internal/infer/report.go` lines 5-38).
- Migrate diff: `rootline/migrate-diff` (`internal/migrate/diff.go` lines 31-58).

Commands run:
- `./rootline query docs --limit 1 --output json` → `{"version":1,"kind":"rootline/query",...}`
- `./rootline stats docs --output json` → `{"version":1,"kind":"rootline/stats",...}`
- `./rootline describe docs --output json` → `{"version":1,"kind":"rootline/describe",...}`
- `./rootline validate --all docs --output json` → valid batch, 18 docs.

Caveat: `apply` output uses `internal/infer.ApplyResult` with no `version/kind` (`internal/infer/apply.go` lines 15-18). Also `apply --dry-run` is not fully safe for schema changes: schema writes are not guarded by dry-run (`cmd/rootline/apply.go` lines 50-64, 90-117; `internal/infer/apply.go` lines 27-107).

## Install/init/template/distribution

- Unix installer downloads GitHub release artifact, verifies checksum, installs binary (`install.sh` lines 1-114).
- Windows installer same pattern (`install.ps1` lines 1-107).
- GoReleaser builds `rootline` for linux/darwin/windows, amd64/arm64 (`.goreleaser.yml` lines 1-31).
- GitHub Action validates docs using downloaded/rootline binary (`action.yml` lines 1-220, 221-303).
- `rootline init --template owner/repo[@tag]` fetches `.stem` files from public GitHub repos:
  - CLI hook: `cmd/rootline/init.go` lines 24-52.
  - Git clone/copy implementation: `internal/templates/fetch.go` lines 1-172.
- Local hooks sync `.claude/skills/{roadmap,rootline}` to `~/.claude/skills`; comments mention Pi discovers there via `.pi/settings.json`, but no `.pi` directory was found (`.githooks/pre-push` lines 57-64; `.githooks/post-merge` lines 6-13).

## Docs/assets organization

- Main docs are flat under `docs/*.md`; schema at `docs/.stem` requires `estado` (`docs/.stem` lines 1-6).
- Existing Claude/Pi-adjacent assets:
  - `.claude/skills/rootline/SKILL.md` lines 1-84, includes command routing and JSON caveats.
  - `.claude/skills/roadmap/SKILL.md` lines 1-120.
  - `.claude/skills/roadmap/base.stem` lines 1-29.
- Visual/brand guidance exists at `docs/identity.md` lines 1-67.
- No `package.json`, `.pi/`, `.claude-plugin/`, or Pi manifest found locally.
- Potential stale constraint: CI and hooks reference `docs/epics/`, but `docs/epics` does not exist in this checkout. Command: `test -d docs/epics; echo docs_epics_exists=$?` returned `1`. See `.github/workflows/ci.yml` lines 16-21 and `.githooks/pre-push` lines 7-27.

## Can this Go repo host Pi package assets?

Yes, for static package/skill assets, but distribution plumbing would need changes.

Decision implications:
- If Pi package is just skills/prompts/config: put under `.claude/skills/...` or a new `.pi/...`; repo can host it. `.gitignore` does not exclude `.pi/` (`.gitignore` lines 1-39).
- Rootline no longer publishes local `.claude/skills/*` to an external marketplace.
- If it must ship with binary releases, update `.goreleaser.yml`, installers, and possibly `action.yml`.
- If it needs engine behavior, prefer CLI integration. External Go imports cannot use `internal/*`.

## Likely affected files

- `.claude/skills/rootline/SKILL.md`
- `.claude/skills/<new-pi-skill>/...` or new `.pi/...`
- `README.md`
- `.goreleaser.yml`, `install.sh`, `install.ps1` if release assets change

## Confidence

High on local structure, CLI contracts, install/distribution paths, and absence of Pi package files. Medium on Pi packaging decision because exact Pi package format/runtime expectations are not present in this repo.