---
source: pablontiv/praxis
name: roadmap
description: |
  Usar cuando el usuario sabe QUÉ construir y necesita planificar CÓMO —
  descomponiendo un proyecto en epics, features, stories y tasks con specs,
  criterios de aceptación y dependencias. También para ver progreso, trabajo
  pendiente, o ejecutar tareas en secuencia. Usar este skill siempre que el
  usuario describa features a construir, pregunte "cómo estructuro este
  proyecto", liste requerimientos o componentes, quiera ver progreso, o diga
  "que falta" / "pendientes" / "next task" / "planificar" / "descomponer" —
  incluso si no dice "roadmap" ni "decompose", e incluso si solo dice
  "necesito construir X" sin pedir explícitamente un plan.
  (No para: evaluar SI algo vale la pena = hypothesize, explorar ideas = discover.)
argument-hint: "<texto libre> | [pending|loop|plan] [args]"
allowed-tools:
  - Write
  - Read
  - Grep
  - Glob
  - Bash
  - TaskCreate
  - TaskList
  - TaskUpdate
  - TaskGet
  - Skill
  - AskUserQuestion
  - ExitPlanMode
  - Agent
hooks:
  Stop:
    - type: agent
      prompt: "Verify that the critic agent was invoked during this skill execution and its evaluation passed. If no critic evaluation occurred and the skill produced artifacts, return {ok: false, reason: 'Critic evaluation was skipped'}. If work is still in progress, return {ok: true}."
      timeout: 60
---

**Evaluates against**: .claude/rules/doc-conventions.md

# /roadmap — Framework de Planificación AI-Native

## Configuración del Proyecto — Bootstrap Obligatorio

**PRIMER PASO de CUALQUIER operación** (pending, loop, plan, sin argumentos, modo autónomo). Ejecutar SIEMPRE, ANTES de cualquier otra acción — incluyendo antes del gate check de rootline. Este paso NO requiere rootline.

### Paso 0: Detección de Modo

1. `test -d .git` en el directorio actual
   - **SI** → **single-repo mode**. Continuar a Paso 1.
   - **NO** → **workspace mode**. Continuar a Paso 0.1.

### Paso 0.1: Workspace Discovery

El workspace mode permite coordinar múltiples repos desde un directorio padre (ej: `/opt/` con backscroll, rootline, forge como subdirectorios). Cada repo mantiene su propio `roadmap.local.md` y `<roadmap-root>` — el workspace los agrega.

1. Si existe `.claude/roadmap.local.md` en cwd con `mode: workspace` en frontmatter:
   - Leer `repos:` map como base (para repos con paths atípicos, ej: `.git` nested)
2. Escanear subdirectorios inmediatos buscando `.git` + `.claude/roadmap.local.md`:
   ```bash
   for d in */; do
     test -d "$d/.git" && test -f "$d/.claude/roadmap.local.md" && echo "$d"
   done
   ```
3. Para cada repo encontrado (auto-discovered + workspace config):
   - Leer su `.claude/roadmap.local.md` → extraer `roadmap-root`
   - Computar `abs-roadmap-root` = `<cwd>/<repo-path>/<roadmap-root>`
   - Computar sus propios helpers (`<where-not-done>`, `<where-active>`, `<where-leaf>`)
     usando la config de ese repo (o defaults si faltan)
4. Construir tabla `<repos>` = `[{name, repo-path, abs-roadmap-root, config}]`
5. Imprimir checkpoint workspace:

```
Bootstrap (workspace mode):
  repos detectados: N
  ┌─────────────┬───────────────────────────────┬──────────┐
  │ Repo        │ roadmap-root                  │ Source   │
  ├─────────────┼───────────────────────────────┼──────────┤
  │ backscroll  │ /opt/backscroll/docs/epics    │ auto     │
  │ rootline    │ /opt/rootline/docs/epics      │ auto     │
  │ homeserver  │ /opt/homeserver/auto.../epics │ ws-cfg   │
  └─────────────┴───────────────────────────────┴──────────┘
```

Después del checkpoint, continuar al **Routing por Subcomando** con `<repos>` disponible.

### Paso 1: Single-repo Bootstrap (sin cambios si viene de Paso 0)

1. Leer `.claude/roadmap.local.md`. Si no existe:
   - Preguntar al usuario: ¿Dónde vive el roadmap? (ej: `docs/epics`)
   - Crear `.claude/roadmap.local.md` con el frontmatter mínimo (ver template abajo)
2. Extraer `roadmap-root` del frontmatter. Si falta → preguntar y actualizar el archivo.
3. Extraer filtros y valores operacionales (ver tablas). Si faltan → usar defaults, NO preguntar.
4. Pre-computar expresiones helper (una vez, reusar en todos los comandos).

**Template mínimo** (crear si no existe):

```yaml
---
roadmap-root: # preguntar al usuario
done-statuses: ['Completed', 'Obsolete']
active-statuses: ['Pending', 'Specified', 'In Progress']
status-values:
  pending: 'Pending'
  specified: 'Specified'
  in-progress: 'In Progress'
  completed: 'Completed'
  blocked: 'Blocked'
  obsolete: 'Obsolete'
leaf-filter: 'isIndex == false'
story-close-verify: []
pr-merge-strategy: 'squash'
commit-style: 'conventional'
auto-push: true
---
```

En todo este documento, `<roadmap-root>` se refiere al valor configurado.

### Configuración de Filtros

| Config key | Default | Placeholder |
|------------|---------|-------------|
| `done-statuses` | `['Completed', 'Obsolete']` | `<done-statuses>` |
| `active-statuses` | `['Pending', 'Specified', 'In Progress']` | `<active-statuses>` |
| `status-values.pending` | `'Pending'` | `<status-pending>` |
| `status-values.specified` | `'Specified'` | `<status-specified>` |
| `status-values.in-progress` | `'In Progress'` | `<status-in-progress>` |
| `status-values.completed` | `'Completed'` | `<status-completed>` |
| `status-values.blocked` | `'Blocked'` | `<status-blocked>` |
| `status-values.obsolete` | `'Obsolete'` | `<status-obsolete>` |
| `leaf-filter` | `'isIndex == false'` | `<leaf-filter>` |
| `story-close-verify` | `[]` | `<story-close-cmds>` |
| `pr-merge-strategy` | `'squash'` | `<pr-merge-strategy>` |
| `commit-style` | `'conventional'` | `<commit-style>` |
| `auto-push` | `true` | `<auto-push>` |

Semántica:
- `done-statuses` y `active-statuses` son **predicados de lectura** para queries y dependencias.
- `status-values.*` son **valores canónicos de escritura**. Al mutar frontmatter, NUNCA hardcodear `Completed`/`Blocked`; usar `<status-completed>`, `<status-blocked>`, etc.

Expresiones helper (pre-computar una vez, reusar en todos los comandos):

- `<where-not-done>`: `not (estado in <done-statuses>)`
- `<where-active>`: `estado in <active-statuses>`
- `<where-leaf>`: `<leaf-filter>`

**Checkpoint obligatorio**: Imprimir los helpers computados antes de ejecutar cualquier query. Ejemplo para un proyecto con `done-statuses: ['Completed', 'Obsolete']`:

```
Bootstrap:
  roadmap-root: docs/epics
  <where-leaf>:     isIndex == false
  <where-not-done>: not (estado in ["Completed", "Obsolete"])
  <where-active>:   estado in ["Pending", "Specified", "In Progress"]
```

**Query de referencia** (con helpers ya sustituidos):

```bash
rootline tree docs/epics/ --where 'isIndex == false && not (estado in ["Completed", "Obsolete"])' --output table
```

**Anti-patrón**: Ejecutar `rootline tree/query/stats` sin incluir `isIndex == false` en `--where`. Sin este filtro, los resultados mezclan index files (READMEs de Epic, Feature, Story) con tasks reales, inflando conteos. Toda query que reporta trabajo pendiente o progreso debe incluir `<where-leaf>`.

---

## Dependencia: rootline CLI

**Requerida para materialización y queries** (`pending`, `loop`, `plan` post-aprobación). NO requerida para generar planes de descomposición.

Instalación:
```bash
curl -fsSL https://raw.githubusercontent.com/pablontiv/rootline/master/install.sh | bash
```

**Gate check**: Antes de ejecutar comandos `rootline` (crear archivos, queries, validación), verificar:
```bash
command -v rootline
```
Si no está disponible → informar al usuario:
> `rootline` no está instalado. Es requerido para materializar el roadmap.
> Instalar con: `curl -fsSL https://raw.githubusercontent.com/pablontiv/rootline/master/install.sh | bash`

**Alcance del gate**: Solo bloquea operaciones que ejecutan rootline (scaffolding, validate, query, tree, graph). La generación de planes de descomposición (modo autónomo, Paso 1-6) NO requiere rootline y puede proceder normalmente.

---

## Modo de Operación

Este skill es **plan-mode aware**. Cuando `defaultMode: "plan"` está activo:

### Fase 1: Planificación (automática en plan mode)
1. Parsear `$ARGUMENTS` para determinar subcomando
2. Leer el guide file correspondiente
3. Ejecutar discovery y generar contenido completo en el plan file
4. Llamar `ExitPlanMode` para aprobación

### Fase 2: Post-aprobación

Después de que el usuario aprueba el plan, informarle que puede ejecutar `/roadmap plan` para crear los archivos del roadmap.

---

## Routing por Subcomando

Una vez completado el bootstrap, determinar el modo según `$ARGUMENTS` y leer el archivo correspondiente:

| $ARGUMENTS | Archivo | Descripción |
|------------|---------|-------------|
| `pending` | [pending-subcommand.md](pending-subcommand.md) | Vista jerárquica de trabajo pendiente |
| `plan` | [plan-subcommand.md](plan-subcommand.md) | Materializar plan de conversación en archivos .md |
| *(sin argumentos)* | [decision-tree-subcommand.md](decision-tree-subcommand.md) | Árbol de decisión para priorizar ramas |
| `loop [--filter] [--max] [--pr]` | [loop-subcommand.md](loop-subcommand.md) | Ejecutar tasks pendientes en loop con confirmación |
| *(texto libre)* | [autonomous-mode.md](autonomous-mode.md) | Descomposición autónoma de proyecto |

**Flag global `--repo`** (solo workspace mode):
- Si `$ARGUMENTS` contiene `--repo <name>`: resolver a un solo repo de `<repos>`,
  establecer sus variables (`<roadmap-root>`, helpers) como si fuera single-repo,
  y proceder normalmente. Remover `--repo <name>` de `$ARGUMENTS` antes del dispatch.
- Ejemplo: `/roadmap pending --repo backscroll` → pending filtrado a backscroll.
- En single-repo mode, `--repo` se ignora silenciosamente.

**Regla de dispatch**:
1. Si `$ARGUMENTS` empieza con `pending`, `loop`, o `plan` → subcomando directo.
2. Si está vacío → decision tree.
3. Si pide **ver estado, progreso, o resumen** (ej: "ver pendientes", "status", "overview", "que falta", "ver roadmaps") → tratar como `pending`.
4. Si describe **features a construir** o pide descomposición → modo autónomo.

---

## Lógica Común (materialización y ejecución)

→ Leer [common-logic.md](common-logic.md) cuando se crean/modifican archivos del roadmap (`plan`, `loop`).
Contiene: auto-numbering, verificación de padre, cascading links, comandos rootline de referencia.

## Referencia

- Ver [framework-reference.md](framework-reference.md) para el documento completo del marco de trabajo
- Templates canónicos: primer Epic materializado en `<roadmap-root>/`
