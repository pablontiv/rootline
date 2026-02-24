---
estado: Pending
tipo: historia
---
# S001: Framework Contract Definitions

**Feature**: [F01 Contract-Based Specification](../README.md)
**Capacidad**: El framework de planificacion define contratos formales (Pre/Post/Invariantes/Trazabilidad) por nivel
**Cubre**: Milestone de F01 — "framework-reference.md tiene seccion 2.3 + contratos en 4.1-4.4"

## Antes / Despues

**Antes**: framework-reference.md define niveles (Epic/Feature/Story/Task) con responsabilidades pero sin contratos formales. No hay modelo de Pre/Post/Invariantes. La validacion de completitud (Paso 4.5) es informal.

**Despues**: Seccion 2.3 establece el principio "Especificar antes de descomponer". Cada nivel (4.1-4.4) tiene contratos formales. Seccion 12.1 documenta la cadena de trazabilidad bidireccional.

## Invariantes

> Propiedades que TODOS los tasks de esta story deben preservar.

- INV1: El contenido existente de framework-reference.md no se elimina ni modifica
- INV2: `rootline validate docs/epics/` sigue pasando sin errores

## Criterios de Aceptacion (semanticos)

- [ ] framework-reference.md tiene seccion 2.3 con modelo Pre/Post/Invariantes/Trazabilidad
- [ ] Secciones 4.1-4.4 tienen bloques de contratos formales
- [ ] Existe seccion 12.1 con cadena de trazabilidad bidireccional

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-add-contracts-to-framework.md) | Agregar seccion 2.3 + extender 4.1-4.4 + agregar 12.1 |

## Fuente de verdad

- `.claude/skills/roadmap/framework-reference.md`
