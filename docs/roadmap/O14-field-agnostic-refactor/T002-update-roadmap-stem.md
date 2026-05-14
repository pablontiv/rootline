---
estado: Completed
tipo: task
---
# T002: Update roadmap `.stem` with `titulo`, `is_done`, and `value_field`

**Outcome**: [O14 Field-agnostic refactor](README.md)
**Contribuye a**: el stem del roadmap usa capacidades nuevas del engine y sirve como test de integración vivo

## Preserva

- INV1: `rootline validate docs/roadmap/.stem` exit 0 antes y después
  - Verificar: `rootline validate /home/shared/rootline/docs/roadmap/.stem`
- INV2: `roadmapctl check` sigue pasando después del cambio
  - Verificar: `roadmapctl check --repo /home/shared/rootline --roadmap-root docs/roadmap --output json --strict`

## Contexto

El stem en `docs/roadmap/.stem` (rootline repo) es el schema que gobierna los archivos del roadmap. Con T001 implementado, podemos agregar `titulo` con `source: body.h1` para que el campo se derive automáticamente del H1 de cada task/outcome. También agregamos `is_done` como campo derivado bool y `value_field: estado` en el link `blocked_by`.

El plan especifica exactamente qué agregar — ver sección "Cambios en el roadmap .stem" del plan fuente.

## Alcance

**In**:
1. Agregar campo `titulo: {type: string, source: body.h1}` al schema
2. Agregar campo `is_done: {type: bool, severity: off}` al schema
3. Agregar `derive: is_done: "estado in ['Completed', 'Obsolete']"` (o equivalente YAML)
4. Agregar `value_field: estado` al link `blocked_by`

**Out**:
- No cambiar valores de `estado`, `tipo`, `id` ni reglas de validate existentes
- No actualizar otros stems fuera de `docs/roadmap/.stem`

## Estado inicial esperado

- T001 completada (engine soporta `source: body.h1`)
- `rootline validate docs/roadmap/.stem` exit 0

## Criterios de Aceptación

- `rootline validate /home/shared/rootline/docs/roadmap/.stem` exit 0
- `rootline query /home/shared/rootline/docs/roadmap --select path,titulo --output json` retorna campo `titulo` con el texto del H1 para cada task/outcome
- `roadmapctl check --repo /home/shared/rootline --roadmap-root docs/roadmap --output json --strict` exit 0

## Fuente de verdad

- `/home/shared/rootline/docs/roadmap/.stem`
