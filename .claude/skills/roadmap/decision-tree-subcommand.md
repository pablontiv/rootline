# /roadmap (sin argumentos) — Arbol de Decision

Generar **arbol de decision** que muestre ramas ejecutables, cadenas de dependencia, y bloqueos para decidir que loop implementar.

## Workspace mode

Si `<repos>` existe (workspace mode): ejecutar Paso 1 **por cada repo** en `<repos>`.
Los comandos son independientes entre repos — lanzar todos en paralelo (N×3 comandos).
En Paso 4, agrupar ramas por repo con prefijo `[repo-name]`.
En Paso 5, agregar criterio: "Focalizarme en un solo proyecto? → `/roadmap --repo <name>`".

## Paso 1: Recopilar datos (3 comandos en paralelo, por repo)

Para cada repo (o el unico repo en single-repo mode), ejecutar en paralelo:
1. `rootline tree <abs-roadmap-root>/ --where "<where-leaf> and <where-not-done>" --output json` — arbol jerarquico con paths, estados y conteos completed/total (~2 KB, reemplaza stats + query)
2. `rootline graph <abs-roadmap-root>/ --where "<where-leaf> and <where-not-done>" --output json` — grafo de dependencias entre pendientes (~3 KB)
3. `git -C <repo-path> log -5 --format='%h %s'` — ultimos commits para proximidad (en single-repo: `git log` sin `-C`)
4. (Opcional) Si `command -v backscroll >/dev/null 2>&1`: `backscroll search "blocked" --robot --max-tokens 1000` — sesiones previas pueden explicar por qué tasks fueron bloqueadas o diferidas

**IMPORTANTE**: Despues de Paso 1, NO ejecutar mas comandos bash. Los Pasos 2-5 procesan los JSONs obtenidos.

## Paso 2: Agrupar en ramas (procesamiento de datos, SIN comandos adicionales)

Usar los outputs JSON de Paso 1 para construir las ramas:

1. **Feature path**: Extraer de `root.children[].children[].path` del tree JSON (cmd 1) — la jerarquia ya agrupa por Epic/Feature/Story/Task
2. **Dependencias intra-story**: Extraer de `edges[]` del graph JSON (cmd 2) — cada edge tiene `source`, `target`, y `type: "blocks"`
3. **Conteos**: Cada nodo del tree tiene `completed` y `total` — usar para progreso por rama
4. **Estado**: Cada hoja del tree tiene `estado` — usar para clasificar tasks

NO ejecutar comandos adicionales. Todo se extrae de los 3 outputs del Paso 1.

## Paso 3: Clasificar ramas (procesamiento de datos, SIN comandos adicionales)

Usando `estado` de cada hoja del tree JSON (cmd 1):

- **Ejecutables**: todas las tasks tienen estado en `<active-statuses>`, sin dependencias insatisfechas (verificar contra `edges[]` del graph, cmd 2)
- **Bloqueadas**: al menos una task tiene `estado: Blocked` o dependencia cross-feature con estado no en `<done-statuses>`

Dentro de ejecutables, identificar **quick wins** (ramas con 1 solo task).

## Anti-patrones de eficiencia

- Loops `for f in ...; do grep/head; done` — usar JSON de rootline
- Queries adicionales post-Paso 1 — toda la data necesaria esta en los 3 outputs
- Usar `rootline query` para listados — `rootline tree` da estructura + estados + conteos en un solo comando
- Usar `rootline stats` por separado — tree ya incluye completed/total por nodo
- Buscar `[[blocks:]]` con grep — rootline graph ya parsea wiki-links
- Maximo 3 comandos (Paso 1), todos en paralelo, ~5.5 KB total, el resto es procesamiento de datos

## Paso 4: Renderizar arbol de decision

Formato de salida (workspace mode usa prefijo `[repo]` en cada rama):

```
ROADMAP DECISION TREE — N/M completados (X%)

Que objetivo priorizar?
│
├─► [backscroll] RAMA: Feature Name (Epic) — N tasks, tipo dominante
│   │
│   T001: nombre                    [estado, tipo]
│   │   ↓ desbloquea
│   T002: nombre                    [estado, tipo]
│       ↓ CIERRA [que capacidad]
│
├─► [rootline] RAMA: ...
│
└─► QUICK WINS (cross-repo)
    [backscroll] T045: nombre       [estado, tipo]
    [rootline]   T012: nombre       [estado, tipo]

BLOQUEADAS SIN CAMINO DIRECTO
│
├── [backscroll] TXXX: nombre    [blocker: descripcion]
└── [rootline]   TXXX: nombre    [blocker: descripcion]
```

En single-repo mode, omitir el prefijo `[repo]` (formato original).

Reglas de renderizado:
- Usar `├─►` para ramas ejecutables, `├──` para bloqueadas
- `↓ desbloquea` entre tasks con dependencia `[[blocks:]]`
- `↓ CIERRA [capacidad]` en el ultimo task de la rama (extraer del nombre del nodo Feature en el tree JSON)
- Marcar tasks cuya dependencia ya esta en `<done-statuses>` pero siguen en Blocked como `[stale?]`
- Ordenar ramas ejecutables por proximidad al ultimo commit (extraer de `git log` del Paso 1)

## Paso 5: Renderizar criterios de decision

Al final del arbol, agregar flowchart de decision:

```
CRITERIOS DE DECISION
│
├─ Quiero focalizarme en un solo proyecto?            (solo workspace mode)
│  ├─ SI → /roadmap --repo <name>
│  └─ NO ↓
├─ Hay rama en progreso (ultimo commit)?
│  ├─ SI → Cerrar esa rama primero
│  └─ NO ↓
├─ Hay deuda tecnica que bloquea futuro trabajo?
│  ├─ SI → Rama que desbloquea mas dependientes
│  └─ NO ↓
├─ Quiero progreso rapido?
│  ├─ SI → Quick win o rama mas corta
│  └─ NO → Rama con mayor impacto arquitectural
```

Adaptar el flowchart al estado real (referenciar ramas concretas en cada hoja).
En single-repo mode, omitir el primer criterio.
