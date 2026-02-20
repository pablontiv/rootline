---
estado: Pending
tipo: documentation
ejecutable_en: 1 sesion
---
# T003: Agregar rootline stats a /roadmap view

**Story**: [S003 Auto-numbering Integration](README.md)

## Contexto

`/roadmap view` actualmente solo ejecuta `rootline tree docs/epics/ --output table`. El comando `rootline stats` provee un resumen de conteos por estado y tipo que complementa la vista jerarquica del tree. En el proyecto homeserver-automation, `rootline stats` tiene 419 referencias — es un comando de uso frecuente para entender el estado del proyecto de un vistazo.

## Alcance

**In**:
1. En SKILL.md seccion `/roadmap view`, agregar llamada a `rootline stats docs/epics/` antes o despues del tree
2. Agregar seccion "Comandos Rootline de Referencia" al final de SKILL.md con tabla de 7 comandos y sus contextos de uso

**Out**: Cambios a otros subcomandos, cambios a como se ejecuta el tree

## Estado inicial esperado

- SKILL.md existe con seccion `/roadmap view` que solo llama a rootline tree
- `rootline stats` funciona y produce output JSON/table con conteos por estado

## Criterios de Aceptacion

- Seccion `/roadmap view` del SKILL.md incluye `rootline stats docs/epics/` como comando adicional
- SKILL.md contiene tabla "Comandos Rootline de Referencia" con: validate, fix, describe --field schema.id.next, new, query --where, tree, stats
- Cada entrada de la tabla especifica cuando usarlo en el flujo del skill

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md` — seccion `/roadmap view` y final del archivo
- `cmd/rootline/stats.go` — sintaxis del comando stats
