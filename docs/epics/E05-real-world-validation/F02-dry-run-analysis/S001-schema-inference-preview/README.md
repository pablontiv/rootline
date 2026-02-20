# S001: Schema Inference Preview

**Feature**: [F02 Dry-Run Analysis](../README.md)
**Estado**: Completado
**Cliente**: Platform Owner
**Capacidad**: Rootline muestra lo que haria sin modificar nada, infiriendo schema de datos reales

## Antes / Despues

**Antes**: `init --dry-run` nunca se ejecuto contra datos con 3 campos reales. No hay evidencia de que la inferencia funcione fuera de fixtures.

**Despues**: rootline infiere correctamente estado/tipo/ejecutable_en, fix y new muestran previews coherentes sin crear archivos.

## Criterios de Aceptacion (semanticos)

- [x] `init --dry-run` muestra .stem inferido con los 3 campos, NO crea archivo
- [x] `fix --dry-run` sin .stem reporta nada que reparar, exit code 0
- [x] `new --dry-run` muestra documento con frontmatter, NO crea archivo

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-run-init-dry-run.md) | Ejecutar init --dry-run en directorio raiz y subdirectorio |
| [T002](T002-run-fix-dry-run.md) | Ejecutar fix --dry-run sin .stem |
| [T003](T003-run-new-dry-run.md) | Ejecutar new --dry-run en story existente |
