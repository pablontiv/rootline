---
estado: Pending
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Update .stem enum, derive, aggregate, and hold field

**Story**: [S001 Schema & Data Migration](README.md)

## Contexto

El archivo `docs/epics/.stem` define el enum de `estado` con 6 valores mezclando espanol e ingles (Completado, Bloqueada, Diferida). Las expresiones derive y aggregate referencian estos valores. El aggregate tiene un fallback a `estado` (valor actual del frontmatter) que retorna `<nil>` cuando el campo no existe. No hay campo `hold` para bloqueo manual por usuario.

## Alcance

**In**:
1. Reemplazar enum values: `Completado` → `Completed`, `Bloqueada` → `Blocked`, `Diferida` → eliminar, agregar `On Hold`
2. Reescribir derive expression: agregar `hold != nil ? "On Hold"` como primera rama, cambiar `"Completado"` → `"Completed"`, `"Bloqueada"` → `"Blocked"`
3. Reescribir aggregate expression: agregar ramas para `On Hold` y `Specified`, cambiar `"Completado"` → `"Completed"`, `"Bloqueada"` → `"Blocked"`, cambiar fallback de `estado` a `"Pending"`
4. Agregar `hold: { type: string }` al schema

**Out**: Migracion de frontmatter (T002), cambios a codigo Go, cambios a skills

## Estado inicial esperado

- `docs/epics/.stem` con enum: `[Pending, In Progress, Specified, Completado, Diferida, Bloqueada]`
- Derive: `blocked_by != nil && ... ? "Bloqueada" : estado`
- Aggregate: fallback es `estado`

## Criterios de Aceptacion

- `docs/epics/.stem` enum contiene exactamente: `[Pending, Specified, In Progress, Completed, Blocked, On Hold]`
- Derive expression tiene rama `hold != nil ? "On Hold"` como primera condicion
- Derive expression usa `"Completed"` y `"Blocked"` (no espanol)
- Aggregate expression tiene ramas para Completed, Blocked, On Hold, In Progress, Specified
- Aggregate fallback es `"Pending"` (no `estado`)
- Schema tiene campo `hold` con `type: string`

## Fuente de verdad

- `docs/epics/.stem`
