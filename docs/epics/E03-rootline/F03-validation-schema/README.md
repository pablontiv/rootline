# F03: Validation and Schema

**Epic**: [E03](../README.md)
**Objetivo**: Rootline valida documentos contra schemas .stem y muestra schemas efectivos
**Beneficio**: Elimina 5 formatos de governance hardcodeados; hooks consultan `describe` en vez de hardcodear valores
**Milestone**: `rootline validate` y `rootline describe` producen JSON output correcto

## Scope

**In**: Rules engine (4 reglas), validate command, describe command con --field extraction
**Out**: Parametric rules (format, max_length), rule disabling mechanism, query engine

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Validation Engine](S001-validation-engine/) | Motor de reglas deterministico valida frontmatter contra schema |
| S002 | [Validate Command](S002-validate-command/) | `rootline validate` CLI command |
| S003 | [Describe Command](S003-describe-command/) | `rootline describe` muestra schema efectivo con source tracing |

## Dependencias

- F02 completado (core engine: .stem parsing + merge + extraction + scanner)

## Fuente de verdad

- `src/rootline/docs/research/I5-describe-contract.md` — describe contract y validation rules
- `src/rootline/docs/intent/v0-rootline.md` — seccion 3 (Commands)
