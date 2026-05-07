# /roadmap plan

> **Pre-requisito**: Leer [common-logic.md](common-logic.md) para auto-numbering, cascading links y comandos rootline.

Tomar el plan de la conversacion actual y descomponerlo en estructura de roadmap.

**Cuando usar**: Despues de que una sesion produce un plan tecnico (investigacion, analisis, fix propuesto, etc.) y se quiere estructurar como roadmap.

**Fuente del plan** (orden de prioridad):
1. Contexto de la conversacion actual (preferido)
2. Fallback: plan file de `~/.claude/plans/${CLAUDE_SESSION_ID}.md`
   (plan files son globales — usar `${CLAUDE_SESSION_ID}` para garantizar que sea de esta sesion)

## Workspace mode

En workspace mode, determinar el repo target antes de descomponer:
1. Si `--repo` fue dado → usar ese repo
2. Si el plan en conversación menciona un nombre que matchea un repo en `<repos>` → usar ese
3. Si ambiguo → `AskUserQuestion`: "En qué repo se materializa este plan?"

Una vez resuelto, usar `<abs-roadmap-root>` y `<repo-path>` de ese repo.
Paso 7 (Commit+Push): `git -C <repo-path> add/commit/push`.

## Fase 1: Descomposicion

1. Identificar el plan mas reciente:
   a. Buscar en el contexto de conversacion actual (plan tecnico, analisis, propuesta, etc.)
   b. Si la conversacion fue compactada o no tiene plan visible → leer plan file de
      `~/.claude/plans/${CLAUDE_SESSION_ID}.md`
   c. Si no hay plan en ninguna fuente → informar: "No hay plan en esta conversacion.
      Primero investigar/planificar, luego ejecutar `/roadmap plan`." → STOP
3. Absorber contexto: leer documentacion existente del area afectada en `<roadmap-root>/`
4. Aplicar framework de descomposicion (mismos criterios que Modo Autonomo):
   - Leer [framework-reference.md](framework-reference.md)
   - Criterios de escala: 3-5 Features/Epic, 1-4 Stories/Feature, 1-5 Tasks/Story
   - Constraint Map (postcondiciones + invariantes)
   - Validacion de completitud (Paso 4.5 del Modo Autonomo en [autonomous-mode.md](autonomous-mode.md))
5. **OBLIGATORIO**: Descomposicion DEBE llegar hasta nivel Task para TODOS los Stories.
   Cada Task con: nombre, tipo, descripcion de 1 linea.

## Fase 2: Presentacion para aprobacion

6. Presentar arbol jerarquico completo + Constraint Map
7. Pedir aprobacion con `AskUserQuestion` (NO usar `ExitPlanMode` — ese es para plan mode
   del sistema, no para aprobaciones internas de un skill)
8. **STOP y esperar aprobacion. NO crear archivos sin aprobacion.**

## Fase 3: Materializacion (post-aprobacion)

**MATERIALIZAR ≠ IMPLEMENTAR.** En esta fase se crean SOLAMENTE archivos `.md` del roadmap
dentro de `<roadmap-root>/` (Feature READMEs, Story READMEs, Task .md files).
Estos archivos DESCRIBEN el trabajo a realizar — NO lo ejecutan.
NUNCA escribir codigo, scripts, configs, hooks, ni ningun archivo fuera de `<roadmap-root>/`.
La implementacion ocurre despues via `/roadmap loop`.

### Paso 1: Bootstrap .stem raiz

Si `<roadmap-root>/` no tiene `.stem`:
- **Con archivos .md existentes** → `rootline init <roadmap-root>/` infiere el schema
- **Greenfield (sin archivos)** → crear un unico `.stem` raiz con el schema completo
  (id sequence, tipo enum, estado, aggregate, derive, links, validate).
  Usar como referencia un proyecto rootline existente del ecosistema del usuario.

El `.stem` raiz es el **unico** que se crea manualmente. Los `.stem` en subdirectorios
(E→F→S→T) solo redefinen `id.prefix` para el nivel inferior, con contenido minimo:

```yaml
version: 2
schema:
    id:
        type: sequence
        prefix: X  # F, S, o T segun nivel
        digits: N  # 2 para F, 3 para S y T
```

### Paso 2: Descubrir schema con rootline describe

Antes de crear archivos en un directorio, ejecutar:

```bash
rootline describe <directorio>/
```

Esto muestra el schema efectivo (merged de `.stem` raiz + padres), incluyendo campos
requeridos, tipos, y `schema.id.next` (proximo ID disponible). Usar esta informacion
para llenar el frontmatter correctamente — no adivinar campos ni valores.

### Paso 3: Scaffolding con rootline new

Para cada artefacto (README de Epic/Feature/Story, archivo de Task):

```bash
rootline new <path-al-archivo.md>
```

Esto genera el frontmatter correcto segun el `.stem` efectivo del directorio destino.
Luego editar el contenido (body del .md) con los templates de:
- [epic-guide.md](epic-guide.md) para READMEs de Epic y Feature
- [story-guide.md](story-guide.md) para READMEs de Story
- [task-guide.md](task-guide.md) para archivos de Task

### Paso 4: Validar

Despues de cada Write: `rootline validate <path>`. Si falla → `rootline fix <path>`.

### Paso 5: Cascading links

Actualizar tablas en READMEs padre (ver seccion "Cascading Links" en SKILL.md).

### Paso 6: Validacion batch final

```bash
rootline validate --all <roadmap-root>/
```
Si hay errores → `rootline fix --all <roadmap-root>/`. Reportar resultado al usuario.

### Paso 7: Commit+Push

- `git add` archivos .md y .stem creados (especificos, no `git add .`)
- `git commit` con mensaje: `chore(roadmap): create {descripcion breve} planning docs`
- `git push`

**STOP OBLIGATORIO**: Despues de commit+push, DETENERSE COMPLETAMENTE.
Informar: "Archivos de planificacion creados. Ejecutar `/roadmap loop` cuando este listo
para implementar."
NO continuar. NO invocar `/roadmap loop`. NO leer tasks para implementar.
NO escribir archivos de implementacion (codigo, scripts, configs, hooks, units, etc.).
