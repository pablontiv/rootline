# Outcome Guide — Crear Objetivo de Roadmap

## Cuándo crear un Outcome

Crear un Outcome solo si agrupa una intención común y reduce complejidad:

- más de 5 tasks relacionadas,
- un resultado observable que requiere varias tasks,
- invariantes compartidas,
- necesidad de tracking visual por objetivo.

Si el trabajo cabe en 1–5 tasks auto-contenidas, crear tasks directas bajo `<roadmap-root>/`.

## Auto-numbering

```bash
find <roadmap-root>/ -maxdepth 1 -type d -name 'O[0-9][0-9]-*' -printf '%f\n' | sort
```

Tomar el mayor `OXX` y sumar 1; si no hay ninguno, usar `O01`.

## Estructura

```text
<roadmap-root>/
└── O01-nombre-del-objetivo/
    ├── README.md
    └── T001-task.md
```

## Template: Outcome README

```markdown
---
estado: Pending
tipo: outcome
---
# OXX: [Nombre del objetivo]

## Objetivo

[Resultado observable que existirá cuando todas las tasks estén completadas.]

## Criterios de Éxito

- CE1: [criterio verificable]
  - Verificar: [comando o procedimiento]
- CE2: [criterio verificable]
  - Verificar: [comando o procedimiento]

## Invariantes

- INV1: [propiedad que ninguna task debe romper]
  - Verificar: [comando o procedimiento]

## Alcance

**In**:
- [incluido]

**Out**:
- [excluido]

## Tasks

| Task | Descripción |
|------|-------------|
| [T001](T001-ejemplo.md) | Descripción breve |
```

## Reglas

- La tabla de tasks no incluye estado; el estado vive en el frontmatter de cada task.
- No crear subniveles bajo Outcome.
- Las dependencias entre tasks se declaran con `[[blocked_by:./TXXX-name.md]]` en la task bloqueada.
