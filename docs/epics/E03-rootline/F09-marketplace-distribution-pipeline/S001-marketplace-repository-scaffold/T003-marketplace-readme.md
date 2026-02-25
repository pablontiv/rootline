---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T003: Escribir README con instrucciones de instalación dual

**Story**: [S001 Marketplace Repository Scaffold](README.md)

## Contexto

El repo agent-marketplace necesita un README orientado al consumidor con dos métodos de instalación: skills.sh (cualquier agente) y Claude Code plugin. Es la primera impresión para usuarios que descubren los skills.

## Alcance

**In**:
1. README.md en raíz de agent-marketplace
2. Sección de instalación con ambos métodos:
   - `npx skills add pablontiv/agent-marketplace` (cualquier agente)
   - `claude plugin add pablontiv/agent-marketplace` (Claude Code)
3. Tabla/catálogo de skills disponibles con descripción y agentes compatibles
4. Sección de prerequisitos (rootline en PATH, link a releases)
5. Link al repo principal de rootline

**Out**: Documentación de desarrollo, contributing guide, changelog

## Estado inicial esperado

- T001 completado: repo agent-marketplace existe con estructura
- T002 completado: skills adaptados para standalone

## Criterios de Aceptacion

- README.md existe en raíz del repo
- Incluye ambos comandos de instalación (skills.sh y Claude plugin)
- Tabla con los 4 skills: nombre, descripción, agentes compatibles
- Sección de prerequisitos con link a rootline releases
- README renderiza correctamente en GitHub

## Fuente de verdad

- `anthropics/skills` README (modelo de referencia)
- rootline GitHub releases page
