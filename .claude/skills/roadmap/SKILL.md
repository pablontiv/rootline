---
source: pablontiv/praxis
name: roadmap
description: |
  Usar cuando el usuario sabe QUÉ construir y necesita planificar CÓMO —
  descomponiendo trabajo en Outcomes opcionales y Tasks ejecutables por agentes,
  con criterios de aceptación, dependencias y validación. También para ver
  progreso, trabajo pendiente o ejecutar tasks en secuencia. Usar si el usuario
  describe capacidades a construir, pregunta "cómo estructuro esto", lista
  requerimientos, quiere ver pendientes/progreso, o dice "next task",
  "planificar" o "descomponer".
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
effort: xhigh
execution-model: sonnet
worktree-per-outcome: false
parallel-independent-tasks: false
hooks:
  Stop:
    - type: agent
      prompt: "Verify that the critic agent was invoked during this skill execution and its evaluation passed. If no critic evaluation occurred and the skill produced artifacts, return {ok: false, reason: 'Critic evaluation was skipped'}. If work is still in progress, return {ok: true}."
      timeout: 60
---

# /roadmap — Planificación AI-Native Simple

Modelo canónico:

```text
Outcome/Objetivo  (opcional)
└── Task          (unidad ejecutable)
```

Para trabajo chico, usar solo tasks. El skill produce únicamente Outcomes y Tasks.

## Invariante de materialización

Cuando el usuario pide crear, generar o materializar tareas, el resultado NO puede
ser un único archivo resumen tipo `*-tasks.md`.

Materializar tareas significa crear archivos canónicos:

```text
<roadmap-root>/OXX-slug/README.md
<roadmap-root>/OXX-slug/TXXX-task.md
```

o tasks directas:

```text
<roadmap-root>/TXXX-task.md
```

Si no se puede crear esa estructura, detenerse y explicar el bloqueo. No hacer
fallback a markdown libre.

## Bootstrap obligatorio

Ejecutar SIEMPRE antes de cualquier operación.

### Paso 0: Detectar modo

```bash
test -d .git
```

- Sí → single-repo mode.
- No → workspace mode.

### Workspace mode

1. Si existe `.claude/roadmap.local.md` en cwd con `mode: workspace`, leer `repos:` como base.
2. Escanear subdirectorios inmediatos con `.git` + `.claude/roadmap.local.md`.
3. Para cada repo, leer `roadmap-root` y calcular:
   - `<repo-path>`
   - `<abs-roadmap-root>`
   - helpers `<where-leaf>`, `<where-not-done>`, `<where-active>`
4. Imprimir checkpoint con repos detectados.

### Single-repo mode

1. Leer `.claude/roadmap.local.md`.
2. Si no existe, preguntar dónde vive el roadmap y crearlo.
3. Extraer `roadmap-root`; si falta, preguntar y actualizar.
4. Extraer config operacional; si falta, usar defaults.
5. Pre-computar helpers.

Template mínimo:

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
outcome-close-verify: []
pr-merge-strategy: 'squash'
commit-style: 'conventional'
auto-push: true
---
```

## Configuración

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
| `leaf-filter` | `'isIndex == false'` | `<where-leaf>` |
| `outcome-close-verify` | `[]` | `<outcome-close-cmds>` |
| `pr-merge-strategy` | `'squash'` | `<pr-merge-strategy>` |
| `commit-style` | `'conventional'` | `<commit-style>` |
| `auto-push` | `true` | `<auto-push>` |

Helpers:

- `<where-not-done>`: `not (estado in <done-statuses>)`
- `<where-active>`: `estado in <active-statuses>`
- `<where-leaf>`: valor de `leaf-filter`

Checkpoint obligatorio:

```text
Bootstrap:
  roadmap-root: docs/roadmap
  <where-leaf>:     isIndex == false
  <where-not-done>: not (estado in ["Completed", "Obsolete"])
  <where-active>:   estado in ["Pending", "Specified", "In Progress"]
```

## Validación de configuración

Si `rootline` existe y `.claude/.stem` existe:

```bash
rootline validate .claude/roadmap.local.md
```

Si `<roadmap-root>` existe:

```bash
rootline describe <roadmap-root>/ --field schema.estado
```

Verificar que los status configurados existan en el schema.

## Dependencia: rootline CLI

Requerido para materializar, consultar y ejecutar roadmaps.

Gate antes de ejecutar rootline:

```bash
command -v rootline
```

Si no está disponible, informar:

```text
`rootline` no está instalado. Es requerido para materializar/consultar el roadmap.
Instalar con: curl -fsSL https://raw.githubusercontent.com/pablontiv/rootline/master/install.sh | bash
```

La generación conceptual de planes puede continuar sin rootline.

## Routing por subcomando

Después del bootstrap:

| `$ARGUMENTS` | Archivo | Descripción |
|--------------|---------|-------------|
| `pending` | [pending-subcommand.md](pending-subcommand.md) | Trabajo pendiente |
| `plan` | [plan-subcommand.md](plan-subcommand.md) | Materializar plan como `.md` |
| *(sin argumentos)* | [decision-tree-subcommand.md](decision-tree-subcommand.md) | Priorizar qué ejecutar |
| `loop [--filter] [--max] [--pr]` | [loop-subcommand.md](loop-subcommand.md) | Ejecutar tasks pendientes |
| *(texto libre)* | [autonomous-mode.md](autonomous-mode.md) | Descomponer en Outcome/Tasks |

## Flag global `--repo`

Solo workspace mode:

- `--repo <name>` resuelve un único repo y remueve el flag antes del dispatch.
- En single-repo mode se ignora.

## Regla de dispatch

1. Si empieza con `pending`, `loop`, `plan` → subcomando directo.
2. Si vacío → decision tree.
3. Si pide estado/progreso/pendientes → `pending`.
4. Si dice "crea las tareas", "materializa", "genera los archivos",
   "pasalo al roadmap", "crea el roadmap" o equivalente → `plan`.
5. Si describe algo a construir o descomponer sin pedir archivos → modo autónomo.

Ambigüedad crítica:

- "descompón/planifica" = proponer estructura, no escribir archivos.
- "crea/materializa/genera tareas" = `plan-subcommand.md`.
- Si no hay plan previo suficiente para materializar, preguntar antes de escribir.

## Lógica común

Leer [common-logic.md](common-logic.md) cuando se crean/modifican archivos del roadmap o se ejecuta loop.

## Referencia

- Modelo completo: [framework-reference.md](framework-reference.md)
- Outcomes: [outcome-guide.md](outcome-guide.md)
- Tasks: [task-guide.md](task-guide.md)
- `.stem` base: [base.stem](base.stem)
