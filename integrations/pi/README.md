# pi-rootline

Pi extension bundle for Rootline schema querying, validation, and analysis.

## Overview

This package provides read-only Pi tools that expose Rootline capabilities (schema validation, querying, analysis) through subprocess calls to the Rootline CLI. Tools execute with JSON output and respect Rootline's versioned contracts.

## Installation

Install locally via:
```bash
pi install -l ./integrations/pi
```

## Tools

Read-only tools are available immediately:

- `rootline-query` — Query records by frontmatter field filters
- `rootline-describe` — Show merged schema for a directory
- `rootline-validate` — Validate records against `.stem` schema
- `rootline-tree` — View directory structure with metadata
- `rootline-stats` — Show record counts and field statistics
- `rootline-graph` — Analyze wiki-link dependencies
- `rootline-explain` — Show field provenance and resolution chain
- `rootline-analyze` — Infer schema from existing documents

(Additional mutating tools are in development.)

## Architecture

See `/docs/roadmap/O02-design-pi-extension-architecture/T006-architecture-decision-record.md` for the design rationale.

## Package Structure

```
.
├── package.json          # Pi manifest with extensions, skills, prompts
├── extensions/           # TypeScript tool implementations (placeholder)
├── skills/               # Skill prompt files (placeholder)
└── prompts/              # Prompt templates (placeholder)
```
