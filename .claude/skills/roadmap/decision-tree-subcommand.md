# /roadmap — Árbol de Decisión

Generar un árbol de decisión que muestre Outcomes ejecutables, tasks directas, dependencias y bloqueos.

## Workspace mode

Si `<repos>` existe, ejecutar la recopilación por repo y agrupar salida con prefijo `[repo]`. Si el usuario quiere focalizar, sugerir `/roadmap --repo <name>`.

## Paso 1: Recopilar datos

Por repo, ejecutar en paralelo:

1. `rootline tree <roadmap-root>/ --where "<where-leaf> && <where-not-done>" --output json`
2. `rootline graph <roadmap-root>/ --where "<where-leaf> && <where-not-done>" --output json`
3. `git log -5 --name-only --format='%h %s'`
4. Opcional: `backscroll search "blocked" --robot --max-tokens 1000`

Después de este paso, no ejecutar más comandos; procesar los JSON obtenidos.

## Paso 2: Construir ramas

- Ramas = Outcomes con pending > 0 + tasks directas pendientes.
- Dependencias = edges del graph con `type == "blocked_by"`; usar los `target` ya resueltos por `rootline graph`, no el texto crudo del wikilink.
- Estado = frontmatter de cada hoja del tree.

## Paso 3: Clasificar

- **Ejecutable**: task activa sin dependencias insatisfechas.
- **Bloqueada**: estado `Blocked` o alguna dependencia no está en `<done-statuses>`.
- **Quick win**: rama con una sola task ejecutable.

Scoring sugerido:

```text
score = 0
      + 50 si el path aparece en git log reciente
      + 10 por cada task que desbloquea
      + 5  si contiene una task In Progress
      - 3  por cada task pendiente
      - 100 si tiene dependencia insatisfecha o estado Blocked
```

Ordenar por `score desc`, `pending_count asc`, `path asc`.

## Paso 4: Renderizar

```text
ROADMAP DECISION TREE — N/M completados

Qué objetivo priorizar?
│
├─► O01 Nombre del Outcome — N tasks pendientes
│   T001 nombre [estado]
│      ↓ desbloquea
│   T002 nombre [estado]
│
├─► TASK DIRECTA
│   T010 nombre [estado]
│
└─► QUICK WINS
    T011 nombre [estado]

BLOQUEADAS
└── T012 nombre [blocked_by: O01-setup/T003-name.md]
```

Reglas:

- Mostrar `↓ desbloquea` usando índice reverso de `blocked_by`.
- Marcar como `[stale?]` tasks en `Blocked` cuyas dependencias ya están completadas.
- No buscar links con grep; `rootline graph` es la fuente de verdad.

## Paso 5: Criterios de decisión

```text
CRITERIOS
├─ Hay trabajo In Progress? → cerrarlo primero
├─ Hay task que desbloquea muchas otras? → priorizarla
├─ Quiero progreso rápido? → quick win
└─ Si no → mayor score determinista
```
