# F02: Dry-Run Analysis

**Epic**: [E05](../README.md)
**Objetivo**: Rootline infiere schemas y previsualiza cambios sin tocar disco
**Beneficio**: Valida el pipeline de inferencia y preview contra datos reales antes de permitir escritura
**Milestone**: `init --dry-run` infiere los 3 campos reales (estado, tipo, ejecutable_en) correctamente

## Scope

**In**: init --dry-run, fix --dry-run, new --dry-run
**Out**: Escritura real a disco, generacion de archivos

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Schema Inference Preview](S001-schema-inference-preview/) | rootline muestra lo que haria sin modificar nada |

## Dependencias

- F01 completado (confianza en read-only)
