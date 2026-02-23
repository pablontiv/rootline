---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T002: Actualizar CLAUDE.md y skills con --where

**Story**: [S002 Docs & Skills Alignment](README.md)

## Contexto

CLAUDE.md y los skills del proyecto usan rootline CLI como herramienta primaria. Con `--where` disponible en todos los comandos transversales, estos archivos deben actualizarse para que el AI assistant use los comandos correctos. El caso mas impactante es `/roadmap pending` que hoy usa un workaround de 5 pasos con 3 comandos, y puede simplificarse a `rootline tree --where "estado != 'Completed'" -o table`.

## Alcance

**In**:
1. `CLAUDE.md` seccion "Rootline as Primary Interface": agregar mencion de `--where` disponible en tree, stats, graph, validate
2. `.claude/skills/roadmap/SKILL.md`:
   - Simplificar `/roadmap pending` de 5 pasos a 2: `rootline tree --where` + `rootline stats`
   - Actualizar tabla de referencia de comandos rootline (agregar --where a tree, stats, graph)
3. `.claude/skills/roadmap/epic-guide.md`: actualizar ejemplos que usan `rootline query` para mencionar `tree --where` como alternativa

**Out**: README.md y docs/ (T001), cambios a codigo

## Estado inicial esperado

- S001 completado (--where funcional en todos los comandos)
- CLAUDE.md sin mencion de --where en tree/stats/graph
- SKILL.md /roadmap pending con workaround multi-comando

## Criterios de Aceptacion

- `grep "where" CLAUDE.md` muestra mencion de --where en comandos transversales
- `/roadmap pending` en SKILL.md tiene maximo 2 pasos (tree --where + stats)
- Tabla de referencia en SKILL.md muestra --where para tree, stats, graph
- `grep "where" .claude/skills/roadmap/epic-guide.md` muestra referencia

## Fuente de verdad

- `CLAUDE.md`
- `.claude/skills/roadmap/SKILL.md`
- `.claude/skills/roadmap/epic-guide.md`
