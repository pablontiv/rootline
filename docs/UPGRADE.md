# Upgrade Guide: Root Marker Requirement

## What Changed

Schema discovery no longer uses the `.git` directory as a project boundary. A project now declares its own boundary with a `root: true` marker in a `.stem` file. Discovery walks up collecting `.stem` files and stops at the first one that declares `root: true`, or at the filesystem root.

This makes schema resolution:

- **Stable** — it no longer depends on the process working directory
- **Bounded** — the walk stops at the declared boundary, so a stray ancestor `.stem` (for example one in your home directory) can no longer silently govern the project
- **Git-independent** — rootline no longer requires a `.git` repository

## What Breaks

An existing project that has `.stem` files but no `root: true` marker. Governed commands — `validate`, `fix`, `query`, `tree`, `graph`, `describe`, `explain`, `set`, `stats` — now fail instead of silently walking past the project. Without a terminal they exit non-zero with the exact fix:

```
Error: Schema discovery reached the filesystem root without finding a declared boundary.

No .stem in this project declares where the project starts.

Fix: add this line to <path>/.stem

  root: true

Discovery then stops there and never reads .stem files above it.
Run rootline in a terminal to be prompted and have this applied for you.
```

This is intentional. The previous behavior could validate zero records and still exit successfully — a false-green CI run.

## How to Migrate

Pick either option. Both are one-time.

### Option 1 — Add the marker manually

Add one line to your project's top-level `.stem` and leave everything else untouched:

```yaml
version: 2
root: true
scope:
  # ... the rest of your existing .stem, unchanged
```

Then run your command again. This is the recommended path for an existing project: it changes nothing but the boundary.

### Option 2 — Let rootline add it interactively

Run any governed command from a terminal. When no boundary is declared, rootline prints:

```
No .stem in this project declares a governance boundary.
To establish one, add 'root: true' to <path>/.stem.

Apply this change now? (y/n)
```

Type `y` and rootline writes `root: true` into that `.stem`, preserving the rest of the file. Type `n` and nothing is changed.

Without a terminal (CI, hooks, scripts) rootline does not prompt — it fails with the error above so a pipeline never hangs waiting for input.

> **Do not use `rootline init --force` to migrate.** `init` regenerates a `.stem` by re-inferring the schema from your documents; on an existing project it overwrites your hand-authored schema. Use it only for a brand-new project.

### For new projects

`rootline init` writes `root: true` into the `.stem` it generates, so a new project declares its boundary from the start and never sees this error.

## Why This Change Matters

### The false-green bug

Outside any schema boundary, a governed command used to succeed while validating nothing:

```
# Old behavior: exit 0, zero records validated (false green)
# New behavior: exit 1, with the fix
```

CI could pass despite governing nothing.

### Working-directory independence

Resolution used to depend on where the `.git` directory sat relative to your shell. It now depends only on the target path and the `.stem` files on disk, so the same record resolves identically no matter where you run the command from.

### Boundary safety

Without a boundary, the walk could climb out of the project:

```
/home/user/.stem                   (personal stem)
/home/user/project/.stem           (project root, no marker)
/home/user/project/docs/.stem
/home/user/project/docs/E01.md
```

The old walk would collect `/home/user/.stem` and apply a personal schema to the project. With a marker on `/home/user/project/.stem`, the walk stops there and never reads the ancestor.

## Rollback

The behavior lives in the binary, not in your `.stem` files. Removing the `root: true` line does **not** restore the old `.git`-boundary behavior — it just leaves the project undeclared again, and governed commands will fail. The only way back to the previous behavior is to install an older rootline binary.

Leaving `root: true` in place is safe with older binaries: they do not recognize the field and ignore it.

## FAQ

### Can I have multiple root markers?

Yes. Nested projects can each declare their own boundary:

```
/repo/.stem            (root: true)   main project boundary
/repo/subproject/.stem (root: true)   sub-project boundary
```

Resolving a record under `subproject/` stops at the closest marker and never reads `/repo/.stem`. Inheritance is narrowed at that point, on purpose, so a sub-project governs itself. `rootline validate --all` reports this as an info-level `nested-root-marker` note, not an error.

### What if I have no `.stem` files at all?

`schema propose` and `analyze` work without any `.stem` — they derive one from your documents. The governed commands (`validate`, `query`, `describe`, and the rest) fail, which is correct: there is no schema to work against.

### What about auto-update?

rootline applies staged updates on the next run without prompting, so you may receive this change without choosing to upgrade. The next governed command will either prompt you (with a terminal) or fail with the exact fix (without one). That is why this note leads with the one-line fix and the error message carries the remedy itself.

## More Information

- `CHANGELOG.md` — the release notes for this change
- The repository's own `.stem` files — working examples, including the root marker at the repository root
