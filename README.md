# Rootline

[![CI](https://github.com/pablontiv/rootline/actions/workflows/ci.yml/badge.svg)](https://github.com/pablontiv/rootline/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue)](LICENSE)

**Schemas, queries and a link graph for the Markdown your team — and your AI agents — already write.**

Rootline is a single-binary CLI that turns a directory of Markdown files into a validated, queryable dataset. You declare what each folder's documents must look like in a small `.stem` file; Rootline validates every file against it, answers questions like *"which tasks are still pending in this epic?"* without grepping, keeps roll-up counts in index files current, and checks that the links between documents actually resolve. Nothing is rendered, nothing is uploaded, and no database is introduced: the files stay plain Markdown in Git.

```bash
rootline validate --all docs/                      # every document conforms to its schema?
rootline query docs/ --where 'status == "pending"' # records as JSON, no grep
rootline graph docs/ --check                       # broken links and cycles → exit code
```

---

## The problem

Teams keep more and more of their operating knowledge in Markdown: PRDs, ADRs, roadmaps, runbooks, task files, Obsidian vaults, the `docs/` folder an AI coding agent reads before every change. It scales badly:

- **Frontmatter drifts.** `status: Pending` in one file, `estado: pending` in the next, a required `owner` missing in half of them. Nobody notices until something downstream breaks.
- **Nothing is queryable.** "What is blocked right now?" means opening files or writing a one-off script.
- **Roll-ups go stale.** The `README.md` that says *12 of 20 tasks done* was true two weeks ago.
- **Links rot silently.** `[[T014-migrate-auth]]` was renamed; three documents still point at it.
- **Agents guess at structure.** Coding agents write Markdown fast and inconsistently unless something tells them — in a machine-readable way — what a valid record is.

Linters check formatting. Static-site frameworks validate frontmatter only inside their build. Query tools read frontmatter but enforce nothing. Rootline does all four jobs — **validate, query, derive, link-check** — from one CLI, with inherited schemas and stable JSON output that humans, CI and agents consume the same way.

## Who it is for

- **Docs-as-code teams** keeping PRDs, ADRs, roadmaps or runbooks in a Git repository and wanting CI to reject malformed records.
- **Engineering teams using AI coding agents** (Claude Code, Codex, Gemini CLI, …) that need a contract for the Markdown those agents read and write.
- **Obsidian / Markdown vault owners** who want validation and queries from the terminal and in CI, not only inside an editor plugin.

If your Markdown is prose with no metadata, Rootline is not the tool. If it has frontmatter you rely on, it is.

---

## Quick start (5 minutes)

### 1. Install

```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/pablontiv/rootline/master/install.sh | bash

# Windows (PowerShell)
irm https://raw.githubusercontent.com/pablontiv/rootline/master/install.ps1 | iex

# From source (Go 1.26+)
go install github.com/pablontiv/rootline/cmd/rootline@latest
```

Release builds auto-update within their compatibility boundary; see [docs/auto-update.md](docs/auto-update.md).

### 2. Infer a schema from what you already have

```bash
rootline init docs/
```

`init` reads the frontmatter in `docs/`, infers field types, enums and required fields, and writes a `docs/.stem` with `root: true` (the boundary where schema discovery stops). Preview with `--dry-run`, or start from someone else's schema with `--template owner/repo`.

A `.stem` is small and readable:

```yaml
version: 2

schema:
  title:  { type: string, required: true }
  status:
    type: enum
    values: [pending, in_progress, done, blocked]
    default: pending
    required: true
  owner:  { type: string }

derive:
  slug: 'slugify(title)'

aggregate:
  done_count: 'len(filter(descendants, .status == "done"))'

links:
  blocked_by:
    target: '.*/T[0-9]{3}-.*\.md$'
```

### 3. Validate

```bash
rootline validate --all docs/
```

A typo in an enum value, a missing required field or a link to a file that does not exist is reported with the file, the rule, the `.stem` that declared it and — when there is an obvious fix — a suggestion:

```json
{
  "rule": "enum",
  "field": "status",
  "message": "value pendig is not in allowed values: [pending, in_progress, done, blocked] (did you mean \"pending\"?)",
  "source": "docs/.stem",
  "severity": "error",
  "suggestion": "pending"
}
```

Exit code is non-zero on any error, so the same command gates a pre-commit hook (`rootline hooks install`) or a CI job.

### 4. Query

```bash
rootline query docs/ --where 'status == "blocked"' --select path,title,owner
rootline query docs/ --where 'status != "done"' --count
rootline query docs/ --has-outbound 'status == "blocked"'   # records that depend on something blocked
```

Filters use [expr-lang](https://expr-lang.org/) syntax over frontmatter, derived and aggregated fields. Output is JSON by default; `--output table`, `jsonl` and `csv` are available where they make sense.

### 5. Fix, scaffold, explain

```bash
rootline fix --all docs/ --dry-run      # proposed repairs (enum typos, missing defaults) — nothing written
rootline new docs/T021-rotate-keys.md   # a new record with frontmatter pre-filled from the effective schema
rootline explain docs/README.md         # where each field value came from: frontmatter, derive, aggregate, which .stem
rootline graph docs/ --format mermaid -o table   # the dependency graph, ready to paste into GitHub
```

---

## How it works

**Directories are tables, files are records, `.stem` files are the schema.** When Rootline resolves the schema for a file it walks *up* the directory tree collecting `.stem` files until it reaches one marked `root: true`, then merges them top-down. A child directory can add fields or tighten constraints; it can never remove a guarantee a parent made. That one rule is what makes a large tree governable: whatever `docs/.stem` promises holds for everything beneath it.

```
docs/.stem              root: true — title, status required for every record
docs/roadmap/.stem      adds: id as sequence T###, owner required
docs/roadmap/T014.md    validated against the merge of both
```

The pipeline is **extract → resolve schema → validate → derive → aggregate → query**, and every command emits a versioned JSON envelope (`"version": 2, "kind": "rootline/validate-batch"`) so output shapes are a contract, not an accident.

Rootline **never leaves a file its own schema refuses.** Repairs go through atomic writes, preserve YAML key order and comments, and each written file is re-validated and restored byte-for-byte if the result fails. Git is optional: `.git` is only consulted by the explicit `--staged`, `hooks` and `migrate` flows, never for schema discovery.

---

## How it compares

| | Rootline | Markdown linters (`markdownlint`, `mdschema`, `remark-lint-frontmatter-schema`) | Frontmatter query tools (`krafna`, `markdowndb`, Obsidian Dataview) | Framework collections (Astro, Contentlayer) |
|---|---|---|---|---|
| Validate frontmatter against a schema | ✅ | ✅ | ❌ | ✅ (only inside the build) |
| Schema inherited through the directory tree | ✅ | ❌ | ❌ | ❌ |
| Infer a schema from existing files | ✅ | partial | ❌ | ❌ |
| Query records with filters | ✅ | ❌ | ✅ | ✅ (in app code) |
| Derived and aggregated fields | ✅ | ❌ | partial | ❌ |
| Link graph, broken-link and cycle checks | ✅ | partial | partial | ❌ |
| Propose and apply safe repairs | ✅ | ❌ | ❌ | ❌ |
| Single static binary, no runtime | ✅ Go | mostly Node (`mdschema`: Go) | varies | Node |

Use a linter for prose style; use Rootline when the metadata in your Markdown is data you depend on.

---

## In CI and in agents

### GitHub Action

This repository ships a composite action. Add it to any workflow:

```yaml
- uses: pablontiv/rootline@master
  with:
    path: docs/
    fail-on: error     # or: warning | never
```

It installs the latest release (or uses a `rootline` already on `PATH`), runs `validate --all` and exposes `valid` and `error_count` as outputs.

### Pre-commit hook

```bash
rootline hooks install      # validates staged Markdown on every commit; respects core.hooksPath
```

### AI coding agents

Rootline is built to be the structured source of truth an agent calls instead of reading files: `query`, `validate`, `describe`, `explain`, `tree`, `stats` and `graph` all return stable JSON, and `--field` extracts exactly the value a tool call needs. `rootline skill install` publishes a managed agent skill into `~/.claude/skills` and `~/.agents/skills` through an explicit plan-and-approve flow — see [docs/skill.md](docs/skill.md).

The engine decides everything that follows from **form** (a field present in >80 % of records is inferred as required; a majority value becomes a default). Decisions that need **meaning** are not guessed: `analyze` marks them `requires_agent` and reports percentage evidence for a human or an agent to resolve.

---

## Command reference

| Area | Commands | Docs |
|---|---|---|
| Validate & govern | `validate`, `describe`, `explain` | [validate](docs/validate.md) · [describe](docs/describe.md) · [explain](docs/explain.md) |
| Query & traverse | `query`, `tree`, `stats`, `graph` | [query](docs/query.md) · [tree](docs/tree.md) · [stats](docs/stats.md) · [graph](docs/graph.md) |
| Build & maintain | `init`, `new`, `set`, `fix` | [init](docs/init.md) · [new](docs/new.md) · [set](docs/set.md) · [fix](docs/fix.md) |
| Analyze & evolve | `analyze`, `schema propose/apply`, `repair apply`, `migrate` | [analyze](docs/analyze.md) · [migrate](docs/migrate.md) |
| Infrastructure | `hooks`, `skill`, `completion` | [skill](docs/skill.md) · [auto-update](docs/auto-update.md) |

Commands with a JSON envelope accept `--output json|table` (plus `jsonl`/`csv` on `query --select`) and `--field <dot.path>` for extracting one value; commands that only write files or print text reject them. `query`, `stats`, `tree`, `graph` and `validate --all` accept `--where 'expr'`. Full contracts: [docs/output.md](docs/output.md).

Deeper topics: [derivation & aggregation](docs/derivation.md) · [levels & `match:` scoping](docs/levels.md) · [extensibility](docs/extensibility.md) · [upgrading from v1 `.stem`](docs/UPGRADE.md).

---

## Development

Requires Go 1.26+ and Git (for hooks and CI). Field names in this repository's own governed data (`docs/roadmap/`) are in Spanish — `estado`, `tipo` — because that is how the roadmap was written; Rootline itself has no opinion on language.

```bash
just check      # gofmt + golangci-lint + go build
just test       # go test ./... -race
just validate   # rootline validate --all docs/roadmap/
```

Commits follow [Conventional Commits](https://www.conventionalcommits.org/) and are enforced by a commit-msg hook. See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md). Visual identity guidelines live in [docs/identity.md](docs/identity.md).

## License

[Apache License 2.0](LICENSE) — free for commercial and non-commercial use.
