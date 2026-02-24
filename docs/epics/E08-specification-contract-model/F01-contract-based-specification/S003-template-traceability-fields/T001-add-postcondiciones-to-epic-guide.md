---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Agregar Postcondiciones/Invariantes a epic-guide.md

**Story**: [S003 Template Traceability Fields](README.md)
**Contribuye a**: epic-guide.md tiene Postcondiciones/Invariantes/OutOfScope en Epic template y Satisface/Invariantes en Feature template

## Contexto

Los templates de Epic y Feature en epic-guide.md no tienen postcondiciones formales ni invariantes. El Epic template tiene Intencion pero no condiciones observables de "done". El Feature template tiene Milestone pero no declara a que postcondicion del Epic contribuye.

## Alcance

**In**:
1. Epic README template: agregar seccion `## Postcondiciones` (tabla ID/Condicion/Features) despues de Intencion
2. Epic README template: agregar seccion `## Invariantes` (lista de propiedades que ningun feature puede violar)
3. Epic README template: agregar seccion `## Out of Scope` (lista de lo que el Epic NO cubre)
4. Feature README template: agregar linea `**Satisface**: P1, P2 — del [Epic](../README.md)` despues de Milestone
5. Feature README template: agregar seccion `## Invariantes` (heredados del Epic + propios)

**Out**: No modificar la seccion "Workflow" ni "Cuándo Crear un Epic".

## Preserva

- INV1: Las secciones existentes de epic-guide.md no se eliminan
- Verificar: diff solo muestra adiciones a los templates

## Estado inicial esperado

- epic-guide.md tiene templates para Epic README y Feature README
- Epic template tiene: estado, tipo, titulo, Metrica, Timeline, Intencion, Features, Orden, Decision Log, Gaps
- Feature template tiene: estado, tipo, titulo, Epic, Objetivo, Beneficio, Milestone, Scope, Stories, Dependencias, Fuente

## Criterios de Aceptacion

- Epic template tiene `## Postcondiciones` con tabla ID/Condicion/Features
- Epic template tiene `## Invariantes` con lista
- Epic template tiene `## Out of Scope` con lista
- Feature template tiene linea `**Satisface**:` despues de Milestone
- Feature template tiene `## Invariantes`

## Fuente de verdad

- `.claude/skills/roadmap/epic-guide.md`
