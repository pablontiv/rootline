# rootline-tools

Claude Code plugin for [rootline](https://github.com/pablontiv/rootline) — a file-based database and constraint engine for structured documentation.

## Skills

| Skill | Description |
|-------|-------------|
| `/validate` | Validate documents against `.stem` schemas |
| `/describe` | Show the effective schema for a directory |
| `/new-doc` | Scaffold a new document with correct frontmatter and auto-numbering |

## Prerequisites

- `rootline` CLI must be installed and available in PATH
- A project with `.stem` schema files

## Installation

```bash
claude plugin add /path/to/rootline/claude-plugin
```

Or from the repository:

```bash
claude plugin add https://github.com/pablontiv/rootline --path claude-plugin
```

## Usage

Once installed, the skills are available as slash commands in Claude Code:

```
/validate docs/epics/          # Validate all files in a directory
/validate path/to/file.md      # Validate a single file
/describe docs/epics/E01-foo/  # Show effective schema for a directory
/new-doc docs/epics/E01-foo/   # Create a new document with auto-numbering
```
