---
tipo: historia
cliente: Platform Owner
---
# S002: Code & Tooling Alignment

**Feature**: [F12 Estado System Standardization](../README.md)
**Capacidad**: Codigo Go, tests, y skills del roadmap alineados con nuevos valores de estado en ingles

## Antes / Despues

**Antes**: `infer.go` hardcodea `"Completado"` en valueMapping y InferEstado. `links.go` lee solo `Frontmatter["estado"]` (ignora valores derivados) y doc comments referencian valores en espanol. Roadmap skill filtra por valores viejos mixtos. 22 test files usan `"Completado"`/`"Bloqueada"` en ~209 ocurrencias.

**Despues**: Codigo Go referencia valores en ingles (`Completed`, `Blocked`). `links.go` usa `EffectiveField()` para leer estado (respeta valores derivados). Roadmap skill loop filtra por `['Specified', 'In Progress']`. Tests consistentes con nuevo modelo.

## Criterios de Aceptacion (semanticos)

- [ ] `go test ./... -race` pasa sin errores
- [ ] `grep -r "Completado" internal/ --include="*.go" | wc -l` retorna 0
- [ ] Roadmap skill loop query usa `estado in ['Specified', 'In Progress']`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-update-production-code.md) | Update infer.go and links.go production code |
| [T002](T002-update-test-files.md) | Update test files and testdata stems |
| [T003](T003-update-roadmap-skill.md) | Update roadmap skill estado references |

## Fuente de verdad

- `internal/proposal/infer.go` (valueMapping, InferEstado)
- `internal/derive/links.go` (InjectLinkedFields, doc comments)
- `.claude/skills/roadmap/SKILL.md` (loop query, dependency check)
- `.claude/skills/roadmap/task-guide.md` (estado table)
