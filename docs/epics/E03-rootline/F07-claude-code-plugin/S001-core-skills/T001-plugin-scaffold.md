---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Crear plugin scaffold con plugin.json y estructura

**Story**: [S001 Core Skills](README.md)

## Contexto

Un Claude Code plugin necesita un directorio con plugin.json manifest y archivos de skills. El plugin wrappea rootline CLI — asume que `rootline` esta en PATH. La estructura sigue el patron de Claude Code plugins: plugin.json declara metadata y lista de skills, cada skill es un SKILL.md con instrucciones.

## Alcance

**In**:
1. Crear directorio `claude-plugin/` en raiz del repo
2. `plugin.json` con name, version, description, skills list
3. Directorios para cada skill: `skills/validate/`, `skills/describe/`, `skills/new-doc/`
4. README del plugin con instrucciones de instalacion

**Out**: Publicacion en registry, hooks, agents, MCP server integration

## Estado inicial esperado

- `.claude/skills/roadmap/` existe como referencia de patron
- rootline CLI funcional

## Criterios de Aceptacion

- `claude-plugin/plugin.json` existe y es JSON valido
- Plugin declara 3 skills en su manifest
- Estructura de directorios: `claude-plugin/skills/{validate,describe,new-doc}/`
- README incluye instrucciones de instalacion (`claude plugin add`)

## Fuente de verdad

- `.claude/skills/roadmap/` (patron existente)
- Claude Code plugin documentation
