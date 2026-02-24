---
estado: Specified
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Agregar Paso 2.5 "Formalizar Contratos" a SKILL.md

**Story**: [S002 Workflow Integration](README.md)
**Contribuye a**: SKILL.md tiene Paso 2.5 "Formalizar Contratos" entre Paso 2 y Paso 3

## Contexto

El modo autonomo del skill salta de "Absorber Contexto" (Paso 2) a "Aplicar Framework" (Paso 3) sin formalizar restricciones. Esto permite descomposicion prematura — Features/Stories/Tasks que no trazan a restricciones observables del Epic.

## Alcance

**In**:
1. Insertar Paso 2.5 "Formalizar Contratos" entre Paso 2 y Paso 3 en la seccion "Modo Autonomo"
2. Contenido: Para cada Epic identificado, ANTES de descomponer, escribir postcondiciones (constraints observables), invariantes (propiedades que no pueden romperse), y out of scope
3. Incluir formato de output para el plan file con constraint map (postcondicion → Features)
4. Incluir validacion bidireccional: toda postcondicion tiene Feature, todo Feature satisface >= 1 postcondicion

**Out**: No modificar otros pasos. No cambiar subcomandos.

## Preserva

- INV1: Los subcomandos (pending, loop, plan) siguen funcionando sin cambios
- Verificar: seccion de subcomandos no fue modificada

## Estado inicial esperado

- SKILL.md tiene Paso 2 (Absorber Contexto) y Paso 3 (Aplicar Framework) consecutivos
- No existe Paso 2.5

## Criterios de Aceptacion

- SKILL.md tiene seccion "### Paso 2.5: Formalizar Contratos" entre Paso 2 y Paso 3
- Paso 2.5 menciona: postcondiciones, invariantes, out of scope
- Paso 2.5 incluye formato de constraint map en plan file
- Paso 2.5 incluye validacion bidireccional

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md`
