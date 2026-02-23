---
estado: Pending
tipo: feature
---
# E06: Cross-Document State

**Estado**: Pending
**Metrica de exito**: `rootline query --where "estado == 'Pending'"` retorna solo tasks genuinamente accionables, filtrando automaticamente los bloqueados por dependencias
**Timeline**: 2026-Q1

## Intencion

Habilitar derivacion de estado entre documentos conectados via wiki-links. Hoy rootline deriva campos dentro de la jerarquia filesystem (aggregate), pero no sigue links tipados entre documentos. El campo `LinkRule.Field` se parsea del .stem pero nunca se consume. Esta epic lo activa: si A `[[blocks:B]]` y B tiene estado Completado, el derive engine expone ese valor para que A pueda calcular si esta bloqueada.

## Features

| Feature | Descripcion |
|---------|-------------|
| [F01 Link Field Resolution](F01-link-field-resolution/) | Resolver valores de campos de documentos enlazados y propagar estado via links |

## Orden de Ejecucion

| Feature | Depende de | Razon |
|---------|-----------|-------|
| F01 | — | Sin dependencias externas (usa derive + graph existentes) |

## Decision Log

| Fecha | Decision | Razon |
|-------|----------|-------|
| 2026-02-22 | LinkRule.Field define nombre de variable en derive env | Consistente con la intencion original del campo en rules.go |
| 2026-02-22 | Valores enlazados como slice (no scalar) | Un documento puede tener multiples links del mismo tipo |

## Gaps Activos

- Orden de evaluacion: si A depende de B y B depende de C, derive debe procesar C antes de B antes de A (topological sort)
- Ciclos: si A blocks B y B blocks A, necesita deteccion y error graceful
