---
name: new-doc
description: |
  Scaffold a new document with correct frontmatter based on .stem schema.
  Uses auto-numbering to generate the next ID automatically.
  This skill should be used when the user says "new doc", "crear documento",
  "scaffold", "create a new file", "add a document", "nuevo archivo",
  "add a task", "new record", "nueva tarea", "rootline new",
  or wants to create a structured document in a rootline-managed directory.
argument-hint: "[directory-or-filepath]"
allowed-tools: Bash, AskUserQuestion
---

# /new-doc — Document Scaffolding

Create a new document with frontmatter pre-populated from the effective `.stem` schema, with automatic ID sequencing.

## Procedure

### 1. Determine target

- If `$ARGUMENTS` is a **file path** (ends in `.md`) → use as target path directly
- If `$ARGUMENTS` is a **directory** → need to build filename (go to step 2)
- If `$ARGUMENTS` is **empty** → run `rootline tree . --output table` to show available directories, then use `AskUserQuestion` to let the user pick a target directory

### 2. Auto-numbering (if target is a directory)

Get the next sequential ID for the directory:

```bash
rootline describe <directory> --field schema.id.next
```

This returns the next ID (e.g., `"T005"`, `"E03"`, `"S002"`).

If the user provides a name, build the filename as: `<ID>-<name>.md`
If no name provided, use `AskUserQuestion` to ask what the document should be called.

### 3. Preview with dry-run

Show what will be created before writing:

```bash
rootline new <filepath> --dry-run
```

This prints the generated markdown with frontmatter to stdout without creating the file.

### 4. Create the document

```bash
rootline new <filepath>
```

The command:
- Scaffolds frontmatter from the effective `.stem` schema
- Sets enum fields to their first allowed value
- Uses defaults where defined
- Marks required fields that need values
- Generates a title from the filename

If `rootline new` fails (e.g., no `.stem` schema found, file already exists), report the error. Use `--force` only if the user explicitly asks to overwrite.

### 5. Validate the new document

Run validation to confirm the scaffold is correct:

```bash
rootline validate <filepath> --output json
```

If validation passes, confirm success. If it fails, report the errors — some fields may need manual values (required fields without defaults).

### 6. Suggest next steps

After creating the document, suggest:
- Edit the file to fill in required fields that are empty
- Run `/validate` to check after editing
- The document's frontmatter structure and what each field means (use `/describe` on the parent directory)
