# F01: Read-Only Exploration

**Epic**: [E05](../README.md)
**Objetivo**: Rootline lee, presenta y consulta datos externos sin modificar nada
**Beneficio**: Valida que extraction, indexing, query y presentation funcionan contra datos no controlados
**Milestone**: Todos los comandos read-only ejecutan sin panic y producen output coherente contra 114 archivos

## Scope

**In**: stats, tree, describe, query, validate (sin .stem), doctor
**Out**: Escritura a disco, generacion de .stem, modificacion de archivos

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Data Exploration](S001-data-exploration/) | rootline muestra stats, tree y schema de 114 archivos sin errores |
| S002 | [Query & Validation](S002-query-validation/) | rootline consulta y valida datos externos coherentemente |

## Dependencias

- Binary rootline compilado y funcional
- Acceso de lectura a `/opt/homeserver/automation/docs/epics/`
