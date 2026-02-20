# E05: Real-World Validation

**Estado**: Completado
**Metrica de exito**: Todos los comandos de rootline ejecutan sin panic contra 114 archivos del homeserver, producen output coherente, y el ciclo init→validate→fix completa sin dejar cambios residuales
**Timeline**: 2026-Q1

## Intencion

Probar que Rootline funciona contra datos no controlados. El homeserver tiene 114 archivos markdown con jerarquia Epic→Feature→Story→Task donde solo los Tasks tienen YAML frontmatter (`estado`, `tipo`, `ejecutable_en`). No tiene `.stem` files.

Este epic es independiente de E03 (build) y E04 (DX). Su objetivo sistémico es distinto: validar que el engine funciona contra datos externos y descubrir edge cases reales.

## Features

| ID | Nombre | Descripcion |
|----|--------|-------------|
| F01 | [Read-Only Exploration](F01-read-only-exploration/) | stats, tree, describe, query y validate ejecutan sin modificar datos |
| F02 | [Dry-Run Analysis](F02-dry-run-analysis/) | init, fix y new previsualizan cambios sin tocar disco |
| F03 | [Write Operations](F03-write-operations/) | Ciclo completo init→validate→fix contra datos externos con limpieza |

## Orden de Ejecucion

| Feature | Depende de | Razon |
|---------|-----------|-------|
| F01 | Binary compilado | Read-only, riesgo minimo |
| F02 | F01 | Requiere confianza en que read-only funciona antes de previsualizar escritura |
| F03 | F02 | Cruza boundary de escritura a disco, requiere confirmacion explicita |

## Target

- **Binary**: `/opt/rootline/rootline` (compilado, 4.9MB)
- **Dataset**: `/opt/homeserver/automation/docs/epics/` (114 archivos markdown, 6 epics, jerarquia 4 niveles)

## Verificacion end-to-end

Despues de cada task:
1. Exit code 0 (salvo errores esperados)
2. JSON output tiene `"version": 1`
3. Sin panics ni stack traces
4. Datos coherentes con la realidad

Limpieza final: `git status` en homeserver confirma 0 cambios residuales.

## Referencias

- [E03: Rootline](../E03-rootline/) — epic de construccion
- [Intent Document v0](../../rootline-intent-v0.md) — vision y decisiones
