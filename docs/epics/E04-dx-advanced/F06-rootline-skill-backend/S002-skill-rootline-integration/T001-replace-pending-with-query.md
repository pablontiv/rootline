---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Reemplazar /roadmap pending con rootline query

**Story**: [S002 Skill Rootline Integration](README.md)

## Contexto

La seccion `/roadmap pending` del SKILL.md contiene un script Python de 60 lineas que parsea frontmatter manualmente, agrupa por Epic/Feature, y renderiza una tabla. Este script tiene un path hardcodeado a `/opt/homeserver/automation` (de otro proyecto) y reimplementa logica que rootline query ya provee: escanear archivos, filtrar por frontmatter, y mostrar resultados en tabla.

## Dependencias

- S001/T001 (fix query `in` operator) debe estar completado para que `--where "estado in Pending,Especificado"` funcione

## Alcance

**In**:
1. Reemplazar el bloque Python en la seccion `/roadmap pending` del SKILL.md
2. Nuevo contenido: instruccion de ejecutar `rootline query docs/epics/ --where "estado in Pending,Especificado" --output table`
3. Mantener la instruccion "Presenta el output tal cual, sin modificaciones"

**Out**: Cambios al comando rootline query, cambios a otras secciones del SKILL.md

## Estado inicial esperado

- `.claude/skills/roadmap/SKILL.md` existe con seccion `/roadmap pending` conteniendo script Python
- rootline query `in` operator funciona (S001/T001 completado)

## Criterios de Aceptacion

- Seccion `/roadmap pending` del SKILL.md contiene `rootline query docs/epics/` como comando, no Python
- No existe referencia a `/opt/homeserver/automation` en SKILL.md
- No existe bloque `python` en la seccion pending
- La instruccion dice presentar output tal cual

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md` — archivo a modificar (seccion `/roadmap pending`)
