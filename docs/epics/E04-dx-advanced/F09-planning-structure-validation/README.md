---
estado: Pending
tipo: feature
---
# F09: Planning Structure Validation

**Epic**: [E04](../README.md)
**Objetivo**: Rootline valida reglas estructurales de directorio (index requerido, conteo de hijos) y dependencias entre tasks via wiki-links (`[[blocks:X]]`) detectadas nativamente por rootline graph
**Beneficio**: Las inconsistencias estructurales (epic sin README, epic con 1 sola feature, dependencias rotas entre tasks, ciclos) se detectan automaticamente en lugar de descubrirse manualmente
**Milestone**: `rootline validate --all docs/epics/` reporta directorios sin README, epics con < 2 features. `rootline graph --check docs/epics/` detecta broken links y ciclos en dependencias declaradas como wiki-links.

## Scope

**In**: Bloque `structural:` en .stem (require_index, min_children), wiki-links `[[blocks:X]]` para dependencias entre tasks, seccion `links:` en .stem para validar tipos de link, integracion con graph --check, actualizacion de skill guides
**Out**: Propagacion de estado por dependencias (task bloqueado no puede estar In Progress), unique_prefix cross-epic (numeracion global), auto-fix de problemas estructurales

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-structural-directory-rules/) | Structural Directory Rules | .stem define reglas de directorio; validate --all las verifica |
| [S002](S002-dependency-frontmatter/) | Dependency Wiki-Links | Tasks declaran deps via wiki-links; rootline graph valida targets y ciclos |
| [S003](S003-skill-loop-integration/) | Skill and Loop Integration | Skills generan wiki-links de deps; loop usa graph para orden |

## Dependencias

- F05-dependency-graph (graph builder y cycle detection existentes)

## Fuente de verdad

- `internal/rules/rules.go` — StemFile struct, LinkSchema
- `internal/rules/validate.go` — Validate function (single-record)
- `cmd/rootline/validate.go` — validate --all pipeline
- `internal/graph/graph.go` — Graph builder y DetectCycles
- `internal/extract/links.go` — ParseLinks (wiki-link parser)
- `.claude/skills/roadmap/task-guide.md` — template de tasks
- `.claude/skills/roadmap/SKILL.md` — /roadmap loop procedure
