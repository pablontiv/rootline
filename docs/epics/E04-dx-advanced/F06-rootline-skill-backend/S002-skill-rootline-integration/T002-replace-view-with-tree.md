---
estado: Completado
tipo: documentation
ejecutable_en: 1 sesion
---
# T002: Reemplazar /roadmap view con rootline tree

**Story**: [S002 Skill Rootline Integration](README.md)

## Contexto

La seccion `/roadmap view` del SKILL.md contiene ~40 lineas de instrucciones manuales que describen como escanear docs/epics/ recursivamente, leer frontmatter de cada T*.md, contar tasks completados, calcular ratios por Story/Feature/Epic, y renderizar un arbol ASCII con simbolos especiales. `rootline tree docs/epics/ --output table` hace exactamente esto en un solo comando.

## Dependencias

- Ninguna (rootline tree ya funciona correctamente)

## Alcance

**In**:
1. Reemplazar las instrucciones manuales en la seccion `/roadmap` o `/roadmap view` del SKILL.md
2. Nuevo contenido: instruccion de ejecutar `rootline tree docs/epics/ --output table`
3. Mantener que es read-only

**Out**: Cambios al comando rootline tree, cambios a otras secciones del SKILL.md

## Estado inicial esperado

- `.claude/skills/roadmap/SKILL.md` existe con seccion `/roadmap view` conteniendo instrucciones manuales de escaneo y renderizado

## Criterios de Aceptacion

- Seccion `/roadmap view` del SKILL.md contiene `rootline tree docs/epics/` como comando
- No existen instrucciones manuales de "Escanear", "Contar tasks", "Renderizar arbol"
- La seccion es significativamente mas corta que la actual (~40 lineas → ~5 lineas)

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md` — archivo a modificar (seccion `/roadmap` o `/roadmap view`)
