---
tipo: historia
cliente: Plugin Consumer
---
# S001: Marketplace Repository Scaffold

**Feature**: [F09 Agent Marketplace](../README.md)
**Capacidad**: Repo agent-marketplace existe en GitHub con dual format (skills.sh + Claude plugin), 4 skills de rootline, y documentación de instalación

## Antes / Despues

**Antes**: Skills viven solo dentro del repo rootline. Otros proyectos no pueden instalarlos. Compartir requiere copia manual de archivos. Incompatible con otros agentes (Cursor, Copilot).

**Despues**: `agent-marketplace` existe en GitHub. `npx skills add pablontiv/agent-marketplace` instala skills en cualquier agente AI. `claude plugin add pablontiv/agent-marketplace` los instala como Claude Code plugin. Skills son idénticos a los del repo fuente.

## Criterios de Aceptacion (semanticos)

- [ ] Repo agent-marketplace creado con estructura skills.sh + .claude-plugin/
- [ ] marketplace.json válido con entrada para rootline-core plugin
- [ ] 4 skills copiados en formato SKILL.md estándar (name + description frontmatter)
- [ ] Skills funcionales via `npx skills add` y `claude plugin add`
- [ ] README con instrucciones para ambos métodos de instalación

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-create-marketplace-repo.md) | Crear repo agent-marketplace con estructura dual |
| [T002](T002-adapt-skills-standalone.md) | Adaptar skills para distribución standalone en formato skills.sh |
| [T003](T003-marketplace-readme.md) | Escribir README con instrucciones de instalación dual |

## Fuente de verdad

- `.claude/skills/` (skills a copiar)
- `anthropics/skills` (modelo de estructura dual)
- skills.sh spec (formato SKILL.md estándar)
