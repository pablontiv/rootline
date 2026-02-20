# S002: Guidance UX

**Feature**: [F06 E05 Hardening](../README.md)
**Estado**: Completado
**Cliente**: Platform Owner
**Capacidad**: describe e init guian al usuario con mensajes claros cuando encuentran escenarios vacios o mixtos

## Antes / Despues

**Antes**: `rootline describe` en directorio sin .stem retorna schema vacio sin explicacion — el usuario no sabe que debe correr `rootline init`. `rootline init` en directorio con mezcla de archivos con y sin frontmatter (READMEs + Tasks) infiere schema sin advertir que el resultado puede ser suboptimo.

**Despues**: `rootline describe` detecta schema vacio y sugiere `rootline init` tanto en JSON (campo hints) como en table output. `rootline init` detecta ratio alto de archivos sin frontmatter y emite warning sugiriendo usar un subdirectorio mas acotado.

## Criterios de Aceptacion (semanticos)

- [x] `rootline describe` sin .stem muestra sugerencia de correr `init`
- [x] `rootline init` con >20% archivos sin frontmatter emite warning

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-describe-hints.md) | describe sugiere init cuando no hay .stem |
| [T002](T002-init-mixed-warning.md) | init advierte sobre contenido mixto |

## Fuente de verdad

- `cmd/rootline/describe.go` — describe command
- `internal/rules/describe.go` — DescribeResult struct
- `cmd/rootline/init.go` — init command
- `internal/infer/infer.go` — Analyze function
