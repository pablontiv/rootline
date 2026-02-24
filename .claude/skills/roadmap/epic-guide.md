# Epic Guide — Descomponer Intención Estratégica

## Cuándo Crear un Epic

Un Epic es válido cuando cumple TODAS estas condiciones:

1. **Objetivo único**: Persigue un solo objetivo sistémico. Si necesitás
   explicar dos capacidades independientes, son dos epics.

2. **Nombre específico**: Se puede describir en 2-4 palabras concretas
   (ej: "Derivation Engine", "Release Pipeline"). Si requiere un nombre
   genérico ("Advanced Capabilities", "DX Improvements") → no tiene
   intención clara.

3. **Features conectadas**: Todas sus features contribuyen a la misma
   métrica de éxito. Si completar F01 no acerca el epic a "done" en
   el mismo sentido que F02 → no son del mismo epic.

4. **Criterios de éxito escribibles**: Podés escribir 2-3 criterios
   de aceptación a nivel epic que engloben todas las features.
   Si no podés → el scope es incoherente.

## Señales de Splitting

Un epic debe partirse cuando:

- El nombre es vago o usa catch-all words (Advanced, Misc, Enhancements)
- Features pertenecen a dominios técnicos distintos sin dependencias cruzadas
- No existe un "done" unificado — cada feature tiene su propio "done" independiente
- Stakeholder no puede explicar qué cambia cuando el epic está "completo"

> **Tip**: Usa `rootline tree docs/epics/<epic>/ --where "estado != 'Completed'" -o table` para ver solo el trabajo pendiente de un epic, sin necesidad de query + filtrado manual.

## Tamaño

- **Target**: 3-5 Features con substancia
- **Máximo**: 7 Features — más allá de esto, dividir
- **Mínimo**: 2 Features — si tiene 1, probablemente es un Feature dentro de otro Epic

## Workflow

### Paso 1: Parsear Argumentos

De `$ARGUMENTS`, extraer:
- **epic-name**: slug kebab-case (ej: `disaster-recovery`)
- **description**: intención estratégica en una línea

### Paso 2: Auto-numbering

```bash
# Detectar próximo número de Epic (requiere .stem con id: {type: sequence, prefix: E, digits: 2})
rootline describe docs/epics/ --field schema.id.next
```

El comando retorna directamente el próximo identificador (ej: `"E05"`). Requiere que `docs/epics/.stem` tenga `id: {type: sequence}` configurado.

### Paso 3: Discovery

Antes de crear, verificar:
1. Leer READMEs de epics existentes → ¿hay overlap con la intención propuesta?
2. Buscar documentación relacionada → `rootline query docs/ --where "..."` con keywords relevantes
3. Verificar roadmaps existentes en `docs/epics/`

Si hay overlap significativo → informar al usuario antes de crear.

### Paso 4: Descomposición

Descomponer la intención en Features. Cada Feature debe ser:
- Un **milestone técnico real** (no subdivisión artificial)
- Cerrable de forma independiente
- Coherente con el objetivo del Epic

**Principio anti-inflación**: Preferir 3-5 Features con substancia que 10 Features granulares. Si una Feature tiene una sola Story, probablemente no debería ser Feature.

Para cada Feature, identificar 1-3 Stories iniciales (placeholders).

### Paso 5: Generar en Plan File

Presentar la descomposición completa ANTES de crear archivos:

```
Epic: EXX-name
├── Intención: [una línea]
├── Métrica de éxito: [medible]
├── Features:
│   ├── FXX-feature-1: [milestone]
│   │   └── Stories: S001-name, S002-name
│   └── FXX-feature-2: [milestone]
│       └── Stories: S001-name
└── Decision Log: (vacío, se llena iterativamente)
```

### Paso 6: Crear Estructura (post-aprobación)

Crear directorios y archivos:

```
docs/epics/EXX-name/
├── README.md                    ← Epic README
├── FXX-feature-1/
│   └── README.md                ← Feature README
└── FXX-feature-2/
    └── README.md                ← Feature README
```

Las Stories se crean como placeholders en las tablas de Features, no como directorios (se materializan con `/roadmap story`).

---

## Template: Epic README

```markdown
---
estado: In Progress
tipo: feature
---
# EXX: [Nombre del Epic]

**Metrica de exito**: [métrica medible]
**Timeline**: YYYY-QX — en curso

## Intencion

[Párrafo describiendo el objetivo sistémico. Qué problema resuelve a nivel estratégico.]

## Postcondiciones

- P1: [constraint observable y verificable]
- P2: [constraint observable y verificable]

## Invariantes

- INV1: [regla que ningún Feature puede violar]

## Out of Scope

- [límite explícito]

## Features

| ID | Nombre | Descripcion |
|----|--------|-------------|
| FXX | [Feature Name](FXX-name/) | [una línea] |

## Orden de Ejecucion

| Feature | Depende de | Razon |
|---------|-----------|-------|
| FXX | — | Foundation (sin dependencias) |
| FXX | FXX | [razón de la dependencia] |

## Decision Log

| Fecha | Decision | Razon |
|-------|----------|-------|

## Gaps Activos

- [Gaps identificados durante la descomposición]
```

## Template: Feature README

```markdown
---
estado: Pending
tipo: feature
---
# FXX: [Nombre del Feature]

**Epic**: [EXX](../README.md)
**Satisface**: P1, P2
**Objetivo**: [Qué capacidad nueva tiene el sistema cuando esto está completo]
**Beneficio**: [Qué problema resuelve o qué habilita para el sistema/usuario]
**Milestone**: [Condición medible de "done" — estado observable, no entregable]

## Scope

**In**: [qué cubre este feature]
**Out**: [qué NO cubre]

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|

## Invariantes

- INV1 (heredado): [del Epic]
- INV2: [propio del Feature]

## Dependencias

- [Features que deben completarse antes de este]

## Fuente de verdad

- [paths relevantes]
```

**Nota sobre Objetivo vs Milestone**:
- **Objetivo** = capacidad funcional ("El sistema puede validar documentos contra schemas")
- **Beneficio** = valor entregado ("Elimina 7 parsers independientes con regex frágiles")
- **Milestone** = condición de "done" ("`rootline validate docs/` retorna JSON con errores")

No confundir deliverable con objetivo: "Task guide incluye tipos" es un deliverable, no un objetivo.
