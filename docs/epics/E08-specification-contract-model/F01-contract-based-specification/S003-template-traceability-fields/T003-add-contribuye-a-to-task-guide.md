---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T003: Agregar Contribuye_a/Preserva a task-guide.md

**Story**: [S003 Template Traceability Fields](README.md)
**Contribuye a**: task-guide.md tiene Contribuye_a + Preserva + 7ma condicion en checklist

## Contexto

El Task template tiene ACs binarios y Estado inicial pero no declara a que criterio de Story contribuye ni que invariantes de la Story debe preservar. El checklist tiene 6 condiciones sin trazabilidad.

## Alcance

**In**:
1. Task template: agregar linea `**Contribuye a**: [criterio de la Story que este Task satisface]` despues de Story link
2. Task template: agregar seccion `## Preserva` con nota: "Invariantes de la Story que este task no puede violar. Se verifican automaticamente en /roadmap loop despues de los ACs."
3. Nota en seccion ACs: "Cada AC debe contribuir al criterio declarado en Contribuye a"
4. Agregar 7ma condicion al checklist: `| 7 | Trazabilidad | ¿Declara "Contribuye a" y "Preserva"? ¿Sus ACs avanzan el criterio de la Story? |`

**Out**: No modificar tipos ni templates de Especificacion Tecnica (eso es F02).

## Preserva

- INV1: Las 6 condiciones existentes del checklist no se modifican
- Verificar: condiciones 1-6 identicas a version anterior

## Estado inicial esperado

- task-guide.md tiene template con: frontmatter, titulo, Story, Contexto, Spec Tecnica, Dependencias, Alcance, Estado inicial, ACs, Fuente
- Checklist tiene 6 condiciones (Sesion unica, Sin memoria, Criterios binarios, Verificable, Idempotente, I/O explicitos)

## Criterios de Aceptacion

- Task template tiene linea `**Contribuye a**:` despues de Story link
- Task template tiene seccion `## Preserva` con nota explicativa
- Seccion ACs tiene nota sobre trazabilidad
- Checklist tiene 7 filas, la 7ma es "Trazabilidad"

## Fuente de verdad

- `.claude/skills/roadmap/task-guide.md`
