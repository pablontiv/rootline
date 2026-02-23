---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Actualizar README.md y docs/ con --where en comandos transversales

**Story**: [S002 Docs & Skills Alignment](README.md)

## Contexto

La documentacion CLI en `README.md` tiene una tabla de comandos que solo lista `--where` para `query`. Con la implementacion de F13/S001, tree, stats, validate, y graph tambien soportan `--where`. La documentacion debe reflejar esto para que usuarios y AI assistants descubran la funcionalidad.

## Alcance

**In**:
1. `README.md` tabla de CLI (seccion de comandos): agregar `--where "expr"` a tree, stats, graph, validate
2. `README.md`: agregar ejemplos de uso filtrado con --where en tree/stats/graph
3. `docs/query.md`: agregar nota cross-reference indicando que `--where` esta disponible en todos los comandos transversales (query, tree, stats, graph, validate --all)
4. `docs/graph.md`: agregar seccion de filtrado con `--where` y ejemplo

**Out**: CLAUDE.md (T002), skills (T002), cambios a codigo

## Estado inicial esperado

- S001 completado (--where funcional en todos los comandos)
- README.md con tabla de CLI que solo menciona --where en query

## Criterios de Aceptacion

- `grep -c "where" README.md` retorna >= 5 (tabla + ejemplos)
- `grep "where" docs/query.md` muestra cross-reference a tree, stats, graph, validate
- `grep "where" docs/graph.md` muestra seccion de filtrado
- Tabla de CLI en README muestra --where para tree, stats, graph, validate ademas de query

## Fuente de verdad

- `README.md`
- `docs/query.md`
- `docs/graph.md`
