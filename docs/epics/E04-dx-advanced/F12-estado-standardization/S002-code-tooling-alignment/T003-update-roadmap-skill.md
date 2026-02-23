---
estado: Pending
tipo: documentation
ejecutable_en: 1 sesion
---
# T003: Update roadmap skill estado references

**Story**: [S002 Code & Tooling Alignment](README.md)

[[blocks:T002-update-test-files]]

## Contexto

El skill `/roadmap` en `.claude/skills/roadmap/` referencia valores de estado viejos en espanol en multiples archivos. El loop query filtra por una lista inflada de 6 valores (`Pending, Especificado, Specified, Bloqueada, Diferida, In Progress`). La dependency check busca `estado: Completado`. La tabla de estados en `task-guide.md` usa valores en espanol (Especificado, Completado).

## Alcance

**In**:
1. `SKILL.md` lineas 204, 237: Cambiar query filter a `estado in ['Specified', 'In Progress']`
2. `SKILL.md` linea 257: Cambiar `estado: Completado` → `estado: Completed`
3. `SKILL.md` linea 258: Actualizar skip message a usar `Completed`
4. `task-guide.md` lineas 267-269: Actualizar tabla de estados con nuevos valores en ingles + Blocked + On Hold
5. `task-guide.md` linea 98: Cambiar template frontmatter `estado: Pending` → mantener (Pending no cambia)
6. `framework-reference.md`: Buscar y actualizar cualquier referencia a valores viejos

**Out**: Cambios a codigo Go, cambios a .stem, migracion de frontmatter

## Estado inicial esperado

- S001 y S002/T001-T002 completados
- Skill files con referencias a valores viejos

## Criterios de Aceptacion

- `grep "Completado" .claude/skills/roadmap/*.md | wc -l` retorna 0
- `grep "Bloqueada" .claude/skills/roadmap/*.md | wc -l` retorna 0
- `grep "Especificado" .claude/skills/roadmap/*.md | wc -l` retorna 0
- `grep "Diferida" .claude/skills/roadmap/*.md | wc -l` retorna 0
- SKILL.md loop query contiene `estado in ['Specified', 'In Progress']`
- task-guide.md tiene tabla con 6 estados en ingles

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md`
- `.claude/skills/roadmap/task-guide.md`
- `.claude/skills/roadmap/framework-reference.md`
- `.claude/skills/roadmap/epic-guide.md`
- `.claude/skills/roadmap/story-guide.md`
