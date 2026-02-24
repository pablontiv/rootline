---
estado: Specified
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Crear repo agent-marketplace con estructura dual

**Story**: [S001 Marketplace Repository Scaffold](README.md)

## Contexto

El estándar skills.sh define skills como carpetas con SKILL.md (frontmatter: name + description). Claude Code plugin marketplaces requieren `.claude-plugin/marketplace.json`. Ambos formatos coexisten en el mismo repo — modelo de referencia: `anthropics/skills`.

## Alcance

**In**:
1. Crear repo `agent-marketplace` en GitHub (manual: `gh repo create pablontiv/agent-marketplace --public`)
2. `skills/rootline-roadmap/SKILL.md` + archivos soporte (framework-reference.md, guides)
3. `skills/rootline-validate/SKILL.md`
4. `skills/rootline-describe/SKILL.md`
5. `skills/rootline-new-doc/SKILL.md`
6. `.claude-plugin/marketplace.json` con plugin entry para rootline-core apuntando a `./skills/rootline-*`

**Out**: Automatización de sync (S002), binarios (S003), hooks, agents

## Estado inicial esperado

- `.claude/skills/` contiene 4 skills funcionales (roadmap, validate, describe, new-doc)
- Cuenta GitHub con permisos para crear repos

## Criterios de Aceptacion

- Repo `agent-marketplace` existe en GitHub
- `.claude-plugin/marketplace.json` es JSON válido con plugin entry para rootline-core
- `skills/rootline-*/SKILL.md` existe para cada uno de los 4 skills con frontmatter `name` + `description`
- Archivos de soporte del roadmap skill están presentes
- `npx skills add pablontiv/agent-marketplace` lista skills disponibles
- `claude plugin add pablontiv/agent-marketplace` lista rootline-core como disponible

## Fuente de verdad

- `.claude/skills/` (fuente de skills)
- `anthropics/skills` repo (modelo de marketplace.json y estructura)
- skills.sh spec (formato SKILL.md)
