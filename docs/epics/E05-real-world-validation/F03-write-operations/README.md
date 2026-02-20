# F03: Write Operations

**Epic**: [E05](../README.md)
**Objetivo**: Rootline ejecuta el ciclo completo init→validate→fix contra datos externos
**Beneficio**: Valida que el write path funciona end-to-end y que la limpieza es reversible
**Milestone**: Ciclo init→validate→fix→cleanup ejecuta sin dejar cambios residuales en el homeserver

## Scope

**In**: init (real), validate con .stem, fix (real), limpieza con rm + git checkout
**Out**: Cambios permanentes al homeserver

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Schema Write & Repair](S001-schema-write-repair/) | rootline genera .stem y valida/repara con el |

## Dependencias

- F02 completado (confianza en dry-run)
- Confirmacion explicita del usuario antes de escribir
