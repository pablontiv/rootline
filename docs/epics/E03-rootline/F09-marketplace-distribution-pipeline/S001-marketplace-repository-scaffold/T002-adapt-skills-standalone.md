---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Adaptar skills para distribución standalone en formato skills.sh

**Story**: [S001 Marketplace Repository Scaffold](README.md)

## Contexto

Los skills actuales fueron escritos para uso local dentro del repo rootline. Al distribuirlos como skills standalone compatibles con skills.sh, necesitan: frontmatter estándar (solo `name` + `description`), paths relativos al skill directory, y una sección de prerequisitos que documente la dependencia en rootline CLI. El skill debe funcionar en cualquier agente AI (Claude, Cursor, Copilot).

## Alcance

**In**:
1. Verificar que cada SKILL.md tenga frontmatter `name` + `description` (campos requeridos por skills.sh)
2. Revisar cada SKILL.md por paths absolutos o relativos al repo rootline
3. Verificar que referencias a archivos de soporte usen paths relativos al skill directory
4. Agregar sección de prerequisitos en cada skill: rootline binary en PATH
5. Renombrar directorios con prefijo `rootline-` (rootline-roadmap, rootline-validate, etc.)

**Out**: Cambiar funcionalidad de los skills, agregar nuevos skills

## Estado inicial esperado

- T001 completado: repo agent-marketplace existe con skills copiados
- Skills copiados son idénticos a `.claude/skills/`

## Criterios de Aceptacion

- Cada SKILL.md tiene frontmatter con solo `name` y `description` (estándar skills.sh)
- Ningún SKILL.md contiene paths absolutos o referencias a `/opt/rootline/`
- Referencias a archivos de soporte usan paths relativos dentro del skill directory
- Cada skill documenta prerequisito: `rootline` en PATH
- Skills funcionan cuando se instalan via `npx skills add` en un proyecto externo

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md` (skill más complejo)
- `anthropics/skills/skills/pdf/SKILL.md` (modelo de referencia skills.sh)
- skills.sh spec (campos de frontmatter reconocidos)
