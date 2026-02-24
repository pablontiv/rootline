---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T002: Agregar Cubre/Invariantes a story-guide.md

**Story**: [S003 Template Traceability Fields](README.md)
**Contribuye a**: story-guide.md tiene Cubre + Invariantes con comandos de verificacion

## Contexto

El Story template tiene Antes/Despues y Criterios de Aceptacion semanticos pero no declara que parte del milestone del Feature cubre. Tampoco tiene invariantes — propiedades que todos los tasks deben preservar.

## Alcance

**In**:
1. Story template: agregar linea `**Cubre**: [que parte del milestone del Feature]` despues de Capacidad
2. Story template: agregar seccion `## Invariantes` con nota explicativa y ejemplo (ej: "coverage > 85%", "all existing tests pass")
3. Nota en seccion ACs: "Cada criterio debe trazar al milestone declarado en Cubre"
4. Agregar nota en seccion "Notas" sobre el campo Cubre y trazabilidad

**Out**: No modificar la seccion "Workflow" ni "Guia para Antes/Despues".

## Preserva

- INV1: Las secciones existentes de story-guide.md no se eliminan
- Verificar: diff solo muestra adiciones

## Estado inicial esperado

- story-guide.md tiene template con: titulo, Feature, Capacidad, Antes/Despues, ACs, Tasks, Fuente
- Seccion Notas tiene 4 bullet points

## Criterios de Aceptacion

- Story template tiene linea `**Cubre**:` despues de Capacidad
- Story template tiene seccion `## Invariantes` con nota y ejemplo
- Seccion ACs tiene nota sobre trazabilidad a Cubre
- Seccion Notas menciona el campo Cubre

## Fuente de verdad

- `.claude/skills/roadmap/story-guide.md`
