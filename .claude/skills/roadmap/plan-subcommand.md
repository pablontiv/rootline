# /roadmap plan

> Pre-requisito: leer [common-logic.md](common-logic.md).

Materializa el plan de la conversación como archivos `.md` del roadmap. No implementa código.

Materializar es una operación estructural. Está prohibido crear un único archivo
con una lista de tareas. Cada task debe tener su propio archivo `TXXX-*.md`.

## Fuente del plan

1. Contexto actual de conversación.
2. Fallback: `~/.claude/plans/${CLAUDE_SESSION_ID}.md`.

Si no hay plan, informar: “No hay plan en esta conversación. Primero planificar, luego ejecutar `/roadmap plan`.” y parar.

## Workspace mode

Resolver repo target:

1. `--repo <name>` si fue dado.
2. Repo mencionado en el plan.
3. Si ambiguo, preguntar.

Usar `<abs-roadmap-root>` y `git -C <repo-path>`.

## Fase 1: Descomposición

1. Identificar el plan más reciente.
2. Leer contexto existente relacionado bajo `<roadmap-root>/`.
3. Aplicar [framework-reference.md](framework-reference.md): máximo Outcome + Tasks.
4. Producir:
   - tasks directas, o
   - Outcome(s) + tasks.
5. Cada task debe tener nombre, descripción, dependencias `blocked_by` con paths relativos explícitos y ACs principales.

## Fase 2: Aprobación

Presentar árbol completo y pedir aprobación con `AskUserQuestion`.

STOP hasta aprobación. No crear archivos antes.

## Fase 3: Materialización

**MATERIALIZAR ≠ IMPLEMENTAR.** Crear solo archivos `.md` y `.stem` dentro de `<roadmap-root>/`.

Guardrail obligatorio antes de escribir:

1. Confirmar que se va a crear una de estas formas:
   - Outcome + tasks: `OXX-slug/README.md` + `OXX-slug/TXXX-*.md`
   - Tasks directas: `TXXX-*.md` en la raíz del roadmap.
2. Si el plan contiene varias tasks, no escribirlas en un archivo único.
3. Si no hay información suficiente para nombrar/separar tasks, preguntar.
4. Si falta `rootline` y no se puede crear estructura canónica, detenerse.

### Paso 1: Bootstrap `.stem` base

Si `<roadmap-root>/` no existe, crear el directorio.

Si `<roadmap-root>/.stem` no existe, copiar el template canónico [base.stem](base.stem) como `<roadmap-root>/.stem`.

Contenido de referencia:

```yaml
version: 2
scope:
  match: "*.md"

schema:
  estado:
    type: enum
    required:
      match: ["O*", "T*"]
    match: ["O*", "T*"]
    values: [Pending, Specified, In Progress, Completed, Blocked, On Hold, Obsolete]

  tipo:
    type: enum
    required:
      match: ["O*", "T*"]
    match: ["O*", "T*"]
    values: [outcome, task]

  id:
    type: sequence
    match:
      "O*": { prefix: O, digits: 2 }
      "T*": { prefix: T, digits: 3 }

links:
  blocked_by:
    target: '^(\./|\.\./|.*/)T[0-9]{3}-[^/]+\.md$'
  reference:
    target: ".*"

validate:
  - field: estado
    rule: non_empty
  - field: tipo
    rule: non_empty
```

No crear `.stem` por subnivel salvo necesidad excepcional del proyecto.

### Paso 2: Crear Outcomes

Para cada Outcome:

```bash
find <roadmap-root>/ -maxdepth 1 -type d -name 'O[0-9][0-9]-*' -printf '%f\n' | sort
# tomar el mayor OXX y sumar 1; si no hay ninguno, usar O01
mkdir -p <roadmap-root>/OXX-slug
rootline new <roadmap-root>/OXX-slug/README.md
```

Editar el README usando [outcome-guide.md](outcome-guide.md).

### Paso 3: Crear Tasks

Tasks dentro de Outcome:

```bash
rootline describe <roadmap-root>/OXX-slug/ --field schema.id.next
rootline new <roadmap-root>/OXX-slug/TXXX-task.md
```

Tasks directas:

```bash
find <roadmap-root>/ -maxdepth 1 -type f -name 'T[0-9][0-9][0-9]-*.md' -printf '%f\n' | sort
# tomar el mayor TXXX y sumar 1; si no hay ninguno, usar T001
rootline new <roadmap-root>/TXXX-task.md
```

Editar cada task usando [task-guide.md](task-guide.md). Si una task depende de otra, escribir el link con path relativo explícito: `[[blocked_by:./T001-name.md]]` dentro del mismo Outcome o `[[blocked_by:../O01-name/T001-name.md]]` entre Outcomes.

Validación anti-regresión:

Antes de continuar, comprobar que cada task del plan corresponde a un archivo
`TXXX-*.md`. Si no, corregir antes de responder.

### Paso 4: Cascading links

Si la task pertenece a un Outcome, agregarla en la tabla `## Tasks` del README del Outcome.

### Paso 5: Validar

Después de cada write:

```bash
rootline validate <path>
```

Al final:

```bash
rootline validate --all <roadmap-root>/
rootline graph <roadmap-root>/ --check
```

Si hay errores corregibles, usar `rootline fix` con criterio conservador. Un warning `scope.match "*.md" matches no files in directory` es aceptable cuando la raíz solo contiene directorios `OXX-*` y ninguna task directa.

### Paso 6: Commit + push

- `git add` solo archivos `.md` y `.stem` creados/modificados del roadmap.
- `git commit -m "chore(roadmap): create planning docs"`
- `git push` si `<auto-push>` es true.

STOP obligatorio. Informar: “Archivos de planificación creados. Ejecutar `/roadmap loop` cuando esté listo para implementar.”
