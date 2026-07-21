# Upgrade Guide: Root Marker Requirement (v0.11+)

## What Changed

Schema discovery in rootline v0.11+ no longer uses the `.git` directory as a project boundary. Instead, projects must explicitly declare their schema governance boundary using a `root: true` marker in a `.stem` file.

This change makes schema resolution:
- **Stable**: resolution no longer depends on the process working directory
- **Bounded**: schema discovery stops at the declared boundary, preventing accidental collection of `.stem` files from ancestor directories (e.g., home directory)
- **Git-independent**: projects can now use rootline without a `.git` repository

## What Breaks

### If your project has `.stem` files but no root marker

Running governed commands (`validate`, `fix`, `query`, `tree`, `graph`, `describe`, `explain`, `set`, `stats`) will now fail with this error:

```
Error: Schema discovery reached the filesystem root without finding a declared boundary.

No .stem in this project declares where the project starts.

Fix: add this line to <path>/.stem

  root: true

Discovery then stops there and never reads .stem files above it.
Run rootline in a terminal to be prompted and have this applied for you.
```

This is intentional — the previous behavior could silently validate zero records, producing a false-green CI run.

### New behavior

1. **With a terminal**: rootline prompts you to add the marker
   ```
   $ rootline validate docs/
   
   Error: No root marker found.
   
   Proposed root: /home/user/project
   Add root: true to /home/user/project/.stem? [y/n]
   ```
   Type `y` to apply the marker or `n` to exit. Once added, `root: true` prevents the walk from going higher.

2. **Without a terminal** (CI, scripts, background processes): rootline fails with the error above, including the exact line to add. This prevents hanging pipelines on an interactive prompt.

## How to Migrate

### For an existing project

Choose one option:

#### Option 1: Interactive migration (requires a terminal)

Run any governed command in your project directory:

```bash
rootline validate docs/
```

If your project lacks a root marker, rootline proposes the root directory and asks for confirmation. Type `y` to add the marker.

#### Option 2: Automatic migration

Run `rootline init --force` in your project directory:

```bash
rootline init --force docs/
```

This regenerates your `.stem` files with the root marker applied to the top-level `.stem`.

#### Option 3: Manual fix

Add one line to the top-level `.stem` file in your project:

```yaml
version: 2
root: true
schema:
  # ... rest of your schema
```

After adding the line, run your command again:

```bash
rootline validate docs/
```

### For new projects

`rootline init` (without `--force`) automatically emits `root: true` in the generated `.stem` files. No additional steps needed.

## Why This Change Matters

### The false-green bug

Before v0.11, a project outside any schema boundary would silently succeed with zero records validated:

```bash
# Project at /home/user/proj/ with .stem files but no root marker
$ rootline validate docs/

# Old behavior: exit 0, zero records validated (false green)
# New behavior: exit 1, clear error message
```

This made it possible for CI to pass despite validating nothing — a dangerous state.

### Working directory independence

Before v0.11, schema resolution depended on the `.git` directory, which could lead to inconsistent results:

```bash
$ cd docs/
$ rootline query .  # walks to .git, collects schema

$ cd /tmp/
$ rootline query /home/user/proj/docs/  # still walks to .git, same result

$ cd /home/  # or any other directory
$ rootline query /home/user/proj/docs/  # Git walk may find different .git, different schema!
```

With v0.11+, schema discovery depends only on the `.stem` files on disk, not on working directory or Git history.

### Boundary safety

Before v0.11, a project could accidentally inherit `.stem` files from outside:

```
/home/user/.stem                   (user's personal stem)
/home/user/project/.stem           (project root, no marker)
/home/user/project/docs/.stem
/home/user/project/docs/E01.md
```

Running `rootline validate` would collect both `.stem` files, applying the user's personal schema to the project. The project has no way to declare its boundary.

With v0.11+:

```
/home/user/.stem                   (ignored)
/home/user/project/.stem           (root: true added here)
/home/user/project/docs/.stem      (inherits from project/.stem only)
```

The root marker stops the walk before reading `/home/user/.stem`.

## Rollback

If you need to revert to pre-v0.11 behavior, remove the `root: true` line from your `.stem` file. However, note that the new behavior is safer and the line can remain in `.stem` files used with older rootline versions (they simply ignore the unknown field).

## FAQ

### Can I have multiple root markers?

Yes. If you have nested projects, each can declare its own boundary:

```
/repo/.stem (root: true)           # main project boundary
/repo/subproject/.stem (root: true)  # sub-project boundary
```

When resolving a record under `subproject/`, rootline stops at the closest marker and does not read `repo/.stem`. This is called "narrowed inheritance" and is intentional for sub-project independence.

### What if I have no `.stem` files at all?

Commands like `schema propose` and `analyze` work without any `.stem` files — they are bootstrap commands. Governed commands like `validate`, `query`, and `describe` will fail with an error, which is correct: there is no schema to validate against.

### What about auto-update?

rootline automatically applies staged updates without prompting. This means you may receive v0.11+ without choosing to upgrade. When you run a governed command next:
- **With a terminal**: you'll see the interactive prompt (non-breaking, recoverable)
- **Without a terminal** (CI, scripts): you'll see the error with the fix. Add `root: true` to your CI configuration or `.stem` files, commit, and push.

For this reason, the migration guide is prominent and the error message includes the exact fix.

## More Information

- See `docs/` for schema documentation
- See `.stem` file examples in the repository's own `.stem` files
- See `CHANGELOG.md` for full v0.11+ release notes
