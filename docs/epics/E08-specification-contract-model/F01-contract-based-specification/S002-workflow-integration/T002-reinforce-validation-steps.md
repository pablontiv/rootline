---
estado: Specified
tipo: documentation
ejecutable_en: 1 sesion
---
# T002: Reforzar Pasos 3/4/4.5 con contratos formales

**Story**: [S002 Workflow Integration](README.md)
**Contribuye a**: Paso 4.5 tiene checks basados en contratos (no informales)

[[blocks:T001-add-paso-2-5-to-skill]]

## Contexto

Paso 2.5 crea postcondiciones e invariantes, pero los pasos posteriores no los usan. Paso 3 no requiere que Features satisfagan postcondiciones. Paso 4 no muestra constraint map. Paso 4.5 tiene checks informales ("verificar mentalmente").

## Alcance

**In**:
1. Modificar Paso 3 tabla: agregar "satisface >= 1 postcondicion del Epic" al criterio de Feature
2. Modificar Paso 4: agregar Constraint Map table al output del plan (tabla postcondicion → Features → cobertura)
3. Reforzar Paso 4.5 item 2: reemplazar "Completeness descendente" con "Completeness por contratos" — cada postcondicion del Epic tiene >= 1 Feature, cada milestone de Feature tiene >= 1 Story, cada criterio de Story tiene >= 1 Task AC
4. Agregar Paso 4.5 item 6: "Invariant propagation check" — invariantes del Epic aparecen en Features, fluyen a Stories, Tasks los preservan

**Out**: No agregar nuevos pasos completos. No modificar subcomandos.

## Preserva

- INV1: El flujo Paso 1→2→2.5→3→4→4.5→5→6 es coherente
- Verificar: leer SKILL.md secuencialmente, los pasos fluyen logicamente

## Estado inicial esperado

- Paso 2.5 ya existe (T001 completado)
- Paso 3 tiene tabla con criterios por nivel
- Paso 4.5 tiene 5 items de validacion

## Criterios de Aceptacion

- Paso 3 tabla fila Feature incluye "satisface >= 1 postcondicion del Epic"
- Paso 4 tiene bloque de Constraint Map con tabla
- Paso 4.5 item 2 dice "Completeness por contratos" (no "descendente")
- Paso 4.5 tiene item 6 "Invariant propagation check"

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md`
